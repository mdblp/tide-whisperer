package common

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

/*TODO : remove from common once it will not be used by api and usecase*/
type (
	// HttpResponseWriter used for middleware api functions.
	//
	// Use a string builder, so we can send back a valid error response
	// even if an error occurred after the first write
	//
	// - Uppercase fields are for the the functions handlers functions
	//
	// - Lowercase for the middleware (~private)
	HttpResponseWriter struct {
		URL         *url.URL
		VARS        map[string]string
		TraceID     string
		Header      http.Header
		WriteBuffer strings.Builder
		StatusCode  int
		Err         *DetailedError
		Size        int
	}
)

func (res *HttpResponseWriter) Grow(n int) {
	if n > 0 { // Avoid Grow panic()
		res.WriteBuffer.Grow(n)
	} else {
		res.Err = &DetailedError{
			Status:          http.StatusInternalServerError,
			Code:            "write_error",
			Message:         "Internal Server Error",
			InternalMessage: "Grow(): WriteBuffer is nil",
			ID:              res.TraceID,
		}
	}
}

func (res *HttpResponseWriter) Write(v []byte) error {
	size, err := res.WriteBuffer.Write(v)
	res.Size += size
	return err
}

func (res *HttpResponseWriter) WriteString(s string) error {
	size, err := res.WriteBuffer.WriteString(s)
	res.Size += size
	return err
}

// WriteError final writing to the response
func (res *HttpResponseWriter) WriteError(err *DetailedError) error {
	if err == nil {
		err = &DetailedError{
			Status:          http.StatusInternalServerError,
			Code:            "unknown_error",
			Message:         "Unknown error",
			InternalMessage: "WriteError() with nil error",
		}
	}

	res.Err = err
	res.Err.ID = res.TraceID

	// Discard the previous content write, so we ends up with
	// a valid json returned to the client
	res.WriteBuffer.Reset()

	jsonErr, _ := json.Marshal(err)
	res.WriteHeader(err.Status)
	return res.Write(jsonErr)
}

func (res *HttpResponseWriter) WriteHeader(statusCode int) {
	res.StatusCode = statusCode
}
