package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/mdblp/tide-whisperer-v2/v2/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tidepool-org/tide-whisperer/common"
	"github.com/tidepool-org/tide-whisperer/usecase"
)

const (
	dayTimeFormat  = "2006-01-02"
	expectedDataV1 = `{"id":"01","time":"2021-01-10T00:00:00.000Z","type":"basal","uploadId":"00","value":10},
		{"id":"02","time":"2021-01-10T00:00:01.000Z","type":"basal","uploadId":"00","value":11},
		{"id":"03","time":"2021-01-10T00:00:02.000Z","type":"basal","uploadId":"00","value":12},
		{"id":"04","time":"2021-01-10T00:00:03.000Z","type":"basal","uploadId":"00","value":13},
		{"id":"05","time":"2021-01-10T00:00:04.000Z","type":"basal","uploadId":"00","value":14}`
	expectedCbgBucket = `{"id":"cbg_bucket1_0","time":"2021-01-01T00:05:00Z","timezone":"GMT","type":"cbg","units":"mmol/L","value":10},
	{"id":"cbg_bucket1_1","time":"2021-01-01T00:10:00Z","timezone":"GMT","type":"cbg","units":"mmol/L","value":10.2},
	{"id":"cbg_bucket1_2","time":"2021-01-01T00:15:00Z","timezone":"GMT","type":"cbg","units":"mmol/L","value":10.8},
	{"id":"cbg_bucket2_0","time":"2021-01-02T00:05:00Z","timezone":"GMT","type":"cbg","units":"mmol/L","value":11},
	{"id":"cbg_bucket2_1","time":"2021-01-02T00:10:00Z","timezone":"GMT","type":"cbg","units":"mmol/L","value":11.2},
	{"id":"cbg_bucket2_2","time":"2021-01-02T00:15:00Z","timezone":"GMT","type":"cbg","units":"mmol/L","value":11.8}`
	expectedBasalBucket = `{"deliveryType":"automated","duration":1000,"id":"basal_bucket1_0","rate":1,"time":"2021-01-01T00:05:00Z","timezone":"Paris","type":"basal"}`
	expectedDataIdV1    = `{"id":"00","time":"2021-01-10T00:00:00.000Z","type":"upload","uploadId":"00"}`
)

func compareString(str1 string, str2 string) bool {
	// cleanse strings
	str1 = strings.ReplaceAll(str1, " ", "")
	str1 = strings.ReplaceAll(str1, "\n", "")
	str1 = strings.ReplaceAll(str1, "\t", "")
	str2 = strings.ReplaceAll(str2, " ", "")
	str2 = strings.ReplaceAll(str2, "\n", "")
	str2 = strings.ReplaceAll(str2, "\t", "")

	fmt.Printf("str1: %v\n", str1)
	fmt.Printf("str2: %v\n", str2)
	return str1 == str2
}

func assertRequest(apiParams map[string]string, urlParams map[string]string, expectedStatusCode int, expectedBody string) error {
	traceID := uuid.New().String()
	userID := apiParams["userID"]

	handlerLogFunc := api.middleware(api.getData, true, "userID")
	request, _ := http.NewRequest("GET", "/v1/dataV2/"+userID, nil)
	request.Header.Set("x-tidepool-trace-session", traceID)
	request.Header.Set("Authorization", "Bearer "+userID)
	request = mux.SetURLVars(request, apiParams)
	q := request.URL.Query()
	for key, value := range urlParams {
		q.Add(key, value)
	}
	request.URL.RawQuery = q.Encode()
	response := httptest.NewRecorder()

	handlerLogFunc(response, request)
	result := response.Result()
	if result.StatusCode != expectedStatusCode {
		return fmt.Errorf("Expected %d to equal %d", response.Code, http.StatusOK)
	}

	body := make([]byte, 2048)
	defer result.Body.Close()
	n, _ := result.Body.Read(body)
	bodyStr := string(body[:n])
	if !compareString(bodyStr, expectedBody) {
		return fmt.Errorf("Expected '%s' to equal '%s'", bodyStr, expectedBody)
	}
	return nil
}

func TestAPI_GetDataV2(t *testing.T) {
	userID := "abcdef"

	patientDataRepository.DataV1 = []string{
		"{\"id\":\"01\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:00.000Z\",\"type\":\"basal\",\"value\":10}",
		"{\"id\":\"02\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:01.000Z\",\"type\":\"basal\",\"value\":11}",
		"{\"id\":\"03\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:02.000Z\",\"type\":\"basal\",\"value\":12}",
		"{\"id\":\"04\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:03.000Z\",\"type\":\"basal\",\"value\":13}",
		"{\"id\":\"05\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:04.000Z\",\"type\":\"basal\",\"value\":14}",
	}
	patientDataRepository.DataIDV1 = []string{
		"{\"id\":\"00\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:00.000Z\",\"type\":\"upload\"}",
	}

	creationTime1, _ := time.Parse(time.RFC3339, "2021-01-01T08:00:00Z")
	day1, _ := time.Parse(dayTimeFormat, "2021-01-01")

	creationTime2, _ := time.Parse(time.RFC3339, "2021-01-02T08:00:00Z")
	day2, _ := time.Parse(dayTimeFormat, "2021-01-02")

	mockTideV2.MockedCbg = []schema.CbgBucket{
		{
			Id:                "bucket1",
			CreationTimestamp: creationTime1,
			UserId:            userID,
			Day:               day1,
			Samples: []schema.CbgSample{
				{
					Value:          10.0,
					Units:          "mmol/L",
					Timestamp:      day1.Add(time.Minute * 5),
					Timezone:       "GMT",
					TimezoneOffset: 0,
				},
				{
					Value:          10.2,
					Units:          "mmol/L",
					Timestamp:      day1.Add(time.Minute * 10),
					Timezone:       "GMT",
					TimezoneOffset: 0,
				},
				{
					Value:          10.8,
					Units:          "mmol/L",
					Timestamp:      day1.Add(time.Minute * 15),
					Timezone:       "GMT",
					TimezoneOffset: 0,
				},
			},
		},
		{
			Id:                "bucket2",
			CreationTimestamp: creationTime2,
			UserId:            userID,
			Day:               day2,
			Samples: []schema.CbgSample{
				{
					Value:          11.0,
					Units:          "mmol/L",
					Timestamp:      day2.Add(time.Minute * 5),
					Timezone:       "GMT",
					TimezoneOffset: 0,
				},
				{
					Value:          11.2,
					Units:          "mmol/L",
					Timestamp:      day2.Add(time.Minute * 10),
					Timezone:       "GMT",
					TimezoneOffset: 0,
				},
				{
					Value:          11.8,
					Units:          "mmol/L",
					Timestamp:      day2.Add(time.Minute * 15),
					Timezone:       "GMT",
					TimezoneOffset: 0,
				},
			},
		},
	}
	mockTideV2.MockedBasal = []schema.BasalBucket{
		{
			Id:                "bucket1",
			CreationTimestamp: creationTime1,
			UserId:            userID,
			Day:               day1,
			Samples: []schema.BasalSample{
				{
					DeliveryType: "automated",
					Duration:     1000,
					Rate:         1.0,
					Sample: schema.Sample{
						Timestamp:      day1.Add(time.Minute * 5),
						Timezone:       "Paris",
						TimezoneOffset: 120,
					},
				},
			},
		},
	}
	t.Cleanup(func() {
		patientDataRepository.DataV1 = nil
		patientDataRepository.DataIDV1 = nil
		mockTideV2.MockedCbg = []schema.CbgBucket{}
		mockTideV2.MockedBasal = []schema.BasalBucket{}
	})

	resetOPAMockRouteV1(true, "/v1/dataV2", userID)

	// testing with cbg and basal buckets
	apiParms := map[string]string{
		"userID": userID,
	}
	urlParams := map[string]string{}

	patientDataUseCase := usecase.NewPatientDataUseCase(logger, mockTideV2, patientDataRepository, true)
	api = InitAPI(ExportController{}, patientDataUseCase, dbAdapter, mockAuth, mockPerms, schemaVersions, logger, mockTideV2)
	expectedBody := "[" + strings.Join(
		[]string{
			expectedDataV1,
			expectedCbgBucket,
			expectedBasalBucket,
			expectedDataIdV1,
		}, ",") + "]"
	err := assertRequest(apiParms, urlParams, http.StatusOK, expectedBody)
	if err != nil {
		t.Fatalf("CBG and Basal buckets: %v", err.Error())
	}

	// testing with cbg only, required to set basal to false
	patientDataUseCase = usecase.NewPatientDataUseCase(logger, mockTideV2, patientDataRepository, false)
	api = InitAPI(ExportController{}, patientDataUseCase, dbAdapter, mockAuth, mockPerms, schemaVersions, logger, mockTideV2)
	expectedBody = "[" + strings.Join(
		[]string{
			expectedDataV1,
			expectedCbgBucket,
			expectedDataIdV1,
		}, ",") + "]"

	err = assertRequest(apiParms, urlParams, http.StatusOK, expectedBody)
	if err != nil {
		t.Fatalf("Cbg bucket only: %v", err.Error())
	}

	patientDataUseCase = usecase.NewPatientDataUseCase(logger, mockTideV2, patientDataRepository, true)
	api = InitAPI(ExportController{}, patientDataUseCase, dbAdapter, mockAuth, mockPerms, schemaVersions, logger, mockTideV2)
	expectedBasalBucket := `{"deliveryType":"automated","duration":1000,"id":"basal_bucket1_0","rate":1,"time":"2021-01-01T00:05:00Z","timezone":"Paris","type":"basal"}`
	expectedBody = "[" + strings.Join(
		[]string{
			expectedDataV1,
			expectedCbgBucket,
			expectedBasalBucket,
			expectedDataIdV1,
		}, ",") + "]"
	err = assertRequest(apiParms, urlParams, http.StatusOK, expectedBody)
	if err != nil {
		t.Fatalf("CBG and Basal buckets: %v", err.Error())
	}
}

func TestAPI_GetRangeV1(t *testing.T) {
	traceID := uuid.New().String()
	userID := "abcdef"
	urlParams := map[string]string{
		"userID": userID,
	}

	resetOPAMockRouteV1(true, "/v1/range", userID)
	patientDataRepository.DataRangeV1 = []string{"2021-01-01T00:00:00.000Z", "2021-01-03T00:00:00.000Z"}
	t.Cleanup(func() {
		patientDataRepository.DataRangeV1 = nil
	})
	expectedValue := "[\"" + patientDataRepository.DataRangeV1[0] + "\",\"" + patientDataRepository.DataRangeV1[1] + "\"]"
	handlerLogFunc := api.middleware(api.getRangeLegacy, true, "userID")

	request, _ := http.NewRequest("GET", "/v1/range/"+userID, nil)
	request.Header.Set("x-tidepool-trace-session", traceID)
	request.Header.Set("Authorization", "Bearer "+userID)
	request = mux.SetURLVars(request, urlParams)
	response := httptest.NewRecorder()

	handlerLogFunc(response, request)
	result := response.Result()
	if result.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d to equal %d", response.Code, http.StatusOK)
	}

	body := make([]byte, 1024)
	defer result.Body.Close()
	n, _ := result.Body.Read(body)
	bodyStr := string(body[:n])

	if bodyStr != expectedValue {
		t.Errorf("Expected '%s' to equal '%s'", bodyStr, expectedValue)
	}
}

func TestAPI_getDataV2_bgUnit(t *testing.T) {
	tests := []struct {
		name                    string
		givenBgUnitQueryParam   string
		expectedBgUnitInUseCase string
	}{
		{"Valid mg/dL", "mg/dL", "mg/dL"},
		{"Valid mmol/L", "mmol/L", "mmol/L"},
		{"Invalid unit", "invalid_unit", ""},
		{"No unit provided", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPatientData := MockPatientDataUseCase{}
			mockPatientData.On("GetData", mock.Anything, mock.Anything).Return(new(bytes.Buffer), nil)
			api := &API{patientData: &mockPatientData}
			/*Build the request with bgUnit query param*/
			request, _ := http.NewRequest("GET", "/v1/dataV2/testBgUnit?bgUnit="+tt.givenBgUnitQueryParam, nil)
			httpResponseWriter := common.HttpResponseWriter{}
			httpResponseWriter.URL = request.URL
			/*Call getDataV2*/
			err := api.getDataV2(context.Background(), &httpResponseWriter)
			/*Assert no error and mock is called with expectedBgUnitInUseCase*/
			assert.NoError(t, err)
			mockPatientData.AssertCalled(t, "GetData", mock.Anything, mock.MatchedBy(func(args usecase.GetDataArgs) bool {
				return args.BgUnit == tt.expectedBgUnitInUseCase
			}))
		})
	}
}
