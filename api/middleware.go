package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/tidepool-org/tide-whisperer/common"
)

// HandlerLoggerFunc expose our httpResponseWriter API
type HandlerLoggerFunc func(context.Context, *common.HttpResponseWriter) error

// RequestLoggerFunc type to simplify func signatures
type RequestLoggerFunc func(HandlerLoggerFunc) HandlerLoggerFunc

var emptyUserIDs = []string{}

// middleware middleware to log received requests
func (a *API) middleware(fn HandlerLoggerFunc, checkPermissions bool, params ...string) http.HandlerFunc {
	// The mux handler func:
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		start := time.Now().UTC()

		// It is recommended by go to get the request information before writing
		// So get theses now

		logErrors := make([]string, 0, 5)
		logRequest := fmt.Sprintf("%s - %s %s HTTP/%d.%d", r.RemoteAddr, r.Method, r.URL.String(), r.ProtoMajor, r.ProtoMinor)

		// TODO: use x-client-trace-id ?
		// https://docs.solo.io/gloo-edge/latest/guides/observability/tracing/
		traceID := r.Header.Get("x-tidepool-trace-session")
		if !common.IsValidUUID(traceID) {
			// We want a trace id, but for now we do not enforce it
			logErrors = append(logErrors, fmt.Sprintf("no-trace:\"%s\"", traceID))
			traceID = uuid.New().String()
		}

		// Make our context
		ctx := common.TimeItContext(r.Context())

		res := common.HttpResponseWriter{
			Header:     r.Header.Clone(), // Clone the header, to be sure
			URL:        r.URL,
			VARS:       nil,
			TraceID:    traceID,
			StatusCode: http.StatusOK, // Default status
			Err:        nil,
		}

		userIDs := emptyUserIDs
		// The handler have parameters, get them
		if len(params) > 0 {
			res.VARS = mux.Vars(r) // Decode route parameter

			if common.Contains(params, "userID") {
				// userID is a commonly used parameter
				// See if we can view the data
				userID := res.VARS["userID"]
				userIDs = []string{userID}

				if len(userID) > 64 {
					// Quick verification on the userID for security reason
					// Partial but may help without beeing a burden
					// 64 characters is probably a good compromise
					res.WriteError(&common.DetailedError{
						Status:          http.StatusBadRequest,
						Code:            "invalid_userid",
						Message:         "Invalid parameter userId",
						InternalMessage: "userID do not match the regex",
					})
				}
			}
		}

		common.TimeIt(ctx, "checkPermissions")
		if checkPermissions && !a.isAuthorized(r, userIDs) {
			err = res.WriteError(&errorNoViewPermission)
		}
		common.TimeEnd(ctx, "checkPermissions")

		// Mainteners: No read from the request below this point!

		// Make the call to the API function if we can:
		if res.Err == nil {
			err = fn(ctx, &res)
			if err != nil {
				logErrors = append(logErrors, fmt.Sprintf("efn:\"%s\"", err))
			}
		}

		// We will send a JSON, so advertise it for all of our requests
		common.TimeIt(ctx, "writeJSONResults")
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(res.StatusCode)
		_, err = w.Write(res.WriteBuffer.Bytes())
		if err != nil {
			logErrors = append(logErrors, fmt.Sprintf("eww:\"%s\"", err))
		}
		common.TimeEnd(ctx, "writeJSONResults")

		// Log errors management
		if res.Err != nil {
			if res.Err.Code != "" {
				logErrors = append(logErrors, fmt.Sprintf("code:\"%s\"", res.Err.Code))
			}
			if res.Err.InternalMessage != "" {
				logErrors = append(logErrors, fmt.Sprintf("err:\"%s\"", res.Err.InternalMessage))
			}
		}

		// Get the time spent on it
		end := time.Now().UTC()
		dur := end.Sub(start).Milliseconds()
		// Log the message
		var logError string
		if len(logErrors) > 0 {
			logError = fmt.Sprintf("{%s} - ", strings.Join(logErrors, ","))
		}

		timerResults := common.TimeResults(ctx)
		if len(timerResults) > 0 {
			timerResults = fmt.Sprintf("{%s} %d ms", timerResults, dur)
		} else {
			timerResults = fmt.Sprintf("%d ms", dur)
		}
		a.logger.Printf("{%s} %s %d - %s%s - %d bytes", traceID, logRequest, res.StatusCode, logError, timerResults, res.Size)
	}
}
