package infrastructure

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tidepool-org/tide-whisperer/common"
	"go.mongodb.org/mongo-driver/bson"

	goComMgo "github.com/tidepool-org/go-common/clients/mongo"
)

var testingConfig = &goComMgo.Config{
	Timeout:                2 * time.Second,
	WaitConnectionInterval: 5 * time.Second,
	MaxConnectionAttempts:  0,
}

func before(t *testing.T, docs ...interface{}) *PatientDataMongoRepository {
	var err error
	var ctx = context.Background()

	logger := log.New(os.Stdout, "mongo-test ", log.LstdFlags|log.LUTC|log.Lshortfile)

	if _, exist := os.LookupEnv("TIDEPOOL_STORE_ADDRESSES"); !exist {
		os.Setenv("TIDEPOOL_STORE_ADDRESSES", "localhost:27018")
		os.Setenv("TIDEPOOL_STORE_DATABASE", "data_test")
	}
	testingConfig.FromEnv()

	store, err := NewPatientDataMongoRepository(testingConfig, logger)
	if err != nil {
		t.Fatalf("Unexpected error while creating store: %s", err)
	}
	store.Start()
	store.WaitUntilStarted()

	if len(docs) > 0 {
		if _, err := dataCollection(store).InsertMany(ctx, docs); err != nil {
			t.Error("Unable to insert documents", err)
		}
	}
	t.Cleanup(func() {
		dataCollection(store).Drop(ctx)
		store.Close()
	})
	return store
}

func getErrString(mongoQuery, expectedQuery bson.M) string {
	return "expected:\n" + formatForReading(expectedQuery) + "\ndid not match returned query\n" + formatForReading(mongoQuery)
}

func formatForReading(toFormat interface{}) string {
	formatted, _ := json.MarshalIndent(toFormat, "", "  ")
	return string(formatted)
}

func basicQuery() bson.M {
	qParams := &common.Params{
		UserID:        "abc123",
		SchemaVersion: &common.SchemaVersion{Maximum: 2, Minimum: 0},
		Dexcom:        true,
		Medtronic:     true,
	}

	return generateMongoQuery(qParams)
}

func allParams() *common.Params {
	earliestDataTime, _ := time.Parse(time.RFC3339, "2015-10-07T15:00:00Z")
	latestDataTime, _ := time.Parse(time.RFC3339, "2016-12-13T02:00:00Z")

	return &common.Params{
		UserID:        "abc123",
		DeviceID:      "device123",
		SchemaVersion: &common.SchemaVersion{Maximum: 2, Minimum: 0},
		Date:          common.Date{"2015-10-07T15:00:00.000Z", "2015-10-11T15:00:00.000Z"},
		Types:         []string{"smbg", "cbg"},
		SubTypes:      []string{"stuff"},
		Carelink:      true,
		Dexcom:        false,
		DexcomDataSource: bson.M{
			"dataSetIds":       []string{"123", "456"},
			"earliestDataTime": earliestDataTime,
			"latestDataTime":   latestDataTime,
		},
		Latest:             false,
		Medtronic:          false,
		MedtronicDate:      "2017-01-01T00:00:00Z",
		MedtronicUploadIds: []string{"555666777", "888999000"},
	}
}

func allParamsQuery() bson.M {
	return generateMongoQuery(allParams())
}

func allParamsIncludingUploadIDQuery() bson.M {
	qParams := allParams()
	qParams.UploadID = "xyz123"

	return generateMongoQuery(qParams)
}

func typeAndSubtypeQuery() bson.M {
	qParams := &common.Params{
		UserID:             "abc123",
		SchemaVersion:      &common.SchemaVersion{Maximum: 2, Minimum: 0},
		Types:              []string{"smbg", "cbg"},
		SubTypes:           []string{"stuff"},
		Dexcom:             true,
		Medtronic:          false,
		MedtronicDate:      "2017-01-01T00:00:00Z",
		MedtronicUploadIds: []string{"555666777", "888999000"},
	}
	return generateMongoQuery(qParams)
}

func uploadIDQuery() bson.M {
	qParams := &common.Params{
		UserID:        "abc123",
		SchemaVersion: &common.SchemaVersion{Maximum: 2, Minimum: 0},
		UploadID:      "xyz123",
	}
	return generateMongoQuery(qParams)
}

func blipQuery() bson.M {
	qParams := &common.Params{
		UserID:        "abc123",
		SchemaVersion: &common.SchemaVersion{Maximum: 2, Minimum: 0},
		LevelFilter:   []int{1, 2},
		Date:          common.Date{"2015-10-07T15:00:00.000Z", "2015-11-07T15:00:00.000Z"},
	}

	return generateMongoQuery(qParams)
}

func typesWithDeviceEventQuery() bson.M {
	qParams := &common.Params{
		UserID:        "abc123",
		SchemaVersion: &common.SchemaVersion{Maximum: 2, Minimum: 0},
		LevelFilter:   []int{1, 2},
		Date:          common.Date{"2015-10-07T15:00:00.000Z", "2015-11-07T15:00:00.000Z"},
		Types:         []string{"deviceEvent", "food"},
	}

	return generateMongoQuery(qParams)
}

func typesWithoutDeviceEventQuery() bson.M {
	qParams := &common.Params{
		UserID:        "abc123",
		SchemaVersion: &common.SchemaVersion{Maximum: 2, Minimum: 0},
		LevelFilter:   []int{1, 2},
		Date:          common.Date{"2015-10-07T15:00:00.000Z", "2015-11-07T15:00:00.000Z"},
		Types:         []string{"food"},
	}

	return generateMongoQuery(qParams)
}

func typesWithDeviceEventAndSubTypeQuery() bson.M {
	qParams := &common.Params{
		UserID:        "abc123",
		SchemaVersion: &common.SchemaVersion{Maximum: 2, Minimum: 0},
		LevelFilter:   []int{1, 2},
		Date:          common.Date{"2015-10-07T15:00:00.000Z", "2015-11-07T15:00:00.000Z"},
		Types:         []string{"deviceEvent", "food"},
		SubTypes:      []string{"reservoirChange"},
	}

	return generateMongoQuery(qParams)
}

func iteratorToAllData(ctx context.Context, iter goComMgo.StorageIterator) ([]map[string]interface{}, error) {
	var data []map[string]interface{}
	var err error
	// TODO all All(ctx, &data) function to StorageIterator
	for iter.Next(ctx) {
		var datum map[string]interface{}
		err = iter.Decode(&datum)
		if err != nil {
			break
		}
		data = append(data, datum)
	}
	return data, err
}

func TestStore_generateMongoQuery_basic(t *testing.T) {
	query := basicQuery()
	expectedQuery := bson.M{
		"_userId":        "abc123",
		"_active":        true,
		"_schemaVersion": bson.M{"$gte": 0, "$lte": 2},
		"source": bson.M{
			"$ne": "carelink",
		},
	}

	eq := reflect.DeepEqual(query, expectedQuery)
	if !eq {
		t.Error(getErrString(query, expectedQuery))
	}

}

func TestStore_generateMongoQuery_allParams(t *testing.T) {

	query := allParamsQuery()

	expectedQuery := bson.M{
		"_userId":        "abc123",
		"deviceId":       "device123",
		"_active":        true,
		"_schemaVersion": bson.M{"$gte": 0, "$lte": 2},
		"type":           bson.M{"$in": strings.Split("smbg,cbg", ",")},
		"subType":        bson.M{"$in": strings.Split("stuff", ",")},
		"time": bson.M{
			"$gte": "2015-10-07T15:00:00.000Z",
			"$lte": "2015-10-11T15:00:00.000Z"},
		"$and": []bson.M{
			{"$or": []bson.M{
				{"type": bson.M{"$ne": "cbg"}},
				{"uploadId": bson.M{"$in": []string{"123", "456"}}},
				{"time": bson.M{"$lt": "2015-10-07T15:00:00Z"}},
				{"time": bson.M{"$gt": "2016-12-13T02:00:00Z"}},
			}},
			{"$or": []bson.M{
				{"time": bson.M{"$lt": "2017-01-01T00:00:00Z"}},
				{"type": bson.M{"$nin": []string{"basal", "bolus", "cbg"}}},
				{"uploadId": bson.M{"$nin": []string{"555666777", "888999000"}}},
			}},
		},
	}

	eq := reflect.DeepEqual(query, expectedQuery)
	if !eq {
		t.Error(getErrString(query, expectedQuery))
	}
}

func TestStore_generateMongoQuery_allparamsWithUploadId(t *testing.T) {

	query := allParamsIncludingUploadIDQuery()

	expectedQuery := bson.M{
		"_userId":        "abc123",
		"deviceId":       "device123",
		"_active":        true,
		"_schemaVersion": bson.M{"$gte": 0, "$lte": 2},
		"type":           bson.M{"$in": strings.Split("smbg,cbg", ",")},
		"subType":        bson.M{"$in": strings.Split("stuff", ",")},
		"uploadId":       "xyz123",
		"time": bson.M{
			"$gte": "2015-10-07T15:00:00.000Z",
			"$lte": "2015-10-11T15:00:00.000Z"},
	}

	eq := reflect.DeepEqual(query, expectedQuery)
	if !eq {
		t.Error(getErrString(query, expectedQuery))
	}
}

func TestStore_generateMongoQuery_uploadId(t *testing.T) {

	query := uploadIDQuery()

	expectedQuery := bson.M{
		"_userId":        "abc123",
		"_active":        true,
		"_schemaVersion": bson.M{"$gte": 0, "$lte": 2},
		"uploadId":       "xyz123",
		"source": bson.M{
			"$ne": "carelink",
		},
	}

	eq := reflect.DeepEqual(query, expectedQuery)
	if !eq {
		t.Error(getErrString(query, expectedQuery))
	}
}

func TestStore_generateMongoQuery_noDates(t *testing.T) {

	query := typeAndSubtypeQuery()

	expectedQuery := bson.M{
		"_userId":        "abc123",
		"_active":        true,
		"type":           bson.M{"$in": strings.Split("smbg,cbg", ",")},
		"subType":        bson.M{"$in": strings.Split("stuff", ",")},
		"_schemaVersion": bson.M{"$gte": 0, "$lte": 2},
		"source": bson.M{
			"$ne": "carelink",
		},
		"$and": []bson.M{
			{"$or": []bson.M{
				{"time": bson.M{"$lt": "2017-01-01T00:00:00Z"}},
				{"type": bson.M{"$nin": []string{"basal", "bolus", "cbg"}}},
				{"uploadId": bson.M{"$nin": []string{"555666777", "888999000"}}},
			}},
		},
	}

	eq := reflect.DeepEqual(query, expectedQuery)
	if !eq {
		t.Error(getErrString(query, expectedQuery))
	}
}

func TestStore_generateMongoQuery_blip(t *testing.T) {

	query := blipQuery()

	expectedQuery := bson.M{
		"$and": []bson.M{
			{
				"_userId":        "abc123",
				"_active":        true,
				"_schemaVersion": bson.M{"$gte": 0, "$lte": 2},
				"source":         bson.M{"$ne": "carelink"},
				"time": bson.M{
					"$gte": "2015-10-07T15:00:00.000Z",
					"$lte": "2015-11-07T15:00:00.000Z"},
			},
			bson.M{"$or": []bson.M{
				bson.M{
					"level":   bson.M{"$in": []string{"0", "1"}},
					"subType": "deviceParameter",
					"type":    "deviceEvent",
				},
				bson.M{"subType": bson.M{"$ne": "deviceParameter"}},
			},
			},
		},
	}

	eq := reflect.DeepEqual(query, expectedQuery)
	if !eq {
		t.Error(getErrString(query, expectedQuery))
	}
}

func TestStore_generateMongoQuery_withDETypes(t *testing.T) {

	query := typesWithDeviceEventQuery()

	expectedQuery := bson.M{
		"$and": []bson.M{
			{
				"_userId":        "abc123",
				"_active":        true,
				"_schemaVersion": bson.M{"$gte": 0, "$lte": 2},
				"source":         bson.M{"$ne": "carelink"},
				"time": bson.M{
					"$gte": "2015-10-07T15:00:00.000Z",
					"$lte": "2015-11-07T15:00:00.000Z"},
				"type": bson.M{"$in": []string{"deviceEvent", "food"}},
			},
			bson.M{"$or": []bson.M{
				bson.M{
					"level":   bson.M{"$in": []string{"0", "1"}},
					"subType": "deviceParameter",
					"type":    "deviceEvent",
				},
				bson.M{"subType": bson.M{"$ne": "deviceParameter"}},
			},
			},
		},
	}

	eq := reflect.DeepEqual(query, expectedQuery)
	if !eq {
		t.Error(getErrString(query, expectedQuery))
	}
}

func TestStore_generateMongoQuery_withoutDETypes(t *testing.T) {

	query := typesWithoutDeviceEventQuery()

	expectedQuery := bson.M{
		"_userId": "abc123",
		"_active": true,
		"time": bson.M{
			"$gte": "2015-10-07T15:00:00.000Z",
			"$lte": "2015-11-07T15:00:00.000Z"},
		"type":           bson.M{"$in": []string{"food"}},
		"_schemaVersion": bson.M{"$gte": 0, "$lte": 2},
		"source": bson.M{
			"$ne": "carelink",
		},
	}

	eq := reflect.DeepEqual(query, expectedQuery)
	if !eq {
		t.Error(getErrString(query, expectedQuery))
	}
}

func TestStore_generateMongoQuery_withDETypesAndSubType(t *testing.T) {

	query := typesWithDeviceEventAndSubTypeQuery()

	expectedQuery := bson.M{
		"_userId":        "abc123",
		"_active":        true,
		"_schemaVersion": bson.M{"$gte": 0, "$lte": 2},
		"source":         bson.M{"$ne": "carelink"},
		"time": bson.M{
			"$gte": "2015-10-07T15:00:00.000Z",
			"$lte": "2015-11-07T15:00:00.000Z"},
		"type":    bson.M{"$in": []string{"deviceEvent", "food"}},
		"subType": bson.M{"$in": []string{"reservoirChange"}},
	}

	eq := reflect.DeepEqual(query, expectedQuery)
	if !eq {
		t.Error(getErrString(query, expectedQuery))
	}
}

func TestStore_Ping(t *testing.T) {

	store := before(t)
	err := store.Ping()

	if err != nil {
		t.Error("there should be no error but got", err.Error())
	}
}

func TestStore_GetDataRangeV1(t *testing.T) {
	userID := "abcdef"
	startDate := "2020-01-01T00:00:00.000Z"
	endDate := "2021-01-01T00:00:00.000Z"
	store := before(t,
		bson.M{
			"id":      uuid.New().String(),
			"_userId": userID,
			"time":    "2020-01-01T00:00:00.000Z",
			"type":    "cbg",
			"units":   "mmol/L",
			"value":   12,
		},
		bson.M{
			"id":      uuid.New().String(),
			"_userId": userID,
			"time":    "2020-06-01T00:00:00.000Z",
			"type":    "cbg",
			"units":   "mmol/L",
			"value":   12,
		},
		bson.M{
			"id":      uuid.New().String(),
			"_userId": userID,
			"time":    "2021-01-01T00:00:00.000Z",
			"type":    "cbg",
			"units":   "mmol/L",
			"value":   12,
		},
	)
	traceID := uuid.New().String()
	res, err := store.GetDataRangeLegacy(context.Background(), traceID, userID)
	if err != nil {
		t.Errorf("Unexpected error during GetDataRangeLegacy: %s", err)
	}
	if res.Start != startDate {
		t.Errorf("Expected %s to equal %s", res.Start, startDate)
	}
	if res.End != endDate {
		t.Errorf("Expected %s to equal %s", res.End, endDate)
	}
}

func TestStore_GetDataV1(t *testing.T) {
	var err error
	var iter goComMgo.StorageIterator
	var data []map[string]interface{}
	userID := "abcdef"
	ddr := &common.Date{
		Start: "2020-05-01T00:00:00.000Z",
		End:   "2021-01-02T00:00:00.000Z",
	}
	store := before(t,
		bson.M{
			"_userId": userID,
			"id":      "1",
			"time":    "2020-01-01T00:00:00.000Z",
			"type":    "cbg",
			"units":   "mmol/L",
			"value":   12,
		},
		bson.M{
			"_userId": userID,
			"id":      "2",
			"time":    "2020-06-01T00:00:00.000Z",
			"type":    "cbg",
			"units":   "mmol/L",
			"value":   12,
		},
		bson.M{
			"_userId": "a00000",
			"id":      "a",
			"time":    "2020-11-01T00:00:00.000Z",
			"type":    "cbg",
			"units":   "mmol/L",
			"value":   12,
		},
		bson.M{
			"_userId": userID,
			"id":      "3",
			"time":    "2021-01-01T00:00:00.000Z",
			"type":    "cbg",
			"units":   "mmol/L",
			"value":   12,
		},
	)
	ctx := context.Background()
	traceID := uuid.New().String()
	iter, err = store.GetDataInDeviceData(ctx, traceID, userID, ddr, []string{})
	if err != nil {
		t.Fatalf("Unexpected error during GetDataRangeLegacy: %s", err)
	}
	defer iter.Close(ctx)

	if data, err = iteratorToAllData(ctx, iter); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if len(data) != 2 {
		t.Fatalf("Expected a result of 2 data having %d", len(data))
	}

	for p, datum := range data {
		id := datum["id"].(string)
		if !(id == "2" || id == "3") {
			t.Log(data)
			t.Fatalf("Invalid datum id %s at %d", id, p)
		}
	}
}

func TestStore_GetUploadDataV1(t *testing.T) {
	var err error
	var iter goComMgo.StorageIterator
	var data []map[string]interface{}
	userID := "abcdef"

	store := before(t,
		bson.M{
			"_userId":  userID,
			"id":       "1",
			"uploadId": "1",
			"time":     "2020-01-01T00:00:00.000Z",
			"type":     "upload",
		},
		bson.M{
			"_userId":  userID,
			"id":       "2",
			"uploadId": "1",
			"time":     "2020-06-01T00:00:00.000Z",
			"type":     "cbg",
			"units":    "mmol/L",
			"value":    12,
		},
		bson.M{
			"_userId":  userID,
			"id":       "3",
			"uploadId": "3",
			"time":     "2020-11-01T00:00:00.000Z",
			"type":     "upload",
		},
		bson.M{
			"_userId":  userID,
			"id":       "4",
			"uploadId": "3",
			"time":     "2021-01-01T00:00:00.000Z",
			"type":     "cbg",
			"units":    "mmol/L",
			"value":    12,
		},
	)
	ctx := context.Background()
	traceID := uuid.New().String()
	ids := []string{"1", "2", "3", "4"}
	iter, err = store.GetUploadData(ctx, traceID, ids)
	if err != nil {
		t.Fatalf("Unexpected error during GetDataRangeLegacy: %s", err)
	}
	defer iter.Close(ctx)

	if data, err = iteratorToAllData(ctx, iter); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if len(data) != 2 {
		t.Fatalf("Expected a result of 2 data having %d", len(data))
	}

	for p, datum := range data {
		id := datum["id"].(string)
		if !(id == "1" || id == "3") {
			t.Log(data)
			t.Fatalf("Invalid datum id %s at %d", id, p)
		}
		uploadId := datum["uploadId"].(string)
		if !(uploadId == "1" || uploadId == "3") {
			t.Log(data)
			t.Fatalf("Invalid datum uploadId %s at %d", uploadId, p)
		}
		datumType := datum["type"].(string)
		if datumType != "upload" {
			t.Log(data)
			t.Fatalf("Invalid datum type %s at %d", datumType, p)
		}

	}
}

func TestStore_GetLatestBasalSecurityProfile(t *testing.T) {

	userID := "abcdef"
	store := before(t,
		bson.M{
			"_active":        true,
			"guid":           "1",
			"deviceId":       "Kaleido-fake-12345",
			"deviceTime":     "2020-01-01T08:20:00",
			"time":           "2020-01-01T08:20:00Z",
			"timezone":       "Etc/GMT-1",
			"timezoneOffset": 60,
			"type":           "basalSecurity",
			"_userId":        userID,
			"basalSchedule": []bson.M{
				{
					"rate":  1.0,
					"start": 0,
				},
				{
					"rate":  0.8,
					"start": 43200000,
				},
				{
					"rate":  1.2,
					"start": 64800000,
				},
				{
					"rate":  0.5,
					"start": 75600000,
				},
			},
		},
		bson.M{
			"_active":        true,
			"guid":           "2",
			"deviceId":       "Kaleido-fake-12345",
			"deviceTime":     "2020-01-01T08:40:00",
			"time":           "2020-01-01T08:40:00Z",
			"timezone":       "Etc/GMT-1",
			"timezoneOffset": 60,
			"type":           "basalSecurity",
			"_userId":        userID,
			"basalSchedule": []bson.M{
				{
					"rate":  1.0,
					"start": 0,
				},
				{
					"rate":  0.8,
					"start": 43200000,
				},
				{
					"rate":  1.2,
					"start": 64800000,
				},
				{
					"rate":  0.5,
					"start": 75600000,
				},
			},
		},
		bson.M{
			"_active":        true,
			"guid":           "3",
			"deviceId":       "Kaleido-fake-12345",
			"deviceTime":     "2020-01-01T09:00:00",
			"time":           "2020-01-01T09:00:00Z",
			"timezone":       "Etc/GMT-1",
			"timezoneOffset": 60,
			"type":           "basalSecurity",
			"_userId":        userID,
			"basalSchedule": []bson.M{
				{
					"rate":  1.0,
					"start": 0,
				},
				{
					"rate":  0.8,
					"start": 43200000,
				},
				{
					"rate":  1.2,
					"start": 64800000,
				},
				{
					"rate":  0.5,
					"start": 75600000,
				},
			},
		},
	)
	ctx := context.Background()
	traceID := uuid.New().String()

	data, err := store.GetLatestBasalSecurityProfile(ctx, traceID, userID)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if data.Guid != "3" {
		t.Fatalf("Expected return id to be 3, having %s", data.Guid)
	}
}

func ptr(t time.Time) *time.Time {
	return &t
}
