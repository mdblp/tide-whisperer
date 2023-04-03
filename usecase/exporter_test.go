package usecase

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/tidepool-org/tide-whisperer/common"
	"github.com/tidepool-org/tide-whisperer/usecase/mocks"
)

func TestExporter_Export(t *testing.T) {
	userID := "userid123"
	traceID := "traceid123"
	startDate := "2023-03-05T09:00:00Z"
	endDate := "2023-04-05T09:00:00Z"
	withPumpSettings := true
	sessionToken := "sessiontoken123"
	testLogger := log.New(os.Stdout, "api-test", log.LstdFlags|log.Lshortfile)
	getDataErrUseCase := mocks.PatientDataUseCase{}
	getDataErrUseCase.On("GetData", mock.Anything, userID, traceID, startDate, endDate, withPumpSettings, sessionToken, mock.Anything).Return(&common.DetailedError{})
	getDataSuccessUseCase := mocks.PatientDataUseCase{}
	getDataSuccessUseCase.On("GetData", mock.Anything, userID, traceID, startDate, endDate, withPumpSettings, sessionToken, mock.Anything).Return(nil)
	uploadSuccess := mocks.Uploader{}
	uploadSuccess.On("Upload", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	type fields struct {
		logger      *log.Logger
		uploader    mocks.Uploader
		patientData mocks.PatientDataUseCase
	}
	type args struct {
		userID           string
		traceID          string
		startDate        string
		endDate          string
		withPumpSettings bool
		sessionToken     string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "should not call uploader when FetData failed",
			fields: fields{
				logger:      testLogger,
				uploader:    mocks.Uploader{},
				patientData: getDataErrUseCase,
			},
			args: args{
				userID:           userID,
				traceID:          traceID,
				startDate:        startDate,
				endDate:          endDate,
				withPumpSettings: withPumpSettings,
				sessionToken:     sessionToken,
			},
		},
		{
			name: "should not uploader when GetData succeeded",
			fields: fields{
				logger:      testLogger,
				uploader:    uploadSuccess,
				patientData: getDataSuccessUseCase,
			},
			args: args{
				userID:           userID,
				traceID:          traceID,
				startDate:        startDate,
				endDate:          endDate,
				withPumpSettings: withPumpSettings,
				sessionToken:     sessionToken,
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
			e.Export(tt.args.userID, tt.args.traceID, tt.args.startDate, tt.args.endDate, tt.args.withPumpSettings, tt.args.sessionToken)
			tt.fields.patientData.AssertExpectations(t)
			tt.fields.uploader.AssertExpectations(t)
		})
	}
}
