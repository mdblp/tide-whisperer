package store

import (
	"errors"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	"github.com/tidepool-org/go-common/clients/mongo"
	"github.com/tidepool-org/tide-whisperer/utils"
)

const (
	data_collection               = "deviceData"
	DATA_STORE_API_PREFIX         = "api/data/store"
	portal_db                     = "portal"
	parameters_history_collection = "patient_parameters_history"
)

type (
	//Interface for the query iterator
	StorageIterator interface {
		Next(result interface{}) bool
		Close() error
	}
	//Interface for our storage layer
	Storage interface {
		Close()
		Ping() error
		GetDeviceData(p *Params) StorageIterator
		GetTimeInRangeData(userPrefs []utils.UserPref) StorageIterator
	}
	//Mongo Storage Client
	MongoStoreClient struct {
		session *mgo.Session
	}

	SchemaVersion struct {
		Minimum int
		Maximum int
	}

	Params struct {
		UserId   string
		Types    []string
		SubTypes []string
		Date
		*SchemaVersion
		Carelink           bool
		Dexcom             bool
		DexcomDataSource   bson.M
		DeviceId           string
		Latest             bool
		Medtronic          bool
		MedtronicDate      string
		MedtronicUploadIds []string
		UploadId           string
		LevelFilter        []int
	}

	Date struct {
		Start string
		End   string
	}

	ClosingSessionIterator struct {
		*mgo.Session
		*mgo.Iter
	}
)

func InArray(needle string, arr []string) bool {
	for _, n := range arr {
		if needle == n {
			return true
		}
	}
	return false
}

func cleanDateString(dateString string) (string, error) {
	if dateString == "" {
		return "", nil
	}
	date, err := time.Parse(time.RFC3339Nano, dateString)
	if err != nil {
		return "", err
	}
	return date.Format(time.RFC3339Nano), nil
}

func GetParams(q url.Values, schema *SchemaVersion, config *mongo.Config) (*Params, error) {

	startStr, err := cleanDateString(q.Get("startDate"))
	if err != nil {
		return nil, err
	}

	endStr, err := cleanDateString(q.Get("endDate"))
	if err != nil {
		return nil, err
	}

	carelink := false
	if values, ok := q["carelink"]; ok {
		if len(values) < 1 {
			return nil, errors.New("carelink parameter not valid")
		}
		carelink, err = strconv.ParseBool(values[len(values)-1])
		if err != nil {
			return nil, errors.New("carelink parameter not valid")
		}
	}

	dexcom := false
	if values, ok := q["dexcom"]; ok {
		if len(values) < 1 {
			return nil, errors.New("dexcom parameter not valid")
		}
		dexcom, err = strconv.ParseBool(values[len(values)-1])
		if err != nil {
			return nil, errors.New("dexcom parameter not valid")
		}
	}

	latest := false
	if values, ok := q["latest"]; ok {
		if len(values) < 1 {
			return nil, errors.New("latest parameter not valid")
		}
		latest, err = strconv.ParseBool(values[len(values)-1])
		if err != nil {
			return nil, errors.New("latest parameter not valid")
		}
	}

	medtronic := false
	if values, ok := q["medtronic"]; ok {
		if len(values) < 1 {
			return nil, errors.New("medtronic parameter not valid")
		}
		medtronic, err = strconv.ParseBool(values[len(values)-1])
		if err != nil {
			return nil, errors.New("medtronic parameter not valid")
		}
	}

	storage := NewMongoStoreClient(config)

	// get Device model
	var device string
	var deviceErr error
	var UserID = q.Get(":userID")
	if device, deviceErr = storage.GetDeviceModel(UserID); deviceErr != nil {
		log.Printf("Error in GetDeviceModel for user %s. Error: %s", UserID, deviceErr)
	}

	LevelFilter := make([]int, 1)
	LevelFilter = append(LevelFilter, 1)
	if device == "DBLHU" {
		LevelFilter = append(LevelFilter, 2)
		LevelFilter = append(LevelFilter, 3)
	}

	p := &Params{
		UserId:   q.Get(":userID"),
		DeviceId: q.Get("deviceId"),
		UploadId: q.Get("uploadId"),
		//the query params for type and subtype can contain multiple values seperated
		//by a comma e.g. "type=smbg,cbg" so split them out into an array of values
		Types:         strings.Split(q.Get("type"), ","),
		SubTypes:      strings.Split(q.Get("subType"), ","),
		Date:          Date{startStr, endStr},
		SchemaVersion: schema,
		Carelink:      carelink,
		Dexcom:        dexcom,
		Latest:        latest,
		Medtronic:     medtronic,
		LevelFilter:   LevelFilter,
	}

	return p, nil

}

func NewMongoStoreClient(config *mongo.Config) *MongoStoreClient {

	mongoSession, err := mongo.Connect(config)
	if err != nil {
		log.Fatal(DATA_STORE_API_PREFIX, err)
	}

	return &MongoStoreClient{
		session: mongoSession,
	}
}

func mgoDataCollection(cpy *mgo.Session) *mgo.Collection {
	return cpy.DB("").C(data_collection)
}
func mgoParametersHistoryCollection(cpy *mgo.Session) *mgo.Collection {
	return cpy.DB(portal_db).C(parameters_history_collection)
}

// generateMongoQuery takes in a number of parameters and constructs a mongo query
// to retrieve objects from the Tidepool database. It is used by the router.Add("GET", "/{userID}"
// endpoint, which implements the Tide-whisperer API. See that function for further documentation
// on parameters
func generateMongoQuery(p *Params) bson.M {

	finalQuery := bson.M{}
	skipParamsQuery := false
	groupDataQuery := bson.M{
		"_userId":        p.UserId,
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

	if p.DeviceId != "" {
		groupDataQuery["deviceId"] = p.DeviceId
		skipParamsQuery = true
	}

	// If we have an explicit upload ID to filter by, we don't need or want to apply any further
	// data source-based filtering
	if p.UploadId != "" {
		groupDataQuery["uploadId"] = p.UploadId
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
			otherDataQuery["type"] = bson.M{"$ne": "deviceEvent"}
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

func (d MongoStoreClient) Close() {
	log.Print(DATA_STORE_API_PREFIX, "Close the session")
	d.session.Close()
	return
}

func (d MongoStoreClient) Ping() error {
	session := d.session.Copy()
	defer session.Close()
	// do we have a store session
	return session.Ping()
}

func (d MongoStoreClient) HasMedtronicDirectData(userID string) (bool, error) {
	if userID == "" {
		return false, errors.New("user id is missing")
	}

	session := d.session.Copy()
	defer session.Close()

	query := bson.M{
		"_userId": userID,
		"type":    "upload",
		"_state":  "closed",
		"_active": true,
		"deletedTime": bson.M{
			"$exists": false,
		},
		"deviceManufacturers": "Medtronic",
	}

	count, err := mgoDataCollection(session).Find(query).Limit(1).Count()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (d MongoStoreClient) GetDexcomDataSource(userID string) (bson.M, error) {
	if userID == "" {
		return nil, errors.New("user id is missing")
	}

	session := d.session.Copy()
	defer session.Close()

	query := bson.M{
		"userId":       userID,
		"providerType": "oauth",
		"providerName": "dexcom",
		"dataSetIds": bson.M{
			"$exists": true,
			"$not": bson.M{
				"$size": 0,
			},
		},
		"earliestDataTime": bson.M{
			"$exists": true,
		},
		"latestDataTime": bson.M{
			"$exists": true,
		},
	}

	dataSources := []bson.M{}
	err := session.DB("tidepool").C("data_sources").Find(query).Limit(1).All(&dataSources)
	if err != nil {
		return nil, err
	} else if len(dataSources) == 0 {
		return nil, nil
	}

	return dataSources[0], nil
}

func (d MongoStoreClient) HasMedtronicLoopDataAfter(userID string, date string) (bool, error) {
	if userID == "" {
		return false, errors.New("user id is missing")
	}
	if date == "" {
		return false, errors.New("date is missing")
	}

	session := d.session.Copy()
	defer session.Close()

	query := bson.M{
		"_active":                            true,
		"_userId":                            userID,
		"_schemaVersion":                     bson.M{"$gt": 0},
		"time":                               bson.M{"$gte": date},
		"origin.payload.device.manufacturer": "Medtronic",
	}

	count, err := mgoDataCollection(session).Find(query).Limit(1).Count()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (d MongoStoreClient) GetLoopableMedtronicDirectUploadIdsAfter(userID string, date string) ([]string, error) {
	if userID == "" {
		return nil, errors.New("user id is missing")
	}
	if date == "" {
		return nil, errors.New("date is missing")
	}

	session := d.session.Copy()
	defer session.Close()

	query := bson.M{
		"_active":        true,
		"_userId":        userID,
		"_schemaVersion": bson.M{"$gt": 0},
		"time":           bson.M{"$gte": date},
		"type":           "upload",
		"deviceModel":    bson.M{"$in": []string{"523", "523K", "554", "723", "723K", "754"}},
	}

	objects := []struct {
		UploadID string `bson:"uploadId"`
	}{}
	err := mgoDataCollection(session).Find(query).Select(bson.M{"_id": 0, "uploadId": 1}).All(&objects)
	if err != nil {
		return nil, err
	}

	uploadIds := make([]string, len(objects))
	for index, object := range objects {
		uploadIds[index] = object.UploadID
	}

	return uploadIds, nil
}

func (d MongoStoreClient) GetDeviceData(p *Params) StorageIterator {

	removeFieldsForReturn := bson.M{"_id": 0, "_userId": 0, "_groupId": 0, "_version": 0, "_active": 0, "_schemaVersion": 0, "createdTime": 0, "modifiedTime": 0}

	// Note: We do not defer closing the session copy here as the iterator is returned back to be
	// caller for processing. Instead, we wrap the session and iterator in an object that
	// closes session when the iterator is closed. See ClosingSessionIterator below.
	session := d.session.Copy()

	var iter *mgo.Iter

	if p.Latest {
		// Create an $aggregate query to return the latest of each `type` requested
		// that matches the query parameters
		pipeline := []bson.M{
			{
				"$match": generateMongoQuery(p),
			},
			{
				"$sort": bson.M{
					"type": 1,
					"time": -1,
				},
			},
			{
				"$group": bson.M{
					"_id": bson.M{
						"type": "$type",
					},
					"groupId": bson.M{
						"$first": "$_id",
					},
				},
			},
			{
				"$lookup": bson.M{
					"from":         "deviceData",
					"localField":   "groupId",
					"foreignField": "_id",
					"as":           "latest_doc",
				},
			},
			{
				"$unwind": "$latest_doc",
			},
			/*
				// TODO: we can only use this code once we upgrade to MongoDB 3.4+
				// We would also need to update the corresponding code in `tide-whisperer.go`
				// (search for "latest_doc")
				{
					"$replaceRoot": bson.M{
						"newRoot": "$latest_doc"
					},
				},
			*/
		}
		pipe := mgoDataCollection(session).Pipe(pipeline)
		iter = pipe.Iter()
	} else {
		iter = mgoDataCollection(session).
			Find(generateMongoQuery(p)).
			Select(removeFieldsForReturn).
			Iter()
	}

	return &ClosingSessionIterator{session, iter}
}

func (d MongoStoreClient) GetDiabeloopParametersHistory(userID string, levels []int) (bson.M, error) {
	if userID == "" {
		return nil, errors.New("user id is missing")
	}
	if levels == nil {
		levels = make([]int, 1)
		levels[0] = 1
	}

	var bsonLevels = make([]interface{}, len(levels))
	for i, d := range levels {
		bsonLevels[i] = d
	}

	session := d.session.Copy()
	defer session.Close()

	query := []bson.M{
		// Filtering on userid
		{
			"$match": bson.M{"userid": userID},
		},
		// unnesting history array (keeping index for future grouping)
		{
			"$unwind": bson.M{"path": "$history", "includeArrayIndex": "historyIdx"},
		},
		// unnesting history.parameters array
		{
			"$unwind": "$history.parameters",
		},
		// filtering level parameters
		{
			"$match": bson.M{
				"history.parameters.level": bson.M{"$in": bsonLevels},
			},
		},
		// removing unnecessary fields
		{
			"$project": bson.M{
				"userid":     1,
				"historyIdx": 1,
				"_id":        0,
				"parameters": bson.M{
					"changeType": "$history.parameters.changeType", "name": "$history.parameters.name",
					"value": "$history.parameters.value", "unit": "$history.parameters.unit",
					"level": "$history.parameters.level", "effectiveDate": "$history.parameters.effectiveDate",
				},
			},
		},
		// grouping by change
		{
			"$group": bson.M{
				"_id":        bson.M{"historyIdx": "$historyIdx", "userid": "$userid"},
				"parameters": bson.M{"$addToSet": "$parameters"},
				"changeDate": bson.M{"$max": "$parameters.effectiveDate"},
			},
		},
		// grouping all changes in one array
		{
			"$group": bson.M{
				"_id":     bson.M{"userid": "$userid"},
				"history": bson.M{"$addToSet": bson.M{"parameters": "$parameters", "changeDate": "$changeDate"}},
			},
		},
		// removing unnecessary fields
		{
			"$project": bson.M{"_id": 0},
		},
	}

	dataSources := []bson.M{}
	err := mgoParametersHistoryCollection(session).Pipe(query).All(&dataSources)
	if err != nil {
		return nil, err
	} else if len(dataSources) == 0 {
		return nil, nil
	}

	return dataSources[0], nil
}
func (d MongoStoreClient) GetDeviceModel(userID string) (string, error) {

	if userID == "" {
		return "", errors.New("user id is missing")
	}

	var payLoadDeviceNameQuery = make([]interface{}, 2)
	payLoadDeviceNameQuery[0] = bson.M{"payload.device.name": bson.M{"$exists": true}}
	payLoadDeviceNameQuery[1] = bson.M{"payload.device.name": bson.M{"$ne": nil}}

	query := bson.M{
		"_userId":        userID,
		"type":           "pumpSettings",
		"_schemaVersion": bson.M{"$gt": 0},
		"_active":        true,
		"$and":           payLoadDeviceNameQuery,
	}

	session := d.session.Copy()
	defer session.Close()

	var res map[string]interface{}
	err := mgoDataCollection(session).Find(query).Sort("-time").Select(bson.M{"payload.device.name": 1}).One(&res)
	if err != nil {
		return "", err
	}

	device := res["payload"].(map[string]interface{})["device"].(map[string]interface{})
	return device["name"].(string), err
}

func (i *ClosingSessionIterator) Next(result interface{}) bool {
	if i.Iter != nil {
		return i.Iter.Next(result)
	}
	return false
}

func (i *ClosingSessionIterator) Close() (err error) {
	if i.Iter != nil {
		err = i.Iter.Close()
		i.Iter = nil
	}
	if i.Session != nil {
		i.Session.Close()
		i.Session = nil
	}
	return err
}
