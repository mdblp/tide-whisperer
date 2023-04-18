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

var (
	now            = time.Now().UTC()
	oneYearAgo     = now.AddDate(-1, 0, 0)
	twoYearsAgo    = now.AddDate(-2, 0, 0)
	fiveMinutesAgo = now.Add(-5 * time.Minute)
	fiveHoursAgo   = now.Add(-5 * time.Hour)
	fiveSecondsAgo = now.Add(-5 * time.Second)
	cbgTime        = time.Date(2023, time.April, 1, 12, 32, 0, 0, time.UTC)
	cbgSample      = tideV2Schema.CbgSample{
		Value:          10,
		Units:          MmolL,
		Timestamp:      cbgTime,
		Timezone:       "UTC",
		TimezoneOffset: 0,
	}
	oneCbgResultMgdl = `[
{"id":"cbg_cbg1_0","time":"2023-04-01T12:32:00Z","timezone":"UTC","type":"cbg","units":"mg/dL","value":6}]
`
	oneCbgResultMmol = `[
{"id":"cbg_cbg1_0","time":"2023-04-01T12:32:00Z","timezone":"UTC","type":"cbg","units":"mmol/L","value":10}]
`
)

type patientDataGiven struct {
	patientDataRepository PatientDataRepository
	tideV2Client          tidewhisperer.ClientInterface
	logger                *log.Logger
	readBasalBucket       bool
	getDataArgs           GetDataArgs
}

type patientDataExpected struct {
	err    *common.DetailedError
	result *bytes.Buffer
}

func expectErrIsNil(t *testing.T, p patientDataExpected) {
	assert.Nil(t, p.err)
}

func expectCbgResultIsInMgdl(t *testing.T, p patientDataExpected) {
	assert.Equal(t, oneCbgResultMgdl, p.result.String())
}

func expectCbgResultIsInMmol(t *testing.T, p patientDataExpected) {
	assert.Equal(t, oneCbgResultMmol, p.result.String())
}

func expectSmbgResultIsInMgdl(t *testing.T, p patientDataExpected) {
	unexpectedUnits := MmolL
	/*convert smbg1 and smbg2 because unit is mmol*/
	expectedValue1 := `"value":6`
	expectedValue2 := `"value":8`
	/*do not convert smbg3 because unit is already mg/dL*/
	expectedValue3 := `"value":50`
	resultString := p.result.String()
	assert.NotContainsf(t, resultString, unexpectedUnits, "GetData result=%s does contains unexpected units=%s", resultString, unexpectedUnits)
	assert.Containsf(t, resultString, expectedValue1, "GetData result=%s does not contains expected value=%s", resultString, expectedValue1)
	assert.Containsf(t, resultString, expectedValue2, "GetData result=%s does not contains expected value=%s", resultString, expectedValue2)
	assert.Containsf(t, resultString, expectedValue3, "GetData result=%s does not contains expected value=%s", resultString, expectedValue3)
}

func expectOnlyParamFilter1And2ArePresent(t *testing.T, p patientDataExpected) {
	resultString := p.result.String()
	paramNotExpected := "paramFilter3"
	paramExpected1 := "paramFilter1"
	paramExpected2 := "paramFilter2"
	assert.NotContainsf(t, resultString, paramNotExpected, "GetData result=%s does contains unexpected parameter=%s", resultString, paramNotExpected)
	assert.Containsf(t, resultString, paramExpected1, "GetData result=%s does not contains expected parameter=%s", resultString, paramExpected1)
	assert.Containsf(t, resultString, paramExpected2, "GetData result=%s does not contains expected parameter=%s", resultString, paramExpected2)
}

func expectAllParamFiltersArePresent(t *testing.T, p patientDataExpected) {
	resultString := p.result.String()
	paramExpected1 := "paramFilter1"
	paramExpected2 := "paramFilter2"
	paramExpected3 := "paramFilter3"
	assert.Containsf(t, resultString, paramExpected1, "GetData result=%s does not contains expected parameter=%s", resultString, paramExpected1)
	assert.Containsf(t, resultString, paramExpected2, "GetData result=%s does not contains expected parameter=%s", resultString, paramExpected2)
	assert.Containsf(t, resultString, paramExpected3, "GetData result=%s does not contains expected parameter=%s", resultString, paramExpected3)
}

func expectParameterAreNotConverted(t *testing.T, p patientDataExpected) {
	resultString := p.result.String()
	assert.Containsf(t, resultString, MgdL, "GetData result=%s does not contains expected unit=%s", resultString, MgdL)
	assert.Containsf(t, resultString, MmolL, "GetData result=%s does not contains expected unit=%s", resultString, MmolL)
}

func expectHistoryParamIsInMgdl(t *testing.T, p patientDataExpected) {
	unexpectedUnits := "mmol/L"
	unexpectedParam := "unexpectedCurrentParam"

	/*convert param1 because unit is mmol*/
	expectedValue1 := `"value":"6"`
	/*convert nothing in param2 because unit is mg/dL*/
	expectedValue2 := `"previousValue":"15"`
	expectedValue3 := `"value":"16"`
	/*convert previousValue only for param 3 because previousUnit is mmol*/
	expectedValue4 := `"previousValue":"44"`
	expectedValue5 := `"value":"81"`

	resultString := p.result.String()
	assert.NotContainsf(t, resultString, unexpectedUnits, "GetData result=%s does contains unexpected units=%s", resultString, unexpectedUnits)
	assert.NotContainsf(t, resultString, unexpectedParam, "GetData result=%s does contains unexpected param=%s", resultString, unexpectedParam)
	assert.Containsf(t, resultString, expectedValue1, "GetData result=%s does not contains expected value=%s", resultString, expectedValue1)
	assert.Containsf(t, resultString, expectedValue2, "GetData result=%s does not contains expected value=%s", resultString, expectedValue2)
	assert.Containsf(t, resultString, expectedValue3, "GetData result=%s does not contains expected value=%s", resultString, expectedValue3)
	assert.Containsf(t, resultString, expectedValue4, "GetData result=%s does not contains expected value=%s", resultString, expectedValue4)
	assert.Containsf(t, resultString, expectedValue5, "GetData result=%s does not contains expected value=%s", resultString, expectedValue5)
}

func paramConvertToMgdlTrue(p patientDataGiven) patientDataGiven {
	p.getDataArgs.ConvertToMgdl = true
	return p
}

func paramConvertToMgdlFalse(p patientDataGiven) patientDataGiven {
	p.getDataArgs.ConvertToMgdl = false
	return p
}

func mockGetLatestBasalSecurityProfileWithDummyReturn(p patientDataGiven) patientDataGiven {
	p.patientDataRepository.(*MockPatientDataRepository).On("GetLatestBasalSecurityProfile", mock.Anything, mock.Anything, p.getDataArgs.UserID).Return(&schema.DbProfile{
		Type:          "test",
		Time:          time.Time{},
		Timezone:      "UTC",
		Guid:          "osefduguid",
		BasalSchedule: nil,
	}, nil)
	return p
}

func paramWithParametersHistoryTrue(p patientDataGiven) patientDataGiven {
	p.getDataArgs.WithParametersHistory = true
	return p
}

func paramFilteringParametersHistoryTrue(p patientDataGiven) patientDataGiven {
	p.getDataArgs.FilteringParametersHistory = true
	return p
}

func paramFilteringParametersHistoryFalse(p patientDataGiven) patientDataGiven {
	p.getDataArgs.FilteringParametersHistory = false
	return p
}

func paramStartDate2YearsAgo(p patientDataGiven) patientDataGiven {
	p.getDataArgs.StartDate = twoYearsAgo.Format(time.RFC3339Nano)
	return p
}

func paramEndDate5MinAgo(p patientDataGiven) patientDataGiven {
	p.getDataArgs.EndDate = fiveMinutesAgo.Format(time.RFC3339Nano)
	return p
}

func noDeviceDataReturnedByRepository(p patientDataGiven) patientDataGiven {
	patientDataRepository := MockPatientDataRepository{}
	patientDataRepository.On("GetDataInDeviceData", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(
		infrastructure.NewEmptyMockDbAdapterIterator(),
		nil,
	)
	p.patientDataRepository = &patientDataRepository
	return p
}
func smbgReturnedByRepository(p patientDataGiven) patientDataGiven {
	patientDataRepository := MockPatientDataRepository{}
	patientDataRepository.On("GetDataInDeviceData", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(
		infrastructure.NewMockDbAdapterIterator([]string{
			"{\"id\":\"1\",\"_userId\":\"user01\",\"uploadId\":\"upload01\",\"time\":\"2021-01-10T00:00:01.000Z\",\"timezone\":\"Europe/Paris\",\"type\":\"smbg\",\"units\":\"mmol/L\",\"value\":10}",
			"{\"id\":\"2\",\"_userId\":\"user01\",\"uploadId\":\"upload01\",\"time\":\"2021-01-10T00:05:01.000Z\",\"timezone\":\"Europe/Paris\",\"type\":\"smbg\",\"units\":\"mmol/L\",\"value\":15}",
			"{\"id\":\"3\",\"_userId\":\"user01\",\"uploadId\":\"upload01\",\"time\":\"2021-01-10T00:10:01.000Z\",\"timezone\":\"Europe/Paris\",\"type\":\"smbg\",\"units\":\"mg/dL\",\"value\":50}",
		}),
		nil,
	)
	patientDataRepository.On("GetUploadData", mock.Anything, p.getDataArgs.TraceID, []string{"upload01"}).Return(
		infrastructure.NewMockDbAdapterIterator([]string{
			"{\"time\":\"2022-08-08T16:40:00Z\",\"type\":\"upload\",\"id\":\"upload01\",\"timezone\":\"UTC\",\"_dataState\":\"open\",\"_deduplicator\":{\"name\":\"org.tidepool.deduplicator.none\",\"version\":\"1.0.0\"},\"_state\":\"open\",\"client\":{\"name\":\"portal-api.yourloops.com\",\"version\":\"1.0.0\" },\"dataSetType\":\"continuous\",\"deviceManufacturers\":[\"Diabeloop\"],\"deviceModel\":\"DBLG1\",\"deviceTags\":[\"cgm\",\"insulin-pump\"],\"revision\": 1,\"uploadId\":\"33031f76c78461670a1a95b5f032bb6a\",\"version\":\"1.0.0\",\"_userId\":\"osef\"}",
		}),
		nil,
	)
	p.patientDataRepository = &patientDataRepository
	return p
}

func oneCbgReturnedInMmolByTideV2(p patientDataGiven) patientDataGiven {
	tideV2Client := tidewhisperer.TideWhispererV2MockClient{}
	tideV2Client.MockedCbg = getCbgBucketWithOneCbgSample(p.getDataArgs.UserID)
	p.tideV2Client = &tideV2Client
	return p
}
func nothingReturnedByTideV2(p patientDataGiven) patientDataGiven {
	tideV2Client := tidewhisperer.TideWhispererV2MockClient{}
	p.tideV2Client = &tideV2Client
	return p
}

func threeParamHistoryForConvertionReturnedByTideV2(p patientDataGiven) patientDataGiven {
	tideV2Client := tidewhisperer.TideWhispererV2MockClient{}
	settingsResult := &tideV2Schema.SettingsResult{
		TimedCurrentSettings: orcaSchema.TimedCurrentSettings{
			CurrentSettings: orcaSchema.CurrentSettings{
				UserId: p.getDataArgs.UserID,
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
	tideV2Client.On("GetSettings", mock.Anything, p.getDataArgs.UserID, mock.Anything).Return(settingsResult, nil)
	p.tideV2Client = &tideV2Client
	return p
}

func threeParamHistoryForFilteringReturnedByTideV2(p patientDataGiven) patientDataGiven {
	tideV2Client := tidewhisperer.TideWhispererV2MockClient{}
	settingsResult := &tideV2Schema.SettingsResult{
		TimedCurrentSettings: orcaSchema.TimedCurrentSettings{
			CurrentSettings: orcaSchema.CurrentSettings{
				UserId:     p.getDataArgs.UserID,
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
			createAddedHistoryParam("paramFilter1", "10", MmolL, &oneYearAgo),
			createUpdatedHistoryParam("paramFilter2", "16", MgdL, &fiveHoursAgo, "15", MgdL),
			createUpdatedHistoryParam("paramFilter3", "81", MgdL, &fiveSecondsAgo, "80", MmolL),
		},
	}
	tideV2Client.On("GetSettings", mock.Anything, p.getDataArgs.UserID, mock.Anything).Return(settingsResult, nil)
	p.tideV2Client = &tideV2Client
	return p
}

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

func getCbgBucketWithOneCbgSample(userid string) []tideV2Schema.CbgBucket {
	cbgDay := time.Date(2023, time.April, 1, 0, 0, 0, 0, time.UTC)
	return []tideV2Schema.CbgBucket{
		{
			Id:                "cbg1",
			CreationTimestamp: now,
			UserId:            userid,
			Day:               cbgDay,
			Samples: []tideV2Schema.CbgSample{
				cbgSample,
			},
		},
	}
}

func emptyPatientDataGiven(userid string) patientDataGiven {
	return patientDataGiven{
		patientDataRepository: nil,
		tideV2Client:          nil,
		logger:                &log.Logger{},
		readBasalBucket:       false,
		getDataArgs: GetDataArgs{
			Ctx:                        common.TimeItContext(context.Background()),
			UserID:                     userid,
			TraceID:                    "trace1",
			StartDate:                  "",
			EndDate:                    "",
			WithPumpSettings:           false,
			WithParametersHistory:      false,
			SessionToken:               "token1",
			FilteringParametersHistory: false,
			ConvertToMgdl:              true,
		},
	}
}

func TestPatientData_GetData(t *testing.T) {
	tests := []struct {
		name     string
		given    []func(patientDataGiven) patientDataGiven
		expected []func(*testing.T, patientDataExpected)
	}{
		{
			name: "should convert cbg to mgdl when ConvertToMgdl is set to true",
			given: []func(patientDataGiven) patientDataGiven{
				paramConvertToMgdlTrue,
				noDeviceDataReturnedByRepository,
				oneCbgReturnedInMmolByTideV2,
			},
			expected: []func(*testing.T, patientDataExpected){
				expectErrIsNil,
				expectCbgResultIsInMgdl,
			},
		},
		{
			name: "should not convert cbg to mgdl when ConvertToMgdl is set to false",
			given: []func(patientDataGiven) patientDataGiven{
				paramConvertToMgdlFalse,
				noDeviceDataReturnedByRepository,
				oneCbgReturnedInMmolByTideV2,
			},
			expected: []func(*testing.T, patientDataExpected){
				expectErrIsNil,
				expectCbgResultIsInMmol,
			},
		},
		{
			name: "should convert smbg to mgdl when ConvertToMgdl is set to true",
			given: []func(patientDataGiven) patientDataGiven{
				paramConvertToMgdlTrue,
				smbgReturnedByRepository,
				nothingReturnedByTideV2,
			},
			expected: []func(*testing.T, patientDataExpected){
				expectErrIsNil,
				expectSmbgResultIsInMgdl,
			},
		},
		{
			name: "should convert history parameters to mgdl if unit is mmol when ConvertToMgdl is set to true",
			given: []func(patientDataGiven) patientDataGiven{
				paramConvertToMgdlTrue,
				paramWithParametersHistoryTrue,
				noDeviceDataReturnedByRepository,
				threeParamHistoryForConvertionReturnedByTideV2,
				mockGetLatestBasalSecurityProfileWithDummyReturn,
			},
			expected: []func(*testing.T, patientDataExpected){
				expectErrIsNil,
				expectHistoryParamIsInMgdl,
			},
		},
		{
			name: "should filter history parameters between startDate and endDate when FilteringParametersHistory is set to true",
			given: []func(patientDataGiven) patientDataGiven{
				paramConvertToMgdlFalse,
				paramWithParametersHistoryTrue,
				paramFilteringParametersHistoryTrue,
				paramStartDate2YearsAgo,
				paramEndDate5MinAgo,
				noDeviceDataReturnedByRepository,
				threeParamHistoryForFilteringReturnedByTideV2,
				mockGetLatestBasalSecurityProfileWithDummyReturn,
			},
			expected: []func(*testing.T, patientDataExpected){
				expectErrIsNil,
				expectOnlyParamFilter1And2ArePresent,
				expectParameterAreNotConverted,
			},
		},
		{
			name: "should not filter history parameters between startDate and endDate if they are empty and FilteringParametersHistory is set to true",
			given: []func(patientDataGiven) patientDataGiven{
				paramConvertToMgdlFalse,
				paramWithParametersHistoryTrue,
				paramFilteringParametersHistoryTrue,
				noDeviceDataReturnedByRepository,
				threeParamHistoryForFilteringReturnedByTideV2,
				mockGetLatestBasalSecurityProfileWithDummyReturn,
			},
			expected: []func(*testing.T, patientDataExpected){
				expectErrIsNil,
				expectAllParamFiltersArePresent,
				expectParameterAreNotConverted,
			},
		},
		{
			name: "should not filter history parameters between startDate and endDate if they are provided and FilteringParametersHistory is set to false",
			given: []func(patientDataGiven) patientDataGiven{
				paramConvertToMgdlFalse,
				paramWithParametersHistoryTrue,
				paramFilteringParametersHistoryFalse,
				paramStartDate2YearsAgo,
				paramEndDate5MinAgo,
				noDeviceDataReturnedByRepository,
				threeParamHistoryForFilteringReturnedByTideV2,
				mockGetLatestBasalSecurityProfileWithDummyReturn,
			},
			expected: []func(*testing.T, patientDataExpected){
				expectErrIsNil,
				expectAllParamFiltersArePresent,
				expectParameterAreNotConverted,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			given := emptyPatientDataGiven("userid_test_getData")
			for _, f := range tt.given {
				given = f(given)
			}
			p := &PatientData{
				patientDataRepository: given.patientDataRepository,
				tideV2Client:          given.tideV2Client,
				logger:                given.logger,
				readBasalBucket:       given.readBasalBucket,
			}
			res, err := p.GetData(given.getDataArgs)
			expected := patientDataExpected{
				err:    err,
				result: res,
			}
			for _, f := range tt.expected {
				f(t, expected)
			}
			given.patientDataRepository.(*MockPatientDataRepository).AssertExpectations(t)
			given.tideV2Client.(*tidewhisperer.TideWhispererV2MockClient).AssertExpectations(t)
		})
	}
}
