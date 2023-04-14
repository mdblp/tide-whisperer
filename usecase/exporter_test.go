package usecase

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/tidepool-org/tide-whisperer/common"
)

func TestExporter_Export(t *testing.T) {
	exportArgsNoCsv := ExportArgs{
		UserID:                "userid123",
		TraceID:               "traceid123",
		StartDate:             "2023-03-05T09:00:00Z",
		EndDate:               "2023-04-05T09:00:00Z",
		WithPumpSettings:      false,
		WithParametersChanges: true,
		SessionToken:          "sessiontoken123",
		ConvertToMgdl:         true,
	}
	exportArgsCsv := ExportArgs{
		UserID:                exportArgsNoCsv.UserID,
		TraceID:               exportArgsNoCsv.TraceID,
		StartDate:             exportArgsNoCsv.StartDate,
		EndDate:               exportArgsNoCsv.EndDate,
		WithPumpSettings:      exportArgsNoCsv.WithPumpSettings,
		WithParametersChanges: exportArgsNoCsv.WithParametersChanges,
		SessionToken:          exportArgsNoCsv.SessionToken,
		ConvertToMgdl:         exportArgsNoCsv.ConvertToMgdl,
		FormatToCsv:           true,
	}
	testLogger := log.New(&bytes.Buffer{}, "", 0)
	getDataErrUseCase := MockPatientDataUseCase{}
	argsMatcher := mock.MatchedBy(func(args GetDataArgs) bool {
		return args.UserID == exportArgsNoCsv.UserID && args.TraceID == exportArgsNoCsv.TraceID && args.SessionToken == exportArgsNoCsv.SessionToken &&
			args.WithPumpSettings == exportArgsNoCsv.WithPumpSettings && args.WithParametersChanges == exportArgsNoCsv.WithParametersChanges &&
			args.StartDate == exportArgsNoCsv.StartDate && args.EndDate == exportArgsNoCsv.EndDate && args.ConvertToMgdl == exportArgsNoCsv.ConvertToMgdl
	})
	getDataErrUseCase.On("GetData", argsMatcher).Return(nil, &common.DetailedError{})
	getDataSuccessUseCase := MockPatientDataUseCase{}
	getDataSuccessUseCase.On("GetData", argsMatcher).Return(bytes.NewBufferString(`{"foo": "bar"}`), nil)
	getDataSuccessWrongJsonUseCase := MockPatientDataUseCase{}
	getDataSuccessWrongJsonUseCase.On("GetData", argsMatcher).Return(bytes.NewBufferString(`{"foo": ERROR}`), nil)
	uploadSuccess := MockUploader{}
	uploadSuccess.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("*bytes.Buffer")).Return(nil)

	type fields struct {
		logger      *log.Logger
		uploader    MockUploader
		patientData MockPatientDataUseCase
	}
	tests := []struct {
		name       string
		fields     fields
		exportArgs ExportArgs
	}{
		{
			name: "should not call uploader when GetData failed",
			fields: fields{
				logger:      testLogger,
				uploader:    MockUploader{},
				patientData: getDataErrUseCase,
			},
			exportArgs: exportArgsNoCsv,
		},
		{
			name: "should call uploader when GetData succeeded",
			fields: fields{
				logger:      testLogger,
				uploader:    uploadSuccess,
				patientData: getDataSuccessUseCase,
			},
			exportArgs: exportArgsNoCsv,
		},
		{
			name: "should not call uploader when GetData succeeded but json cannot be converted to csv",
			fields: fields{
				logger:      testLogger,
				uploader:    MockUploader{},
				patientData: getDataSuccessWrongJsonUseCase,
			},
			exportArgs: exportArgsCsv,
		},
		{
			name: "should call uploader when GetData succeeded and json can be converted to csv",
			fields: fields{
				logger:      testLogger,
				uploader:    uploadSuccess,
				patientData: getDataSuccessUseCase,
			},
			exportArgs: exportArgsCsv,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Exporter{
				logger:      tt.fields.logger,
				uploader:    &tt.fields.uploader,
				patientData: &tt.fields.patientData,
			}
			e.Export(tt.exportArgs)
			tt.fields.patientData.AssertExpectations(t)
			tt.fields.uploader.AssertExpectations(t)
		})
	}
}
