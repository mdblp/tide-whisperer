package usecase

import (
	"bytes"
	"context"
	"fmt"
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
	BgUnit                string
	FormatToCsv           bool
}

func (e Exporter) Export(args ExportArgs) {
	e.logger.Println("launching export process")
	backgroundCtx := common.TimeItContext(context.Background())
	exportTime := time.Now().UTC().Format("2006-01-02T15:04:05")
	filename := strings.Join([]string{args.UserID, exportTime}, "_")
	getDataArgs := GetDataArgs{
		UserID:                     args.UserID,
		TraceID:                    args.TraceID,
		StartDate:                  args.StartDate,
		EndDate:                    args.EndDate,
		WithPumpSettings:           args.WithPumpSettings,
		WithParametersHistory:      args.WithParametersChanges,
		SessionToken:               args.SessionToken,
		BgUnit:                     args.BgUnit,
		FilteringParametersHistory: true,
	}
	buffer, err := e.patientData.GetData(backgroundCtx, getDataArgs)
	if err != nil {
		e.logger.Printf("get patient data failed: %v \n", err)
		return
	}

	finalBuffer := buffer

	/*Transform to CSV */
	if args.FormatToCsv {
		var csvBuffer *bytes.Buffer
		var csvErr error
		if csvBuffer, csvErr = jsonToCsv(buffer.String()); csvErr != nil {
			e.logger.Printf("jsonToCsv failed: %v \n", csvErr)
			return
		}
		finalBuffer = csvBuffer
		filename = fmt.Sprintf("%s.csv", filename)
	} else {
		filename = fmt.Sprintf("%s.json", filename)
	}

	errUpload := e.uploader.Upload(backgroundCtx, filename, finalBuffer)
	if errUpload != nil {
		e.logger.Printf("S3 upload failed: %v \n", errUpload)
	}
	e.logger.Println("upload to S3 done with success, terminating go routine")
}
