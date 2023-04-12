package usecase

import (
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
	UserID                string
	TraceID               string
	StartDate             string
	EndDate               string
	WithPumpSettings      bool
	WithParametersChanges bool
	SessionToken          string
	ConvertToMgdl         bool
}

func (e Exporter) Export(args ExportArgs) {
	e.logger.Println("launching export process")
	backgroundCtx := common.TimeItContext(context.Background())
	startExportTime := strings.ReplaceAll(time.Now().UTC().Round(time.Second).String(), " ", "_")
	getDataArgs := GetDataArgs{
		Ctx:                   backgroundCtx,
		UserID:                args.UserID,
		TraceID:               args.TraceID,
		StartDate:             args.StartDate,
		EndDate:               args.EndDate,
		WithPumpSettings:      args.WithPumpSettings,
		WithParametersChanges: args.WithParametersChanges,
		SessionToken:          args.SessionToken,
		ConvertToMgdl:         args.ConvertToMgdl,
	}
	buffer, err := e.patientData.GetData(getDataArgs)
	if err != nil {
		e.logger.Printf("get patient data failed: %v \n", err)
		return
	}

	filename := strings.Join([]string{args.UserID, startExportTime}, "_")
	errUpload := e.uploader.Upload(backgroundCtx, filename, buffer)
	if errUpload != nil {
		e.logger.Printf("S3 upload failed: %v \n", errUpload)
	}
	e.logger.Println("upload to S3 done with success, terminating go routine")
}
