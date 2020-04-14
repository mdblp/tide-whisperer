package data

import (
	"encoding/json"
	"github.com/tidepool-org/tide-whisperer/store"
	// "go.mongodb.org/mongo-driver/mongo"
	"log"
	"net/http"
	"time"
)

type (
	// LoggerInfo struct for grouping fields used in log functions
	LoggerInfo struct {
		UserID       string
		UserIDs      []string
		requestID    string
		apiCallStart time.Time
		queryStart   time.Time
	}
	// only documented for swaggo
	cbgCounts struct {
		veryLow  int
		low      int
		target   int
		high     int
		veryHigh int
	}
	// only documented for swaggo
	cbgLastTimes struct {
		veryLow  time.Time
		low      time.Time
		target   time.Time
		high     time.Time
		veryHigh time.Time
	}
	// only documented for swaggo
	cbgRates struct {
		veryLow  float32
		low      float32
		target   float32
		high     float32
		veryHigh float32
	}
	// only documented for swaggo
	cbgTotalTimes struct {
		veryLow  int
		low      int
		target   int
		high     int
		veryHigh int
	}
	// TirResult only documented for swaggo
	TirResult struct {
		userId      string
		lastCbgTime time.Time
		count       cbgCounts
		lastTime    cbgLastTimes
		rate        cbgRates
		totalTime   cbgTotalTimes
	}
)

func writeMongoJSONResponse(res http.ResponseWriter, req *http.Request, cursor store.StorageIterator, logData *LoggerInfo) {
	res.Header().Add("Content-Type", "application/json")
	res.Write([]byte("["))

	var writeCount int
	var results map[string]interface{}
	for cursor.Next(req.Context()) {
		err := cursor.Decode(&results)
		if err != nil {
			logIndicatorError(logData, "Mongo Decode", err)
		}
		if len(results) > 0 {
			if bytes, err := json.Marshal(results); err != nil {
				logIndicatorError(logData, "Marshal", err)
			} else {
				if writeCount > 0 {
					res.Write([]byte(","))
				}
				res.Write([]byte("\n"))
				res.Write(bytes)
				writeCount++
			}
		}
	}
	if writeCount > 0 {
		res.Write([]byte("\n"))
	}
	res.Write([]byte("]"))
}
func logIndicatorError(logData *LoggerInfo, message string, err error) {
	log.Printf("%s request %s users %s %s returned error: %s", DataAPIPrefix, logData.requestID, logData.UserIDs, message, err)
}
func logIndicatorSlowQuery(logData *LoggerInfo, message string) {
	if queryDuration := time.Now().Sub(logData.queryStart).Seconds(); queryDuration > slowQueryDuration {
		log.Printf("%s request %s users %s %s took %.3fs", DataAPIPrefix, logData.requestID, logData.UserIDs, message, queryDuration)
	}
}

// GetTimeInRange API function for time in range indicators
// @Summary Get time in range indicators for the given user ids
// @Description Get the api status
// @ID tide-whisperer-api-gettimeinrange
// @Accept json
// @Produce json
// @Param userIds query []string true "List of user ids to fetch" collectionFormat(csv)
// @Param endDate query string false "End date to get indicators" format(dateTime)
// @Security TidepoolAuth
// @Success 200 {array} TirResult
// @Failure 403 {string} string "error description"
// @Failure 500 {string} string "error description"
// @Router /indicators/tir [get]
func (a *API) GetTimeInRange(res http.ResponseWriter, req *http.Request) {
	logInfo := &LoggerInfo{
		requestID:    newRequestID(),
		apiCallStart: time.Now(),
	}
	params, err := store.GetAggParams(req.URL.Query(), &a.schemaVersion)
	if err != nil {
		logIndicatorError(logInfo, "store.GetAggParams", err)
		jsonError(res, errorInvalidParameters, logInfo.apiCallStart)
		return
	}

	logInfo.UserIDs = params.UserIDs
	if !(a.isAuthorized(req, params.UserIDs)) {
		jsonError(res, errorNoViewPermission, logInfo.apiCallStart)
		return
	}

	storageWithCtx := a.store.WithContext(req.Context())

	logInfo.queryStart = time.Now()
	iter, err := storageWithCtx.GetTimeInRangeData(params, false)
	if err != nil {
		logIndicatorError(logInfo, "Mongo Query", err)
		jsonError(res, errorNoViewPermission, logInfo.apiCallStart)
	}
	logIndicatorSlowQuery(logInfo, "GetTimeInRangeData")

	defer iter.Close(req.Context())
	writeMongoJSONResponse(res, req, iter, logInfo)
}
