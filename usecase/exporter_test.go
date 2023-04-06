package usecase

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/tidepool-org/tide-whisperer/common"
)

func TestExporter_Export(t *testing.T) {
	userID := "userid123"
	traceID := "traceid123"
	startDate := "2023-03-05T09:00:00Z"
	endDate := "2023-04-05T09:00:00Z"
	withPumpSettings := true
	sessionToken := "sessiontoken123"
	testLogger := log.New(os.Stdout, "api-test", log.LstdFlags|log.Lshortfile)
	getDataErrUseCase := MockPatientDataUseCase{}
	argsMatcher := mock.MatchedBy(func(args GetDataArgs) bool {
		return args.UserID == userID && args.TraceID == traceID && args.SessionToken == sessionToken &&
			args.WithPumpSettings == withPumpSettings && args.StartDate == startDate && args.EndDate == endDate
	})
	getDataErrUseCase.On("GetData", argsMatcher).Return(&common.DetailedError{})
	getDataSuccessUseCase := MockPatientDataUseCase{}
	getDataSuccessUseCase.On("GetData", argsMatcher).Return(nil)
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
			name: "should not call uploader when FetData failed",
			fields: fields{
				logger:      testLogger,
				uploader:    MockUploader{},
				patientData: getDataErrUseCase,
			},
			exportArgs: ExportArgs{
				UserID:           userID,
				TraceID:          traceID,
				StartDate:        startDate,
				EndDate:          endDate,
				WithPumpSettings: withPumpSettings,
				SessionToken:     sessionToken,
			},
		},
		{
			name: "should not uploader when GetData succeeded",
			fields: fields{
				logger:      testLogger,
				uploader:    uploadSuccess,
				patientData: getDataSuccessUseCase,
			},
			exportArgs: ExportArgs{
				UserID:           userID,
				TraceID:          traceID,
				StartDate:        startDate,
				EndDate:          endDate,
				WithPumpSettings: withPumpSettings,
				SessionToken:     sessionToken,
			},
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
