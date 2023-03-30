package api

import (
	"bytes"
	"context"
	"log"
	"strings"
	"time"

	"github.com/tidepool-org/tide-whisperer/common"
)

type ExportController struct {
	logger          *log.Logger
	uploader        UploaderUseCase
	useCase         PatientDataUseCase
	readBasalBucket bool
}

func NewExportController(logger *log.Logger, uploader UploaderUseCase, useCase PatientDataUseCase, readBasalBucket bool) ExportController {
	return ExportController{
		logger:          logger,
		uploader:        uploader,
		useCase:         useCase,
		readBasalBucket: readBasalBucket,
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
	var exportBuffer bytes.Buffer
	// Mongo iterators
	userID := res.VARS["userID"]

	query := res.URL.Query()
	startDate := query.Get("startDate")
	endDate := query.Get("endDate")
	withPumpSettings := query.Get("withPumpSettings") == "true"
	sessionToken := getSessionToken(res)
	//TODO verify if there is not an already existing export ongoing
	// if export is already ongoing just skip everything and return
	// if nothing is ongoing run a new export as go func
	go func() {
		backgroundCtx := common.TimeItContext(context.Background())
		c.logger.Println("launching export process")
		//TODO update status to ongoing
		startExportTime := time.Now().UTC().Round(time.Second).String()
		err := c.useCase.GetData(backgroundCtx, userID, res.TraceID, startDate, endDate, withPumpSettings, c.readBasalBucket, sessionToken, &exportBuffer)
		if err != nil {
			c.logger.Printf("get patient data failed: %v \n", err)
			//TODO update status to fail with getData error details
		}
		filename := strings.Join([]string{userID, startExportTime}, "_")
		errUpload := c.uploader.Upload(backgroundCtx, filename, &exportBuffer)
		if errUpload != nil {
			//TODO update status to fail with s3 error details
			c.logger.Printf("S3 upload failed: %v \n", errUpload)
		}
		//TODO update status to success
		c.logger.Println("upload to S3 done with success, terminating go routine")
	}()
	return nil
}
