package usecase

import (
	"bytes"
	"context"
	"log"
	"strings"
	"time"

	"github.com/tidepool-org/tide-whisperer/common"
)

type Exporter struct {
	logger      *log.Logger
	uploader    Uploader
	patientData PatientDataUseCase
}

func NewExporter(logger *log.Logger, patientData PatientDataUseCase, uploader Uploader) Exporter {
	return Exporter{
		logger:      logger,
		uploader:    uploader,
		patientData: patientData,
	}
}

type ExportArgs struct {
	UserID           string
	TraceID          string
	StartDate        string
	EndDate          string
	WithPumpSettings bool
	SessionToken     string
}

func (e Exporter) Export(exportArgs ExportArgs) {
	e.logger.Println("launching export process")
	var buffer bytes.Buffer
	backgroundCtx := common.TimeItContext(context.Background())
	startExportTime := strings.ReplaceAll(time.Now().UTC().Round(time.Second).String(), " ", "_")
	getDataArgs := GetDataArgs{
		Ctx:              backgroundCtx,
		UserID:           exportArgs.UserID,
		TraceID:          exportArgs.TraceID,
		StartDate:        exportArgs.StartDate,
		EndDate:          exportArgs.EndDate,
		WithPumpSettings: exportArgs.WithPumpSettings,
		SessionToken:     exportArgs.SessionToken,
		Buff:             &buffer,
	}
	err := e.patientData.GetData(getDataArgs)
	if err != nil {
		e.logger.Printf("get patient data failed: %v \n", err)
		return
	}

	/*convert data to mgdl if needed*/

	filename := strings.Join([]string{exportArgs.UserID, startExportTime}, "_")
	errUpload := e.uploader.Upload(backgroundCtx, filename, &buffer)
	if errUpload != nil {
		e.logger.Printf("S3 upload failed: %v \n", errUpload)
	}
	e.logger.Println("upload to S3 done with success, terminating go routine")
}
