package api

import (
	"context"
	"log"

	"github.com/tidepool-org/tide-whisperer/common"
	"github.com/tidepool-org/tide-whisperer/usecase"
)

type ExportController struct {
	logger   *log.Logger
	exporter ExporterUseCase
}

func NewExportController(logger *log.Logger, exporter ExporterUseCase) ExportController {
	return ExportController{
		logger:   logger,
		exporter: exporter,
	}
}

// ExportData
// @Summary Export patient data to S3 file.
// @Description Export patient data to a file stored on S3.
// This operation is asynchronous and always returning 200.
// @ID tide-whisperer-export
// @Produce json
// @Success 200
// @Failure 403 {object} common.DetailedError
// @Failure 404 {object} common.DetailedError
// @Param userID path string true "The ID of the user to search data for"
// @Param startDate query string false "ISO Date time (RFC3339) for search lower limit" format(date-time)
// @Param endDate query string false "ISO Date time (RFC3339) for search upper limit" format(date-time)
// @Param withPumpSettings query string false "true to include the pump settings in the results" format(boolean)
// @Param x-tidepool-trace-session header string false "Trace session uuid" format(uuid)
// @Security Auth0
// @Router /export/{userID} [get]
func (c ExportController) ExportData(ctx context.Context, res *common.HttpResponseWriter) error {
	// Mongo iterators
	userID := res.VARS["userID"]
	query := res.URL.Query()
	startDate := query.Get("startDate")
	endDate := query.Get("endDate")
	withPumpSettings := query.Get("withPumpSettings") == "true"

	sessionToken := getSessionToken(res)
	exportArgs := usecase.ExportArgs{
		UserID:           userID,
		TraceID:          res.TraceID,
		StartDate:        startDate,
		EndDate:          endDate,
		WithPumpSettings: withPumpSettings,
		SessionToken:     sessionToken,
	}
	go c.exporter.Export(exportArgs)
	return nil
}
