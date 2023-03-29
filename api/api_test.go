package api

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/gorilla/mux"
	"github.com/mdblp/go-common/clients/auth"
	"github.com/mdblp/shoreline/token"
	twV2Client "github.com/mdblp/tide-whisperer-v2/v2/client/tidewhisperer"
	"github.com/stretchr/testify/mock"
	"github.com/tidepool-org/go-common/clients/opa"
	"github.com/tidepool-org/go-common/clients/status"
	"github.com/tidepool-org/go-common/clients/version"
	"github.com/tidepool-org/tide-whisperer/common"
	"github.com/tidepool-org/tide-whisperer/infrastructure"
	"github.com/tidepool-org/tide-whisperer/usecase"
)

var (
	schemaVersions = common.SchemaVersion{
		Maximum: 99,
		Minimum: 1,
	}
	logger                = log.New(os.Stdout, "api-test", log.LstdFlags|log.Lshortfile)
	dbAdapter             = infrastructure.NewMockDbAdapter()
	patientDataRepository = infrastructure.NewMockPatientDataRepository()
	mockAuth              = auth.NewMock()
	mockPerms             = opa.NewMock()
	mockTideV2            = twV2Client.NewMock()
	patientDataUC         = usecase.NewPatientDataUseCase(logger, mockTideV2, patientDataRepository)
	api                   = InitAPI(ExportController{}, patientDataUC, dbAdapter, mockAuth, mockPerms, schemaVersions, logger, mockTideV2, false)
	rtr                   = mux.NewRouter()
)

// Utility function to reset all mocks to default value
func resetMocks() {
	mockAuth.ExpectedCalls = nil
	auth := mockPerms.GetMockedAuth(true, map[string]interface{}{}, "tidewhisperer-get")
	mockPerms.SetMockOpaAuth("/patient", &auth, nil)
	auth2 := mockPerms.GetMockedAuth(true, map[string]interface{}{}, "tidewhisperer-compute")
	mockPerms.SetMockOpaAuth("/compute/tir", &auth2, nil)
}

// Utility function to prepare request on GetStatus route
func getStatusPrepareRequest() (*http.Request, *httptest.ResponseRecorder) {
	version.ReleaseNumber = "1.2.3"
	version.FullCommit = "e0c73b95646559e9a3696d41711e918398d557fb"
	api.SetHandlers("", rtr)
	request, _ := http.NewRequest("GET", "/status", nil)
	response := httptest.NewRecorder()
	return request, response
}

// Utility function to prepare resposnes on GetStatus route
func getStatusParseResponse(response *httptest.ResponseRecorder) status.ApiStatus {
	body, _ := ioutil.ReadAll(response.Body)
	// Checking body content
	dataBody := status.ApiStatus{}
	json.Unmarshal([]byte(string(body)), &dataBody)
	return dataBody
}

// Testing GetStatus route
// TestGetStatus_StatusOk calling GetStatus route with an enabled dbAdapter
func TestGetStatus_StatusOk(t *testing.T) {
	resetMocks()
	mockAuth.On("Authenticate", mock.Anything).Return(&token.TokenData{UserId: "patient", IsServer: false})
	request, response := getStatusPrepareRequest()
	api.getStatus(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("Resp given [%d] expected [%d] ", response.Code, http.StatusOK)
	}
	// Checking body content
	dataBody := getStatusParseResponse(response)
	expectedStatus := status.ApiStatus{
		Status:  status.Status{Code: 200, Reason: "OK"},
		Version: version.ReleaseNumber + "+" + version.FullCommit,
	}
	if !reflect.DeepEqual(dataBody, expectedStatus) {
		t.Fatalf("patientData.GetStatus given [%v] expected [%v] ", dataBody, expectedStatus)
	}

}

// TestGetStatus_StatusKo calling GetStatus route with a disabled dbAdapter
func TestGetStatus_StatusKo(t *testing.T) {
	resetMocks()
	mockAuth.On("Authenticate", mock.Anything).Return(&token.TokenData{UserId: "patient", IsServer: false})
	dbAdapter.EnablePingError()

	request, response := getStatusPrepareRequest()
	api.getStatus(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("Resp given [%d] expected [%d] ", response.Code, http.StatusInternalServerError)
	}
	// Checking body content
	dataBody := getStatusParseResponse(response)
	expectedStatus := status.ApiStatus{
		Status:  status.Status{Code: 500, Reason: "Mock Ping Error"},
		Version: version.ReleaseNumber + "+" + version.FullCommit,
	}
	if !reflect.DeepEqual(dataBody, expectedStatus) {
		t.Fatalf("patientData.GetStatus given [%v] expected [%v] ", dataBody, expectedStatus)
	}
}

// Testing Get501 route
// TestGet501 calling Get501 route to check route is not authorized
func TestGet501(t *testing.T) {
	request, _ := http.NewRequest("GET", "/swagger", nil)
	response := httptest.NewRecorder()
	api.SetHandlers("", rtr)
	api.get501(response, request)
	if response.Code != http.StatusNotImplemented {
		t.Fatalf("Resp given [%d] expected [%d] ", response.Code, http.StatusNotImplemented)
	}
}
