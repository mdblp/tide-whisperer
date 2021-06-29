package data

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/tidepool-org/go-common/clients/mongo"
	"github.com/tidepool-org/tide-whisperer/store"
)

type (

	// errorCounter to record only the first error to avoid spamming the log and takes too much time
	errorCounter struct {
		firstError error
		numErrors  int
	}
	// writeFromIter struct to pass to the function which write the http result from the mongo iterator for diabetes data
	writeFromIter struct {
		res  *httpResponseWriter
		iter mongo.StorageIterator
		// parametersHistory fetched from portal database
		parametersHistory map[string]interface{}
		// uploadIDs encountered during the operation
		uploadIDs []string
		// writeCount the number of data written
		writeCount int
		// datum decode errors
		decode errorCounter
		// datum JSON marshall errors
		jsonError errorCounter
	}
)

// Parameters level to keep in api response
var parameterLevelFilter = [...]int{1, 2}

func (a *API) setHandlesV1(prefix string, rtr *mux.Router) {
	// rtr.HandleFunc(prefix+"/status", a.requestLogger(a.getStatus)).Methods("GET")
	rtr.HandleFunc(prefix+"/range/{userID}", a.middlewareV1(a.getRangeV1, "userID")).Methods("GET")
	rtr.HandleFunc(prefix+"/data/{userID}", a.middlewareV1(a.getDataV1, "userID")).Methods("GET")
	rtr.HandleFunc(prefix+"/{.*}", a.middlewareV1(a.getNotFoundV1)).Methods("GET")
}

// getNotFoundV1 should it be version free?
func (a *API) getNotFoundV1(ctx context.Context, res *httpResponseWriter) error {
	res.WriteHeader(http.StatusNotFound)
	return nil
}

// @Summary Get the data dates range for a specific patient
//
// @Description Get the data dates range for a specific patient, returning a JSON array of two ISO 8601 strings: ["2021-01-01T10:00:00.430Z", "2021-02-10T10:18:27.430Z"]
//
// @ID tide-whisperer-api-v1-getrange
// @Produce json
// @Success 200 {array} string "Array of two ISO 8601 datetime"
// @Failure 400 {object} data.detailedError
// @Failure 403 {object} data.detailedError
// @Failure 404 {object} data.detailedError
// @Failure 500 {object} data.detailedError
// @Param userID path string true "The ID of the user to search data for"
// @Param x-tidepool-trace-session header string false "Trace session uuid" format(uuid)
// @Security TidepoolAuth
// @Router /v1/range/{userID} [get]
func (a *API) getRangeV1(ctx context.Context, res *httpResponseWriter) error {
	userID := res.VARS["userID"]

	dates, err := a.store.GetDataRangeV1(ctx, res.TraceID, userID)
	if err != nil {
		logError := &detailedError{
			Status:          errorRunningQuery.Status,
			Code:            errorRunningQuery.Code,
			Message:         errorRunningQuery.Message,
			InternalMessage: err.Error(),
		}
		return res.WriteError(logError)
	}

	if dates.Start == "" || dates.End == "" {
		return res.WriteError(&errorNotfound)
	}

	result := make([]string, 2)
	result[0] = dates.Start
	result[1] = dates.End

	jsonResult, err := json.Marshal(result)
	if err != nil {
		logError := &detailedError{
			Status:          http.StatusInternalServerError,
			Code:            "json_marshall_error",
			Message:         "internal server error",
			InternalMessage: err.Error(),
		}
		return res.WriteError(logError)
	}

	return res.Write(jsonResult)
}

// @Summary Get the data for a specific patient
//
// @Description Get the data for a specific patient, returning a JSON array of objects
//
// @ID tide-whisperer-api-v1-getdata
// @Produce json
//
// @Success 200 {array} string "Array of objects"
// @Failure 400 {object} data.detailedError
// @Failure 403 {object} data.detailedError
// @Failure 404 {object} data.detailedError
// @Failure 500 {object} data.detailedError
//
// @Param userID path string true "The ID of the user to search data for"
//
// @Param startDate query string false "ISO Date time (RFC3339) for search lower limit" format(date-time)
//
// @Param endDate query string false "ISO Date time (RFC3339) for search upper limit" format(date-time)
//
// @Param withPumpSettings query string false "true to include the pump settings in the results" format(boolean)
//
// @Param x-tidepool-trace-session header string false "Trace session uuid" format(uuid)
// @Security TidepoolAuth
//
// @Router /v1/data/{userID} [get]
func (a *API) getDataV1(ctx context.Context, res *httpResponseWriter) error {
	var err error
	// Mongo iterators
	var iterData mongo.StorageIterator
	var iterPumpSettings mongo.StorageIterator
	var iterUploads mongo.StorageIterator

	userID := res.VARS["userID"]

	query := res.URL.Query()
	startDate := query.Get("startDate")
	endDate := query.Get("endDate")
	withPumpSettings := query.Get("withPumpSettings") == "true"

	// Check startDate & endDate parameter
	if startDate != "" || endDate != "" {
		var logError *detailedError
		var startTime time.Time
		var endTime time.Time
		var timeRange int64 = 1 // endDate - startDate in seconds, initialized to 1 to avoid trigger an error, see below

		if startDate != "" {
			startTime, err = time.Parse(time.RFC3339Nano, startDate)
		}
		if err == nil && endDate != "" {
			endTime, err = time.Parse(time.RFC3339Nano, endDate)
		}

		if err == nil && startDate != "" && endDate != "" {
			timeRange = endTime.Unix() - startTime.Unix()
		}

		if timeRange > 0 {
			// Make an estimated guessed about the amount of data we need to send
			// to help our buffer, since we may send ten or so megabytes of JSON
			// I saw ~ 1.15 byte per second in my test
			// fmt.Printf("Grow: %d * 1.15 -> %d\n", timeRange, int(math.Round(float64(timeRange)*1.15)))
			res.Grow(int(math.Round(float64(timeRange) * 1.15)))
		} else {
			err = fmt.Errorf("startDate is after endDate")
		}

		if err != nil {
			logError = &detailedError{
				Status:          errorInvalidParameters.Status,
				Code:            errorInvalidParameters.Code,
				Message:         errorInvalidParameters.Message,
				InternalMessage: err.Error(),
			}
			return res.WriteError(logError)
		}
	}

	dates := &store.Date{
		Start: startDate,
		End:   endDate,
	}

	writeParams := &writeFromIter{
		res:       res,
		uploadIDs: make([]string, 0, 16),
	}

	if withPumpSettings {
		// Initial query to fetch for this user, the client wants the
		// latest pumpSettings
		iterPumpSettings, err = a.store.GetLatestPumpSettingsV1(ctx, res.TraceID, userID)
		if err != nil {
			logError := &detailedError{
				Status:          errorRunningQuery.Status,
				Code:            errorRunningQuery.Code,
				Message:         errorRunningQuery.Message,
				InternalMessage: err.Error(),
			}
			return res.WriteError(logError)
		}
		defer iterPumpSettings.Close(ctx)
		// Fetch parameters history from portal:
		writeParams.parametersHistory, err = a.store.GetDiabeloopParametersHistory(ctx, userID, parameterLevelFilter[:])
		if err != nil {
			// Just log the problem, don't crash the query
			writeParams.parametersHistory = nil
			a.logger.Printf("{%s} - {GetDiabeloopParametersHistory:\"%s\"}", res.TraceID, err)
		}
	}

	// Fetch normal data:
	iterData, err = a.store.GetDataV1(ctx, res.TraceID, userID, dates)
	if err != nil {
		logError := &detailedError{
			Status:          errorRunningQuery.Status,
			Code:            errorRunningQuery.Code,
			Message:         errorRunningQuery.Message,
			InternalMessage: err.Error(),
		}
		return res.WriteError(logError)
	}
	defer iterData.Close(ctx)

	// We return a JSON array, first charater is: '['
	err = res.WriteString("[\n")
	if err != nil {
		return err
	}

	if withPumpSettings && iterPumpSettings != nil {
		writeParams.iter = iterPumpSettings
		err = writeFromIterV1(ctx, writeParams)
		if err != nil {
			return err
		}
	}

	writeParams.iter = iterData
	err = writeFromIterV1(ctx, writeParams)
	if err != nil {
		return err
	}

	// Fetch uploads
	if len(writeParams.uploadIDs) > 0 {
		iterUploads, err = a.store.GetDataFromIDV1(ctx, res.TraceID, writeParams.uploadIDs)
		if err != nil {
			// Just log the problem, don't crash the query
			writeParams.parametersHistory = nil
			a.logger.Printf("{%s} - {GetDataFromIDV1:\"%s\"}", res.TraceID, err)
		} else {
			defer iterUploads.Close(ctx)
			writeParams.iter = iterUploads
			err = writeFromIterV1(ctx, writeParams)
			if err != nil {
				return err
			}
		}
	}

	// Silently failed theses error to the client, but record them to the log
	if writeParams.decode.firstError != nil {
		a.logger.Printf("{%s} - {nErrors:%d,MongoDecode:\"%s\"}", res.TraceID, writeParams.decode.numErrors, writeParams.decode.firstError)
	}
	if writeParams.jsonError.firstError != nil {
		a.logger.Printf("{%s} - {nErrors:%d,jsonMarshall:\"%s\"}", res.TraceID, writeParams.jsonError.numErrors, writeParams.jsonError.firstError)
	}

	// Last JSON array charater:
	return res.WriteString("]\n")
}

// writeFromIterV1 Common code to write
func writeFromIterV1(ctx context.Context, p *writeFromIter) error {
	var err error

	iter := p.iter
	p.iter = nil

	for iter.Next(ctx) {
		var jsonDatum []byte
		var datum map[string]interface{}

		err = iter.Decode(&datum)
		if err != nil {
			p.decode.numErrors++
			if p.decode.firstError == nil {
				p.decode.firstError = err
			}
			continue
		}
		if len(datum) > 0 {
			datumID, haveID := datum["id"].(string)
			if !haveID {
				// Ignore datum with no id, should never happend
				continue
			}
			datumType, haveType := datum["type"].(string)
			if !haveType {
				// Ignore datum with no type, should never happend
				continue
			}
			uploadID, haveUploadID := datum["uploadId"].(string)
			if !haveUploadID {
				// No upload ID, abnormal situation
				continue
			}
			if datumType == "deviceEvent" {
				datumSubType, haveSubType := datum["subType"].(string)
				if haveSubType && datumSubType == "deviceParameter" {
					datumLevel, haveLevel := datum["level"]
					if haveLevel {
						intLevel, err := strconv.Atoi(fmt.Sprintf("%v", datumLevel))
						if err == nil && !containsInt(parameterLevelFilter[:], intLevel) {
							continue
						}
					}
				}
			}
			// Record the uploadID
			if !(datumType == "upload" && uploadID == datumID) {
				if !contains(p.uploadIDs, uploadID) {
					p.uploadIDs = append(p.uploadIDs, uploadID)
				}
			}
			// Add the parameter history to the pump settings
			if datumType == "pumpSettings" && p.parametersHistory != nil {
				payload := datum["payload"].(map[string]interface{})
				payload["history"] = p.parametersHistory["history"]
				datum["payload"] = payload
			}
			// Create the JSON string for this datum
			if jsonDatum, err = json.Marshal(datum); err != nil {
				if p.jsonError.firstError == nil {
					p.jsonError.firstError = err
				}
				p.jsonError.numErrors++
				continue
			}

			if p.writeCount > 0 {
				// Add the coma and line return (for readability)
				err = p.res.WriteString(",\n")
				if err != nil {
					return err
				}
			}
			err = p.res.Write(jsonDatum)
			if err != nil {
				return err
			}
			p.writeCount++
		} // else ignore
	}
	return nil
}
