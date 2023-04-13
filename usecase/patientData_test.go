package usecase

import (
	"bytes"
	"context"
	"log"
	"testing"
	"time"

	orcaSchema "github.com/mdblp/orca/schema"
	"github.com/mdblp/tide-whisperer-v2/v2/client/tidewhisperer"
	tideV2Schema "github.com/mdblp/tide-whisperer-v2/v2/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tidepool-org/tide-whisperer/common"
	"github.com/tidepool-org/tide-whisperer/infrastructure"
	"github.com/tidepool-org/tide-whisperer/schema"
)

var now = time.Now().UTC().Round(time.Second)

func TestPatientData_GetData(t *testing.T) {
	testUserId := "testUserId"
	cbgDay := time.Date(2023, time.April, 1, 0, 0, 0, 0, time.UTC)
	cbgTime := time.Date(2023, time.April, 1, 12, 32, 0, 0, time.UTC)
	oneCbgArray := []tideV2Schema.CbgBucket{
		{
			Id:                "cbg1",
			CreationTimestamp: now,
			UserId:            testUserId,
			Day:               cbgDay,
			Samples: []tideV2Schema.CbgSample{
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

	oneCbgResultMgdl := "[\n{\"id\":\"cbg_cbg1_0\",\"time\":\"2023-04-01T12:32:00Z\",\"timezone\":\"UTC\",\"type\":\"cbg\",\"units\":\"mg/dL\",\"value\":6}]\n"
	oneCbgResultMmol := "[\n{\"id\":\"cbg_cbg1_0\",\"time\":\"2023-04-01T12:32:00Z\",\"timezone\":\"UTC\",\"type\":\"cbg\",\"units\":\"mmol/L\",\"value\":10}]\n"
	getDataArgs := GetDataArgs{
		Ctx:                   common.TimeItContext(context.Background()),
		UserID:                testUserId,
		TraceID:               "trace1",
		StartDate:             "",
		EndDate:               "",
		WithPumpSettings:      false,
		WithParametersChanges: false,
		SessionToken:          "token1",
		ConvertToMgdl:         false,
	}

	getDataArgsWithConversion := GetDataArgs{
		Ctx:                   getDataArgs.Ctx,
		UserID:                getDataArgs.UserID,
		TraceID:               getDataArgs.TraceID,
		StartDate:             getDataArgs.StartDate,
		EndDate:               getDataArgs.EndDate,
		WithPumpSettings:      getDataArgs.WithPumpSettings,
		WithParametersChanges: getDataArgs.WithParametersChanges,
		SessionToken:          getDataArgs.SessionToken,
		ConvertToMgdl:         true,
	}

	patientDataRepository := MockPatientDataRepository{}
	patientDataRepository.On("GetDataInDeviceData", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(
		infrastructure.NewEmptyMockDbAdapterIterator(),
		nil,
	)
	patientDataRepository.On("GetLatestBasalSecurityProfile", mock.Anything, mock.Anything, testUserId).Return(&schema.DbProfile{
		Type:          "test",
		Time:          time.Time{},
		Timezone:      "UTC",
		Guid:          "osefduguid",
		BasalSchedule: nil,
	}, nil)
	tideV2Client := tidewhisperer.TideWhispererV2MockClient{}
	tideV2Client.MockedCbg = oneCbgArray
	tideV2Client.On("GetSettings", mock.Anything, testUserId, mock.Anything).Return(&tideV2Schema.SettingsResult{
		TimedCurrentSettings: orcaSchema.TimedCurrentSettings{
			CurrentSettings: orcaSchema.CurrentSettings{
				UserId:     testUserId,
				Device:     nil,
				Cgm:        nil,
				Pump:       nil,
				Parameters: nil,
			},
			Time:           nil,
			Timezone:       "",
			TimezoneOffset: nil,
		},
		HistoryParameters: nil,
	}, nil)

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

func TestPatientData_GetData_ShouldConvertOnlyMmolHistoryParameters(t *testing.T) {
	testUserId := "testParamsMmolUserId"
	patientDataRepository := MockPatientDataRepository{}
	patientDataRepository.On("GetDataInDeviceData", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(
		infrastructure.NewEmptyMockDbAdapterIterator(),
		nil,
	)
	patientDataRepository.On("GetLatestBasalSecurityProfile", mock.Anything, mock.Anything, testUserId).Return(&schema.DbProfile{
		Type:          "test",
		Time:          time.Time{},
		Timezone:      "UTC",
		Guid:          "osefduguid",
		BasalSchedule: nil,
	}, nil)
	tideV2Client := tidewhisperer.TideWhispererV2MockClient{}

	settingsResult := &tideV2Schema.SettingsResult{
		TimedCurrentSettings: orcaSchema.TimedCurrentSettings{
			CurrentSettings: orcaSchema.CurrentSettings{
				UserId: testUserId,
				Device: nil,
				Cgm:    nil,
				Pump:   nil,
				Parameters: []orcaSchema.CurrentParameter{
					{
						Name:          "unexpectedCurrentParam",
						Value:         "0.1",
						Unit:          MmolL,
						Level:         1,
						EffectiveDate: &now,
					},
				},
			},
			Time:           nil,
			Timezone:       "",
			TimezoneOffset: nil,
		},
		HistoryParameters: []orcaSchema.HistoryParameter{
			{
				CurrentParameter: orcaSchema.CurrentParameter{
					Name:          "param1",
					Value:         "10",
					Unit:          MmolL,
					Level:         1,
					EffectiveDate: &now,
				},
				ChangeType:     "ADDED",
				PreviousValue:  "",
				PreviousUnit:   "",
				Timestamp:      now,
				Timezone:       "UTC",
				TimezoneOffset: 0,
			},
			{
				CurrentParameter: orcaSchema.CurrentParameter{
					Name:          "param2",
					Value:         "16",
					Unit:          MgdL,
					Level:         1,
					EffectiveDate: &now,
				},
				ChangeType:     "MODIFIED",
				PreviousValue:  "15",
				PreviousUnit:   MgdL,
				Timestamp:      now,
				Timezone:       "UTC",
				TimezoneOffset: 0,
			},
			{
				CurrentParameter: orcaSchema.CurrentParameter{
					Name:          "param3",
					Value:         "81",
					Unit:          MgdL,
					Level:         1,
					EffectiveDate: &now,
				},
				ChangeType:     "MODIFIED",
				PreviousValue:  "80",
				PreviousUnit:   MmolL,
				Timestamp:      now,
				Timezone:       "UTC",
				TimezoneOffset: 0,
			},
		},
	}
	tideV2Client.On("GetSettings", mock.Anything, testUserId, mock.Anything).Return(settingsResult, nil)

	unexpectedUnits := "\"units\":\"mmol/L\""
	unexpectedParam := "\"name\":\"unexpectedCurrentParam\""

	/*convert param1 because unit is mmol*/
	expectedValue1 := "\"value\":\"6\""
	/*convert nothing in param2 because unit is mg/dL*/
	expectedValue2 := "\"previousValue\":\"15\""
	expectedValue3 := "\"value\":\"16\""
	/*convert previousValue only for param 3 because previousUnit is mmol*/
	expectedValue4 := "\"previousValue\":\"44\""
	expectedValue5 := "\"value\":\"81\""

	t.Run("convert to mgdl all mmol history params", func(t *testing.T) {
		p := &PatientData{
			patientDataRepository: &patientDataRepository,
			tideV2Client:          &tideV2Client,
			logger:                &log.Logger{},
			readBasalBucket:       false,
		}

		getDataArgs := GetDataArgs{
			Ctx:                   common.TimeItContext(context.Background()),
			UserID:                testUserId,
			TraceID:               "testParamsMmolTraceId",
			StartDate:             "",
			EndDate:               "",
			WithPumpSettings:      false,
			WithParametersChanges: true,
			SessionToken:          "sessionToken",
			ConvertToMgdl:         true,
		}
		res, err := p.GetData(getDataArgs)

		/*No error should be thrown*/
		assert.Nil(t, err)
		strRes := res.String()

		assert.NotContainsf(t, strRes, unexpectedUnits, "GetData result=%s does contains unexpected units=%s", strRes, unexpectedUnits)
		assert.NotContainsf(t, strRes, unexpectedParam, "GetData result=%s does contains unexpected param=%s", strRes, unexpectedParam)
		assert.Contains(t, strRes, expectedValue1, "GetData result=%s does not contains expected value=%s", strRes, expectedValue1)
		assert.Contains(t, strRes, expectedValue2, "GetData result=%s does not contains expected value=%s", strRes, expectedValue2)
		assert.Contains(t, strRes, expectedValue3, "GetData result=%s does not contains expected value=%s", strRes, expectedValue3)
		assert.Contains(t, strRes, expectedValue4, "GetData result=%s does not contains expected value=%s", strRes, expectedValue4)
		assert.Contains(t, strRes, expectedValue5, "GetData result=%s does not contains expected value=%s", strRes, expectedValue5)
	})
}

func TestPatientData_GetData_ShouldConvertOnlyMmolSmbgs(t *testing.T) {
	testUserId := "testSmbgsConvertionUserId"
	testTraceId := "testSmbgsConvertionTraceId"
	patientDataRepository := MockPatientDataRepository{}
	patientDataRepository.On("GetDataInDeviceData", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(
		infrastructure.NewMockDbAdapterIterator([]string{
			"{\"id\":\"1\",\"_userId\":\"user01\",\"uploadId\":\"upload01\",\"time\":\"2021-01-10T00:00:01.000Z\",\"timezone\":\"Europe/Paris\",\"type\":\"smbg\",\"units\":\"mmol/L\",\"value\":10}",
			"{\"id\":\"2\",\"_userId\":\"user01\",\"uploadId\":\"upload01\",\"time\":\"2021-01-10T00:05:01.000Z\",\"timezone\":\"Europe/Paris\",\"type\":\"smbg\",\"units\":\"mmol/L\",\"value\":15}",
			"{\"id\":\"3\",\"_userId\":\"user01\",\"uploadId\":\"upload01\",\"time\":\"2021-01-10T00:10:01.000Z\",\"timezone\":\"Europe/Paris\",\"type\":\"smbg\",\"units\":\"mg/dL\",\"value\":50}",
		}),
		nil,
	)
	patientDataRepository.On("GetUploadData", mock.Anything, testTraceId, []string{"upload01"}).Return(
		infrastructure.NewMockDbAdapterIterator([]string{
			"{\"time\":\"2022-08-08T16:40:00Z\",\"type\":\"upload\",\"id\":\"upload01\",\"timezone\":\"UTC\",\"_dataState\":\"open\",\"_deduplicator\":{\"name\":\"org.tidepool.deduplicator.none\",\"version\":\"1.0.0\"},\"_state\":\"open\",\"client\":{\"name\":\"portal-api.yourloops.com\",\"version\":\"1.0.0\" },\"dataSetType\":\"continuous\",\"deviceManufacturers\":[\"Diabeloop\"],\"deviceModel\":\"DBLG1\",\"deviceTags\":[\"cgm\",\"insulin-pump\"],\"revision\": 1,\"uploadId\":\"33031f76c78461670a1a95b5f032bb6a\",\"version\":\"1.0.0\",\"_userId\":\"osef\"}",
		}),
		nil,
	)
	tideV2Client := tidewhisperer.TideWhispererV2MockClient{}

	unexpectedUnits := "\"units\":\"mmol/L\""

	/*convert smbg1 and smbg2 because unit is mmol*/
	expectedValue1 := "\"value\":6"
	expectedValue2 := "\"value\":8"
	/*do not convert smbg3 because unit is mg/dL*/
	expectedValue3 := "\"value\":50"

	t.Run("convert to mgdl all mmol smbgs", func(t *testing.T) {
		p := &PatientData{
			patientDataRepository: &patientDataRepository,
			tideV2Client:          &tideV2Client,
			logger:                &log.Logger{},
			readBasalBucket:       false,
		}

		getDataArgs := GetDataArgs{
			Ctx:                   common.TimeItContext(context.Background()),
			UserID:                testUserId,
			TraceID:               testTraceId,
			StartDate:             "",
			EndDate:               "",
			WithPumpSettings:      false,
			WithParametersChanges: false,
			SessionToken:          "sessionToken",
			ConvertToMgdl:         true,
		}
		res, err := p.GetData(getDataArgs)

		/*No error should be thrown*/
		assert.Nil(t, err)
		strRes := res.String()

		assert.NotContainsf(t, strRes, unexpectedUnits, "GetData result=%s does contains unexpected units=%s", strRes, unexpectedUnits)
		assert.Contains(t, strRes, expectedValue1, "GetData result=%s does not contains expected value=%s", strRes, expectedValue1)
		assert.Contains(t, strRes, expectedValue2, "GetData result=%s does not contains expected value=%s", strRes, expectedValue2)
		assert.Contains(t, strRes, expectedValue3, "GetData result=%s does not contains expected value=%s", strRes, expectedValue3)
	})
}
