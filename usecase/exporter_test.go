package usecase

import (
	"bytes"
	"log"
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
	testLogger            = log.New(&bytes.Buffer{}, "", 0)
	exportArgsFormatJSON  = ExportArgs{
		UserID:                userID,
		TraceID:               traceID,
		StartDate:             startDate,
		EndDate:               endDate,
		WithPumpSettings:      withPumpSettings,
		WithParametersChanges: withParametersChanges,
		SessionToken:          sessionToken,
		BgUnit:                MgdL,
		FormatToCsv:           false,
	}
	exportArgsFormatCsv = ExportArgs{
		UserID:                userID,
		TraceID:               traceID,
		StartDate:             startDate,
		EndDate:               endDate,
		WithPumpSettings:      withPumpSettings,
		WithParametersChanges: withParametersChanges,
		SessionToken:          sessionToken,
		BgUnit:                MgdL,
		FormatToCsv:           true,
	}
	argsMatcher = mock.MatchedBy(func(args GetDataArgs) bool {
		return args.UserID == userID && args.TraceID == traceID && args.SessionToken == sessionToken &&
			args.WithPumpSettings == withPumpSettings && args.WithParametersHistory == withParametersChanges &&
			args.StartDate == startDate && args.EndDate == endDate && args.BgUnit == MgdL
	})
)

type given struct {
	logger      *log.Logger
	uploader    Uploader
	patientData PatientDataUseCase
	exportArgs  ExportArgs
}

func TestExporter_Export(t *testing.T) {
	tests := []struct {
		name  string
		given []func(given) given
		/*No expect because there is no output for this function.
		We're just checking mock have been called accordingly*/
	}{
		{
			name: "should not call uploader when GetData failed",
			given: []func(given) given{
				getDataUseCaseError,
				emptyMockUploader,
			},
		},
		{
			name: "should call uploader when GetData returns valid JSON",
			given: []func(given) given{
				getDataUseCaseSuccessValidJSON,
				successUploader,
			},
		},
		{
			name: "should not call uploader when GetData returns invalid json and formatToCsv is true",
			given: []func(given) given{
				formatToCsvTrue,
				getDataUseCaseSuccessInvalidJSON,
				emptyMockUploader,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			given := buildEmptyGiven()
			for _, f := range tt.given {
				given = f(given)
			}
			e := Exporter{
				logger:      given.logger,
				uploader:    given.uploader,
				patientData: given.patientData,
			}
			e.Export(given.exportArgs)
			given.uploader.(*MockUploader).AssertExpectations(t)
		})
	}
}

func getDataUseCaseError(g given) given {
	patientData := MockPatientDataUseCase{}
	patientData.On("GetData", argsMatcher).Return(nil, &common.DetailedError{})
	g.patientData = &patientData
	return g
}
func getDataUseCaseSuccessValidJSON(g given) given {
	patientData := MockPatientDataUseCase{}
	patientData.On("GetData", argsMatcher).Return(bytes.NewBufferString(`{"foo": "bar"}`), nil)
	g.patientData = &patientData
	return g
}
func getDataUseCaseSuccessInvalidJSON(g given) given {
	patientData := MockPatientDataUseCase{}
	patientData.On("GetData", argsMatcher).Return(bytes.NewBufferString(`{"foo": invalid}`), nil)
	g.patientData = &patientData
	return g
}
func emptyMockUploader(g given) given {
	uploader := MockUploader{}
	g.uploader = &uploader
	return g
}
func successUploader(g given) given {
	uploadSuccess := MockUploader{}
	uploadSuccess.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("*bytes.Buffer")).Return(nil)
	g.uploader = &uploadSuccess
	return g
}

func formatToCsvTrue(g given) given {
	g.exportArgs.FormatToCsv = true
	return g
}

func buildEmptyGiven() given {
	return given{
		logger:      testLogger,
		uploader:    &MockUploader{},
		patientData: &MockPatientDataUseCase{},
		exportArgs:  exportArgsFormatJSON,
	}
}
