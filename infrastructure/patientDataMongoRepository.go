package infrastructure

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	goComMgo "github.com/tidepool-org/go-common/clients/mongo"
	"github.com/tidepool-org/tide-whisperer/common"
	"github.com/tidepool-org/tide-whisperer/schema"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	dataCollectionName = "deviceData"
	idxUserIDTypeTime  = "UserIdTypeTimeWeighted"
)

var unwantedFields = bson.M{
	"_id":                0,
	"_userId":            0,
	"_groupId":           0,
	"_version":           0,
	"_active":            0,
	"_schemaVersion":     0,
	"createdTime":        0,
	"modifiedTime":       0,
	"conversionOffset":   0,
	"clockDriftOffset":   0,
	"timezoneOffset":     0,
	"deviceTime":         0,
	"deviceId":           0,
	"deviceSerialNumber": 0,
	"source":             0,
}

var wantedRangeFields = bson.M{
	"_id":  0,
	"time": 1,
}

var tideWhispererIndexes = map[string][]mongo.IndexModel{
	"deviceData": {
		{
			Keys: bson.D{{Key: "_userId", Value: 1}, {Key: "type", Value: 1}, {Key: "time", Value: -1}},
			Options: options.Index().
				SetName(idxUserIDTypeTime).
				SetWeights(bson.M{"_userId": 10, "type": 5, "time": 1}),
		},
	},
}

type PatientDataMongoRepository struct {
	*goComMgo.StoreClient
}

// InArray returns a boolean indicating the presence of a string value in a string array
func InArray(needle string, arr []string) bool {
	for _, n := range arr {
		if needle == n {
			return true
		}
	}
	return false
}

// NewPatientDataMongoRepository creates a new patientData repository for mongo
func NewPatientDataMongoRepository(config *goComMgo.Config, logger *log.Logger) (*PatientDataMongoRepository, error) {
	if config != nil {
		config.Indexes = tideWhispererIndexes
	}
	pdmr := PatientDataMongoRepository{}
	store, err := goComMgo.NewStoreClient(config, logger)
	pdmr.StoreClient = store
	return &pdmr, err
}

func dataCollection(p *PatientDataMongoRepository) *mongo.Collection {
	return p.Collection(dataCollectionName)
}

// generateMongoQuery takes in a number of parameters and constructs a mongo query
// to retrieve objects from the Tidepool database. It is used by the router.Add("GET", "/{userID}"
// endpoint, which implements the Tide-whisperer API. See that function for further documentation
// on parameters
func generateMongoQuery(p *common.Params) bson.M {

	finalQuery := bson.M{}
	skipParamsQuery := false
	groupDataQuery := bson.M{
		"_userId":        p.UserID,
		"_active":        true,
		"_schemaVersion": bson.M{"$gte": p.SchemaVersion.Minimum, "$lte": p.SchemaVersion.Maximum}}

	//if optional parameters are present, then add them to the query
	if len(p.Types) > 0 && p.Types[0] != "" {
		groupDataQuery["type"] = bson.M{"$in": p.Types}
		if !InArray("deviceEvent", p.Types) {
			skipParamsQuery = true
		}
	}

	if len(p.SubTypes) > 0 && p.SubTypes[0] != "" {
		groupDataQuery["subType"] = bson.M{"$in": p.SubTypes}
		if !InArray("deviceParameter", p.SubTypes) {
			skipParamsQuery = true
		}
	}

	if p.Date.Start != "" && p.Date.End != "" {
		groupDataQuery["time"] = bson.M{"$gte": p.Date.Start, "$lte": p.Date.End}
	} else if p.Date.Start != "" {
		groupDataQuery["time"] = bson.M{"$gte": p.Date.Start}
	} else if p.Date.End != "" {
		groupDataQuery["time"] = bson.M{"$lte": p.Date.End}
	}

	if !p.Carelink {
		groupDataQuery["source"] = bson.M{"$ne": "carelink"}
	}

	if p.DeviceID != "" {
		groupDataQuery["deviceId"] = p.DeviceID
		skipParamsQuery = true
	}

	// If we have an explicit upload ID to filter by, we don't need or want to apply any further
	// data source-based filtering
	if p.UploadID != "" {
		groupDataQuery["uploadId"] = p.UploadID
		finalQuery = groupDataQuery
	} else {
		andQuery := []bson.M{}
		if !p.Dexcom && p.DexcomDataSource != nil {
			dexcomQuery := []bson.M{
				{"type": bson.M{"$ne": "cbg"}},
				{"uploadId": bson.M{"$in": p.DexcomDataSource["dataSetIds"]}},
			}
			if earliestDataTime, ok := p.DexcomDataSource["earliestDataTime"].(time.Time); ok {
				dexcomQuery = append(dexcomQuery, bson.M{"time": bson.M{"$lt": earliestDataTime.Format(time.RFC3339)}})
			}
			if latestDataTime, ok := p.DexcomDataSource["latestDataTime"].(time.Time); ok {
				dexcomQuery = append(dexcomQuery, bson.M{"time": bson.M{"$gt": latestDataTime.Format(time.RFC3339)}})
			}
			andQuery = append(andQuery, bson.M{"$or": dexcomQuery})
		}

		if !p.Medtronic && len(p.MedtronicUploadIds) > 0 {
			medtronicQuery := []bson.M{
				{"time": bson.M{"$lt": p.MedtronicDate}},
				{"type": bson.M{"$nin": []string{"basal", "bolus", "cbg"}}},
				{"uploadId": bson.M{"$nin": p.MedtronicUploadIds}},
			}
			andQuery = append(andQuery, bson.M{"$or": medtronicQuery})
		}

		if len(andQuery) > 0 {
			groupDataQuery["$and"] = andQuery
			finalQuery = groupDataQuery
		} else if skipParamsQuery || len(p.LevelFilter) == 0 {
			finalQuery = groupDataQuery
		} else {
			paramQuery := []bson.M{}
			// create the level filter as string
			levelFilterAsString := []string{}
			for value := range p.LevelFilter {
				levelFilterAsString = append(levelFilterAsString, strconv.Itoa(value))
			}
			paramQuery = append(paramQuery, groupDataQuery)

			deviceParametersQuery := bson.M{}
			deviceParametersQuery["type"] = "deviceEvent"
			deviceParametersQuery["subType"] = "deviceParameter"
			deviceParametersQuery["level"] = bson.M{"$in": levelFilterAsString}
			otherDataQuery := bson.M{}
			otherDataQuery["subType"] = bson.M{"$ne": "deviceParameter"}

			orQuery := []bson.M{}
			orQuery = append(orQuery, deviceParametersQuery)
			orQuery = append(orQuery, otherDataQuery)

			paramQuery = append(paramQuery, bson.M{"$or": orQuery})
			finalQuery = bson.M{"$and": paramQuery}
		}
	}

	return finalQuery
}

// GetDataRangeLegacy returns the time data range
//
// If no data for the requested user, return nil or empty string dates
func (p *PatientDataMongoRepository) GetDataRangeLegacy(ctx context.Context, traceID string, userID string) (*common.Date, error) {

	dateRange := &common.Date{
		Start: "",
		End:   "",
	}
	var res map[string]interface{}

	query := bson.M{
		"_userId": userID,
		// Use only diabetes data, excluding upload & pumpSettings
		"type": bson.M{"$not": bson.M{"$in": []string{"upload", "pumpSettings"}}},
	}

	opts := options.FindOne()
	opts.SetHint(idxUserIDTypeTime)
	opts.SetProjection(wantedRangeFields)
	opts.SetComment(traceID)

	// Finding Last time (i.e. findOne with sort time DESC)
	opts.SetSort(bson.D{primitive.E{Key: "time", Value: -1}})
	err := dataCollection(p).FindOne(ctx, query, opts).Decode(&res)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	dateRange.End = res["time"].(string)

	// Finding First time (i.e. findOne with sort time ASC)
	opts.SetSort(bson.D{primitive.E{Key: "time", Value: 1}})
	err = dataCollection(p).FindOne(ctx, query, opts).Decode(&res)
	if err != nil {
		return nil, err
	}
	dateRange.Start = res["time"].(string)

	return dateRange, nil
}

// GetDataV1 v1 api call to fetch diabetes data, excludes "upload" and "pumpSettings"
// and potentially other types
func (p *PatientDataMongoRepository) GetDataInDeviceData(ctx context.Context, traceID string, userID string, dates *common.Date, excludeTypes []string) (goComMgo.StorageIterator, error) {
	if !InArray("upload", excludeTypes) {
		excludeTypes = append(excludeTypes, "upload")
	}
	if !InArray("pumpSettings", excludeTypes) {
		excludeTypes = append(excludeTypes, "pumpSettings")
	}
	query := bson.M{
		"_userId": userID,
		"type":    bson.M{"$not": bson.M{"$in": excludeTypes}},
	}

	if dates.Start != "" && dates.End != "" {
		query["time"] = bson.M{"$gte": dates.Start, "$lt": dates.End}
	} else if dates.Start != "" {
		query["time"] = bson.M{"$gte": dates.Start}
	} else if dates.End != "" {
		query["time"] = bson.M{"$lt": dates.End}
	}

	opts := options.Find()
	opts.SetProjection(unwantedFields)
	opts.SetHint(idxUserIDTypeTime)
	opts.SetComment(traceID)

	return dataCollection(p).Find(ctx, query, opts)
}

// GetLoopMode v1 api call to fetch Loop Mode objects
// and potentially other types
func (p *PatientDataMongoRepository) GetLoopMode(ctx context.Context, traceID string, userID string, dates *common.Date) ([]schema.LoopModeEvent, error) {
	matchUserType := bson.M{
		"$match": bson.M{
			"_userId": userID,
			"type":    "deviceEvent",
			"subType": "loopMode",
		},
	}
	projectDuration := bson.M{
		"$project": bson.M{
			"startTime": bson.M{"$toDate": "$time"},
			"milliseconds": bson.M{
				"$switch": bson.M{
					"branches": bson.A{
						bson.M{
							"case": bson.M{"$eq": bson.A{"$duration.units", "milliseconds"}},
							"then": "$duration.value",
						},
						bson.M{
							"case": bson.M{"$eq": bson.A{"$duration.units", "seconds"}},
							"then": bson.M{"$multiply": bson.A{"$duration.value", 1000}},
						},
						bson.M{
							"case": bson.M{"$eq": bson.A{"$duration.units", "minutes"}},
							"then": bson.M{"$multiply": bson.A{"$duration.value", 1000 * 60}},
						},
						bson.M{
							"case": bson.M{"$eq": bson.A{"$duration.units", "hours"}},
							"then": bson.M{"$multiply": bson.A{"$duration.value", "$duration.value", 1000 * 60 * 60}},
						},
					},
					"default": nil,
				},
			},
		},
	}

	projectEndTime := bson.M{
		"$project": bson.M{
			"startTime": 1,
			"endTime":   bson.M{"$add": bson.A{"$startTime", "$milliseconds"}},
			"_id":       0,
		},
	}
	aggregatesStep := []bson.M{matchUserType, projectDuration, projectEndTime}

	startTime, _ := time.Parse(time.RFC3339Nano, dates.Start)
	endTime, _ := time.Parse(time.RFC3339Nano, dates.End)
	if !startTime.IsZero() && !endTime.IsZero() {
		dateMatch := bson.M{
			"$match": bson.M{
				"$or": bson.A{
					bson.M{"endTime": nil},
					bson.M{
						"startTime": bson.M{
							"$gte": startTime,
							"$lt":  endTime,
						},
					},
					bson.M{
						"endTime": bson.M{
							"$gte": startTime,
							"$lt":  endTime,
						},
					},
				},
			},
		}
		aggregatesStep = append(aggregatesStep, dateMatch)
	} else if !startTime.IsZero() {
		dateMatch := bson.M{
			"$match": bson.M{
				"$or": bson.A{
					bson.M{"endTime": nil},
					bson.M{
						"startTime": bson.M{
							"$gte": startTime,
						},
					},
					bson.M{
						"endTime": bson.M{
							"$gte": startTime,
						},
					},
				},
			},
		}
		aggregatesStep = append(aggregatesStep, dateMatch)
	} else if !endTime.IsZero() {
		dateMatch := bson.M{
			"$match": bson.M{
				"$or": bson.A{
					bson.M{"endTime": nil},
					bson.M{
						"startTime": bson.M{
							"$lt": endTime,
						},
					},
					bson.M{
						"endTime": bson.M{
							"$lt": endTime,
						},
					},
				},
			},
		}
		aggregatesStep = append(aggregatesStep, dateMatch)
	}
	// Sorting step
	aggregatesStep = append(aggregatesStep, bson.M{
		"$sort": bson.M{
			"startTime": 1,
		},
	})
	opts := options.Aggregate()
	opts.SetComment(traceID)

	cursor, err := dataCollection(p).Aggregate(ctx, aggregatesStep, opts)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return []schema.LoopModeEvent{}, nil
		}
		return nil, err
	}
	defer cursor.Close(ctx)
	var res []schema.LoopModeEvent
	err = cursor.All(ctx, &res)
	return res, err
}

func (p *PatientDataMongoRepository) GetLatestBasalSecurityProfile(ctx context.Context, traceID string, userID string) (*schema.DbProfile, error) {
	if userID == "" {
		return nil, errors.New("invalid user id")
	}

	query := bson.M{
		"_userId": userID,
		"type":    "basalSecurity",
	}
	opts := options.FindOne()
	//opts.SetProjection(unwantedPumpSettingsFields) TODO
	opts.SetSort(bson.M{"time": -1})
	opts.SetComment(traceID)
	var result schema.DbProfile
	err := dataCollection(p).FindOne(ctx, query, opts).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return &result, nil
}

// GetUploadDataV1 Fetch upload data from theirs upload ids, using the $in query parameter
func (p *PatientDataMongoRepository) GetUploadData(ctx context.Context, traceID string, uploadIds []string) (goComMgo.StorageIterator, error) {
	query := bson.M{
		"uploadId": bson.M{"$in": uploadIds},
		"type":     "upload",
	}

	opts := options.Find()
	opts.SetProjection(unwantedFields)
	opts.SetComment(traceID)
	return dataCollection(p).Find(ctx, query, opts)
}
