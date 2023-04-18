package usecase

import (
	"bytes"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/tidepool-org/tide-whisperer/common"
)

var (
	userID                = "userid123"
	traceID               = "traceid123"
	startDate             = "2023-03-05T09:00:00Z"
	endDate               = "2023-04-05T09:00:00Z"
	withPumpSettings      = false
	withParametersChanges = true
	sessionToken          = "sessiontoken123"
	testLogger            = log.New(os.Stdout, "api-test", log.LstdFlags|log.Lshortfile)
	exportArgs            = ExportArgs{
		UserID:                userID,
		TraceID:               traceID,
		StartDate:             startDate,
		EndDate:               endDate,
		WithPumpSettings:      withPumpSettings,
		WithParametersChanges: withParametersChanges,
		SessionToken:          sessionToken,
		BgUnit:                MgdL,
	}
	argsMatcher = mock.MatchedBy(func(args GetDataArgs) bool {
		return args.UserID == userID && args.TraceID == traceID && args.SessionToken == sessionToken &&
			args.WithPumpSettings == withPumpSettings && args.WithParametersHistory == withParametersChanges &&
			args.StartDate == startDate && args.EndDate == endDate && args.BgUnit == MgdL
	})
)

type given struct {
	logger      *log.Logger
	uploader    MockUploader
	patientData MockPatientDataUseCase
	exportArgs  ExportArgs
}

func TestExporter_Export(t *testing.T) {
	tests := []struct {
		name  string
		given given
		then  func(t *testing.T, uploader MockUploader)
	}{
		{
			name:  "should not call uploader when GetData is returning an error",
			given: getDataMockReturningError(),
			then:  assertUploadNotHaveBeenCalled,
		},
		{
			name:  "should call uploader when GetData returns data without error",
			given: getDataMockReturningSuccess(),
			then:  assertUploadHaveBeenCalled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Exporter{
				logger:      tt.given.logger,
				uploader:    &tt.given.uploader,
				patientData: &tt.given.patientData,
			}
			e.Export(tt.given.exportArgs)
			tt.then(t, tt.given.uploader)
		})
	}
}

func assertUploadNotHaveBeenCalled(t *testing.T, uploader MockUploader) {
	uploader.AssertExpectations(t)
}

func assertUploadHaveBeenCalled(t *testing.T, uploader MockUploader) {
	uploader.AssertExpectations(t)
}

func getDataMockReturningError() given {
	getDataErrUseCase := MockPatientDataUseCase{}
	getDataErrUseCase.On("GetData", argsMatcher).Return(nil, &common.DetailedError{})
	return given{
		logger:      testLogger,
		uploader:    MockUploader{},
		patientData: getDataErrUseCase,
		exportArgs:  exportArgs,
	}
}

func getDataMockReturningSuccess() given {
	getDataSuccessUseCase := MockPatientDataUseCase{}
	getDataSuccessUseCase.On("GetData", argsMatcher).Return(&bytes.Buffer{}, nil)
	uploadSuccess := MockUploader{}
	uploadSuccess.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("*bytes.Buffer")).Return(nil)
	return given{
		logger:      testLogger,
		uploader:    uploadSuccess,
		patientData: getDataSuccessUseCase,
		exportArgs:  exportArgs,
	}
}
