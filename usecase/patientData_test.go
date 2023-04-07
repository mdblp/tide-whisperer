package usecase

import (
	"bytes"
	"context"
	"log"
	"testing"
	"time"

	"github.com/mdblp/tide-whisperer-v2/v2/client/tidewhisperer"
	"github.com/mdblp/tide-whisperer-v2/v2/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tidepool-org/tide-whisperer/common"
	"github.com/tidepool-org/tide-whisperer/infrastructure"
)

var now = time.Now().UTC().Round(time.Second)
var cbgDay = time.Date(2023, time.April, 1, 0, 0, 0, 0, time.UTC)
var cbgTime = time.Date(2023, time.April, 1, 12, 32, 0, 0, time.UTC)
var oneCbgArray = []schema.CbgBucket{
	{
		Id:                "cbg1",
		CreationTimestamp: now,
		UserId:            "user1",
		Day:               cbgDay,
		Samples: []schema.CbgSample{
			{
				Value:          10,
				Units:          MmolL,
				Timestamp:      cbgTime,
				Timezone:       "UTC",
				TimezoneOffset: 0,
			},
		},
	},
}

var oneCbgResultMgdl = "[\n{\"id\":\"cbg_cbg1_0\",\"time\":\"2023-04-01T12:32:00Z\",\"timezone\":\"UTC\",\"type\":\"cbg\",\"units\":\"mg/dL\",\"value\":6}]\n"
var oneCbgResultMmol = "[\n{\"id\":\"cbg_cbg1_0\",\"time\":\"2023-04-01T12:32:00Z\",\"timezone\":\"UTC\",\"type\":\"cbg\",\"units\":\"mmol/L\",\"value\":10}]\n"
var getDataArgs = GetDataArgs{
	Ctx:              common.TimeItContext(context.Background()),
	UserID:           "user1",
	TraceID:          "trace1",
	StartDate:        "",
	EndDate:          "",
	WithPumpSettings: false,
	SessionToken:     "token1",
	ConvertToMgdl:    false,
}

var getDataArgsWithConversion = GetDataArgs{
	Ctx:              getDataArgs.Ctx,
	UserID:           getDataArgs.UserID,
	TraceID:          getDataArgs.TraceID,
	StartDate:        getDataArgs.StartDate,
	EndDate:          getDataArgs.EndDate,
	WithPumpSettings: getDataArgs.WithPumpSettings,
	SessionToken:     getDataArgs.SessionToken,
	ConvertToMgdl:    true,
}

func TestPatientData_GetData(t *testing.T) {

	patientDataRepository := MockPatientDataRepository{}
	patientDataRepository.On("GetDataInDeviceData", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(
		infrastructure.NewEmptyMockDbAdapterIterator(),
		nil,
	)
	tideV2Client := tidewhisperer.TideWhispererV2MockClient{}
	tideV2Client.MockedCbg = oneCbgArray
	expectedMgdlBuffer := bytes.Buffer{}
	expectedMgdlBuffer.WriteString(oneCbgResultMgdl)
	expectedMmolBuffer := bytes.Buffer{}
	expectedMmolBuffer.WriteString(oneCbgResultMmol)
	type fields struct {
		patientDataRepository PatientDataRepository
		tideV2Client          tidewhisperer.ClientInterface
		logger                *log.Logger
		readBasalBucket       bool
	}
	type args struct {
		args GetDataArgs
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantErr  *common.DetailedError
		wantBuff *bytes.Buffer
	}{
		{
			name: "should return cbg converted to mgdl",
			fields: fields{
				patientDataRepository: &patientDataRepository,
				tideV2Client:          &tideV2Client,
				logger:                &log.Logger{},
				readBasalBucket:       false,
			},
			args: args{
				args: getDataArgsWithConversion,
			},
			wantErr:  nil,
			wantBuff: &expectedMgdlBuffer,
		},
		{
			name: "should return cbg in mmol",
			fields: fields{
				patientDataRepository: &patientDataRepository,
				tideV2Client:          &tideV2Client,
				logger:                &log.Logger{},
				readBasalBucket:       false,
			},
			args: args{
				args: getDataArgs,
			},
			wantErr:  nil,
			wantBuff: &expectedMmolBuffer,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PatientData{
				patientDataRepository: tt.fields.patientDataRepository,
				tideV2Client:          tt.fields.tideV2Client,
				logger:                tt.fields.logger,
				readBasalBucket:       tt.fields.readBasalBucket,
			}
			res, err := p.GetData(tt.args.args)
			assert.Equalf(t, tt.wantErr, err, "GetData error %v not equal to expected %v", err, tt.args.args)
			assert.Equal(t, tt.wantBuff.String(), res.String())
		})
	}
}
