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

func (e Exporter) Export(userID string, traceID string, startDate string, endDate string, withPumpSettings bool, sessionToken string) {
	e.logger.Println("launching export process")
	var buffer bytes.Buffer
	backgroundCtx := common.TimeItContext(context.Background())
	startExportTime := strings.ReplaceAll(time.Now().UTC().Round(time.Second).String(), " ", "_")
	err := e.patientData.GetData(backgroundCtx, userID, traceID, startDate, endDate, withPumpSettings, sessionToken, &buffer)
	if err != nil {
		e.logger.Printf("get patient data failed: %v \n", err)
		return
	}
	filename := strings.Join([]string{userID, startExportTime}, "_")
	errUpload := e.uploader.Upload(backgroundCtx, filename, &buffer)
	if errUpload != nil {
		e.logger.Printf("S3 upload failed: %v \n", errUpload)
	}
	e.logger.Println("upload to S3 done with success, terminating go routine")
}
