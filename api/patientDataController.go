package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/tidepool-org/tide-whisperer/common"
)

// @Summary Get the data for a specific patient using new bucket api
// @Description Get the data for a specific patient, returning a JSON array of objects
// @ID tide-whisperer-api-v1V2-getdata
// @Produce json
// @Success 200 {array} string "Array of objects"
// @Failure 400 {object} common.DetailedError
// @Failure 403 {object} common.DetailedError
// @Failure 404 {object} common.DetailedError
// @Failure 500 {object} common.DetailedError
// @Param userID path string true "The ID of the user to search data for"
// @Param startDate query string false "ISO Date time (RFC3339) for search lower limit" format(date-time)
// @Param endDate query string false "ISO Date time (RFC3339) for search upper limit" format(date-time)
// @Param withPumpSettings query string false "true to include the pump settings in the results" format(boolean)
// @Param cbgBucket query string false "no parameter or not equal to true to get cbg from buckets" format(boolean)
// @Param basalBucket query string false "true to get basals from buckets, if the parameter is not there or not equal to true the basals are from deviceData" format(boolean)
// @Param x-tidepool-trace-session header string false "Trace session uuid" format(uuid)
// @Security TidepoolAuth
// @Router /v1/dataV2/{userID} [get]
func (a *API) getDataV2(ctx context.Context, res *common.HttpResponseWriter) error {
	var buffer bytes.Buffer
	// Mongo iterators
	userID := res.VARS["userID"]

	query := res.URL.Query()
	startDate := query.Get("startDate")
	endDate := query.Get("endDate")
	withPumpSettings := query.Get("withPumpSettings") == "true"
	err := a.patientData.GetData(ctx, userID, res.TraceID, startDate, endDate, withPumpSettings, a.readBasalBucket, &buffer, res)
	if err == nil {
		res.Write(buffer.Bytes())
	}
	return err
}

// @Summary Get the data for a specific patient. Deprecated, this route will be deleted in the future and be replaced
// by tide-whisperer-api-v1V2-getdata
// @Description Get the data for a specific patient, returning a JSON array of objects
// @ID tide-whisperer-api-v1-getdata
// @Produce json
// @Success 200 {array} string "Array of objects"
// @Failure 400 {object} common.DetailedError
// @Failure 403 {object} common.DetailedError
// @Failure 404 {object} common.DetailedError
// @Failure 500 {object} common.DetailedError
// @Param userID path string true "The ID of the user to search data for"
// @Param startDate query string false "ISO Date time (RFC3339) for search lower limit" format(date-time)
// @Param endDate query string false "ISO Date time (RFC3339) for search upper limit" format(date-time)
// @Param withPumpSettings query string false "true to include the pump settings in the results" format(boolean)
// @Param cbgBucket query string false "no parameter or not equal to true to get cbg from buckets" format(boolean)
// @Param basalBucket query string false "true to get basals from buckets, if the parameter is not there or not equal to true the basals are from deviceData" format(boolean)
// @Param x-tidepool-trace-session header string false "Trace session uuid" format(uuid)
// @Security TidepoolAuth
// @Router /v1/data/{userID} [get]
func (a *API) getData(ctx context.Context, res *common.HttpResponseWriter) error {
	return a.getDataV2(ctx, res)
}

// @Summary Get the data dates range for a specific patient
//
// @Description Get the data dates range for a specific patient, returning a JSON array of two ISO 8601 strings: ["2021-01-01T10:00:00.430Z", "2021-02-10T10:18:27.430Z"]
//
// @ID tide-whisperer-api-v1-getrange
// @Produce json
// @Success 200 {array} string "Array of two ISO 8601 datetime"
// @Failure 400 {object} common.DetailedError
// @Failure 403 {object} common.DetailedError
// @Failure 404 {object} common.DetailedError
// @Failure 500 {object} common.DetailedError
// @Param userID path string true "The ID of the user to search data for"
// @Param x-tidepool-trace-session header string false "Trace session uuid" format(uuid)
// @Security TidepoolAuth
// @Router /v1/range/{userID} [get]
// Deprecated: not removed for backward compatibility but should not be used
func (a *API) getRangeLegacy(ctx context.Context, res *common.HttpResponseWriter) error {
	userID := res.VARS["userID"]

	dates, err := a.patientData.GetDataRangeLegacy(ctx, res.TraceID, userID)
	if err != nil {
		logError := &common.DetailedError{
			Status:          errorRunningQuery.Status,
			Code:            errorRunningQuery.Code,
			Message:         errorRunningQuery.Message,
			InternalMessage: err.Error(),
		}
		return res.WriteError(logError)
	}

	if dates == nil || dates.Start == "" || dates.End == "" {
		return res.WriteError(&errorNotfound)
	}

	result := make([]string, 2)
	result[0] = dates.Start
	result[1] = dates.End

	jsonResult, err := json.Marshal(result)
	if err != nil {
		logError := &common.DetailedError{
			Status:          http.StatusInternalServerError,
			Code:            "json_marshall_error",
			Message:         "internal server error",
			InternalMessage: err.Error(),
		}
		return res.WriteError(logError)
	}

	return res.Write(jsonResult)
}
