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

var now = time.Now().UTC()
var oneYearAgo = now.AddDate(-1, 0, 0)
var twoYearsAgo = now.AddDate(-2, 0, 0)
var fiveMinutesAgo = now.Add(-5 * time.Minute)
var fiveHoursAgo = now.Add(-5 * time.Hour)
var fiveSecondsAgo = now.Add(-5 * time.Second)

func createHistoryParam(name string, value string, unit string, date *time.Time, changeType string, previousValue string, previousUnit string) orcaSchema.HistoryParameter {
	return orcaSchema.HistoryParameter{
		CurrentParameter: orcaSchema.CurrentParameter{
			Name:          name,
			Value:         value,
			Unit:          unit,
			Level:         1,
			EffectiveDate: date,
		},
		ChangeType:     changeType,
		PreviousValue:  previousValue,
		PreviousUnit:   previousUnit,
		Timestamp:      *date,
		Timezone:       "UTC",
		TimezoneOffset: 0,
	}
}

func createUpdatedHistoryParam(name string, value string, unit string, date *time.Time, previousValue string, previousUnit string) orcaSchema.HistoryParameter {
	return createHistoryParam(name, value, unit, date, orcaSchema.UPDATED, previousValue, previousUnit)
}

func createAddedHistoryParam(name string, value string, unit string, date *time.Time) orcaSchema.HistoryParameter {
	return createHistoryParam(name, value, unit, date, orcaSchema.ADDED, "", "")
}

func setupEmptyPatientDataRepositoryMock(userId string) MockPatientDataRepository {
	patientDataRepository := MockPatientDataRepository{}
	patientDataRepository.On("GetDataInDeviceData", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(
		infrastructure.NewEmptyMockDbAdapterIterator(),
		nil,
	)
	patientDataRepository.On("GetLatestBasalSecurityProfile", mock.Anything, mock.Anything, userId).Return(&schema.DbProfile{
		Type:          "test",
		Time:          time.Time{},
		Timezone:      "UTC",
		Guid:          "osefduguid",
		BasalSchedule: nil,
	}, nil)
	return patientDataRepository
}

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
		Ctx:                        common.TimeItContext(context.Background()),
		UserID:                     testUserId,
		TraceID:                    "trace1",
		StartDate:                  "",
		EndDate:                    "",
		WithPumpSettings:           false,
		WithParametersChanges:      false,
		SessionToken:               "token1",
		ConvertToMgdl:              false,
		FilteringParametersChanges: false,
	}

	getDataArgsWithConversion := GetDataArgs{
		Ctx:                        getDataArgs.Ctx,
		UserID:                     getDataArgs.UserID,
		TraceID:                    getDataArgs.TraceID,
		StartDate:                  getDataArgs.StartDate,
		EndDate:                    getDataArgs.EndDate,
		WithPumpSettings:           getDataArgs.WithPumpSettings,
		WithParametersChanges:      getDataArgs.WithParametersChanges,
		SessionToken:               getDataArgs.SessionToken,
		FilteringParametersChanges: getDataArgs.FilteringParametersChanges,
		ConvertToMgdl:              true,
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
	patientDataRepository := setupEmptyPatientDataRepositoryMock(testUserId)
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
			createAddedHistoryParam("param1", "10", MmolL, &now),
			createUpdatedHistoryParam("param2", "16", MgdL, &now, "15", MgdL),
			createUpdatedHistoryParam("param3", "81", MgdL, &now, "80", MmolL),
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
		assert.Containsf(t, strRes, expectedValue1, "GetData result=%s does not contains expected value=%s", strRes, expectedValue1)
		assert.Containsf(t, strRes, expectedValue2, "GetData result=%s does not contains expected value=%s", strRes, expectedValue2)
		assert.Containsf(t, strRes, expectedValue3, "GetData result=%s does not contains expected value=%s", strRes, expectedValue3)
		assert.Containsf(t, strRes, expectedValue4, "GetData result=%s does not contains expected value=%s", strRes, expectedValue4)
		assert.Containsf(t, strRes, expectedValue5, "GetData result=%s does not contains expected value=%s", strRes, expectedValue5)
	})
}

func TestPatientData_GetData_FilterHistoryParameters(t *testing.T) {
	testUserId := "testParamsFilteringUserId"
	testTraceId := "testParamsFilteringTraceId"
	patientDataRepository := setupEmptyPatientDataRepositoryMock(testUserId)
	tideV2Client := tidewhisperer.TideWhispererV2MockClient{}

	settingsResult := &tideV2Schema.SettingsResult{
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
		HistoryParameters: []orcaSchema.HistoryParameter{
			createAddedHistoryParam("param1", "10", MmolL, &twoYearsAgo),
			createUpdatedHistoryParam("param2", "16", MgdL, &fiveHoursAgo, "15", MgdL),
			createUpdatedHistoryParam("param3", "81", MgdL, &fiveSecondsAgo, "80", MmolL),
		},
	}
	tideV2Client.On("GetSettings", mock.Anything, testUserId, mock.Anything).Return(settingsResult, nil)

	testCases := []struct {
		name                    string
		startDate               string
		endDate                 string
		withFilteringParameters bool
		expectedParams          []string
		unexpectedParams        []string
	}{
		{
			name:                    "should filter history params between startDate and endDate",
			startDate:               oneYearAgo.Format(time.RFC3339Nano),
			endDate:                 fiveMinutesAgo.Format(time.RFC3339Nano),
			withFilteringParameters: true,
			expectedParams:          []string{"param2"},
			unexpectedParams:        []string{"param1", "param3"},
		},
		{
			name:                    "should not filter any params if startDate and endDate are empty",
			startDate:               "",
			endDate:                 "",
			withFilteringParameters: true,
			expectedParams:          []string{"param1", "param2", "param3"},
			unexpectedParams:        []string{},
		},
		{
			name:                    "should not filter any params if startDate and endDate are provided but filtering boolean set to false",
			startDate:               fiveMinutesAgo.Format(time.RFC3339Nano),
			endDate:                 now.Format(time.RFC3339Nano),
			withFilteringParameters: false,
			expectedParams:          []string{"param1", "param2", "param3"},
			unexpectedParams:        []string{},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			p := &PatientData{
				patientDataRepository: &patientDataRepository,
				tideV2Client:          &tideV2Client,
				logger:                &log.Logger{},
				readBasalBucket:       false,
			}

			getDataArgs := GetDataArgs{
				Ctx:                        common.TimeItContext(context.Background()),
				UserID:                     testUserId,
				TraceID:                    testTraceId,
				StartDate:                  tt.startDate,
				EndDate:                    tt.endDate,
				WithPumpSettings:           false,
				WithParametersChanges:      true,
				FilteringParametersChanges: tt.withFilteringParameters,
			}

			res, err := p.GetData(getDataArgs)
			assert.Nil(t, err)
			strRes := res.String()
			for _, expectedParam := range tt.expectedParams {
				assert.Containsf(t, strRes, expectedParam, "GetData result=%s does not contains expected param=%s", strRes, expectedParam)
			}
			for _, unexpectedParam := range tt.unexpectedParams {
				assert.NotContainsf(t, strRes, unexpectedParam, "GetData result=%s does contains unexpected param=%s", strRes, unexpectedParam)
			}
		})
	}
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
			Ctx:                        common.TimeItContext(context.Background()),
			UserID:                     testUserId,
			TraceID:                    testTraceId,
			StartDate:                  "",
			EndDate:                    "",
			WithPumpSettings:           false,
			WithParametersChanges:      false,
			SessionToken:               "sessionToken",
			ConvertToMgdl:              true,
			FilteringParametersChanges: false,
		}
		res, err := p.GetData(getDataArgs)

		/*No error should be thrown*/
		assert.Nil(t, err)
		strRes := res.String()

		assert.NotContainsf(t, strRes, unexpectedUnits, "GetData result=%s does contains unexpected units=%s", strRes, unexpectedUnits)
		assert.Containsf(t, strRes, expectedValue1, "GetData result=%s does not contains expected value=%s", strRes, expectedValue1)
		assert.Containsf(t, strRes, expectedValue2, "GetData result=%s does not contains expected value=%s", strRes, expectedValue2)
		assert.Containsf(t, strRes, expectedValue3, "GetData result=%s does not contains expected value=%s", strRes, expectedValue3)
	})
}
