package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/mdblp/go-common/clients/auth"
	tideV2Client "github.com/mdblp/tide-whisperer-v2/v2/client/tidewhisperer"
	"github.com/tidepool-org/go-common/clients/opa"
	"github.com/tidepool-org/go-common/clients/status"
	"github.com/tidepool-org/tide-whisperer/common"
	"github.com/tidepool-org/tide-whisperer/usecase"
)

type (
	// API struct for tide-whisperer
	API struct {
		patientData     PatientDataUseCase
		databaseAdapter usecase.DatabaseAdapter
		authClient      auth.ClientInterface
		perms           opa.Client
		schemaVersion   common.SchemaVersion
		logger          *log.Logger
		tideV2Client    tideV2Client.ClientInterface
		readBasalBucket bool
	}

	varsHandler func(http.ResponseWriter, *http.Request, map[string]string)

	//generic type as device data can be comprised of many things
	deviceData map[string]interface{}
)

const (
	// DataAPIPrefix logging prefix
	DataAPIPrefix             = "api/data "
	medtronicLoopBoundaryDate = "2017-09-01"
	slowQueryDuration         = 0.1 // seconds
)

var (
	errorStatusCheck       = common.DetailedError{Status: http.StatusInternalServerError, Code: "data_status_check", Message: "checking of the status endpoint showed an error"}
	errorNoViewPermission  = common.DetailedError{Status: http.StatusForbidden, Code: "data_cant_view", Message: "user is not authorized to view data"}
	errorNoPermissions     = common.DetailedError{Status: http.StatusInternalServerError, Code: "data_perms_error", Message: "error finding permissions for user"}
	errorRunningQuery      = common.DetailedError{Status: http.StatusInternalServerError, Code: "data_store_error", Message: "internal server error"}
	errorLoadingEvents     = common.DetailedError{Status: http.StatusInternalServerError, Code: "json_marshal_error", Message: "internal server error"}
	errorTideV2Http        = common.DetailedError{Status: http.StatusInternalServerError, Code: "tidev2_error", Message: "internal server error"}
	errorInvalidParameters = common.DetailedError{Status: http.StatusBadRequest, Code: "invalid_parameters", Message: "one or more parameters are invalid"}
	errorNotfound          = common.DetailedError{Status: http.StatusNotFound, Code: "data_not_found", Message: "no data for specified user"}
)

func InitAPI(patientDataUC PatientDataUseCase, dbAdapter usecase.DatabaseAdapter, auth auth.ClientInterface, permsClient opa.Client, schemaV common.SchemaVersion, logger *log.Logger, V2Client tideV2Client.ClientInterface, envReadBasalBucket bool) *API {
	return &API{
		patientData:     patientDataUC,
		databaseAdapter: dbAdapter,
		authClient:      auth,
		perms:           permsClient,
		schemaVersion:   schemaV,
		logger:          logger,
		tideV2Client:    V2Client,
		readBasalBucket: envReadBasalBucket,
	}
}

// SetHandlers set the API routes
func (a *API) SetHandlers(prefix string, rtr *mux.Router) {
	rtr.HandleFunc("/swagger", a.get501).Methods("GET")

	a.setHandlers(prefix+"/v1", rtr)
	rtr.HandleFunc("/v2", a.get501).Methods("GET")

	// v0 routes:
	rtr.HandleFunc("/status", a.getStatus).Methods("GET")
}

func (a *API) setHandlers(prefix string, rtr *mux.Router) {
	rtr.HandleFunc(prefix+"/range/{userID}", a.middlewareV1(a.getRange, true, "userID")).Methods("GET")
	rtr.HandleFunc(prefix+"/data/{userID}", a.middlewareV1(a.getData, true, "userID")).Methods("GET")
	rtr.HandleFunc(prefix+"/dataV2/{userID}", a.middlewareV1(a.getData, true, "userID")).Methods("GET")
	rtr.HandleFunc(prefix+"/{.*}", a.middlewareV1(a.getNotFound, false)).Methods("GET")
}

func (h varsHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	h(res, req, vars)
}

func (a *API) get501(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(501)
	return
}

// getNotFound should it be version free?
func (a *API) getNotFound(ctx context.Context, res *common.HttpResponseWriter) error {
	res.WriteHeader(http.StatusNotFound)
	return nil
}

// @Summary Get the api status
// @Description Get the api status
// @ID tide-whisperer-api-getstatus
// @Produce json
// @Success 200 {object} status.ApiStatus
// @Failure 500 {object} status.ApiStatus
// @Router /status [get]
func (a *API) getStatus(res http.ResponseWriter, req *http.Request) {
	start := time.Now()
	var s status.ApiStatus
	if err := a.databaseAdapter.Ping(); err != nil {
		errorLog := errorStatusCheck.SetInternalMessage(err)
		a.logError(&errorLog, start)
		s = status.NewApiStatus(errorLog.Status, err.Error())
	} else {
		s = status.NewApiStatus(http.StatusOK, "OK")
	}
	if jsonDetails, err := json.Marshal(s); err != nil {
		a.jsonError(res, errorLoadingEvents.SetInternalMessage(err), start)
	} else {
		res.Header().Add("content-type", "application/json")
		res.WriteHeader(s.Status.Code)
		res.Write(jsonDetails)
	}
	return
}

// log error detail and write as application/json
func (a *API) jsonError(res http.ResponseWriter, err common.DetailedError, startedAt time.Time) {
	a.logError(&err, startedAt)
	jsonErr, _ := json.Marshal(err)

	res.Header().Add("content-type", "application/json")
	res.WriteHeader(err.Status)
	res.Write(jsonErr)
}

func (a *API) logError(err *common.DetailedError, startedAt time.Time) {
	err.ID = uuid.New().String()
	a.logger.Println(DataAPIPrefix, fmt.Sprintf("[%s][%s] failed after [%.3f]secs with error [%s][%s] ", err.ID, err.Code, time.Now().Sub(startedAt).Seconds(), err.Message, err.InternalMessage))
}

func (a *API) isAuthorized(req *http.Request, targetUserIDs []string) bool {
	td := a.authClient.Authenticate(req)
	if td == nil {
		a.logger.Printf("%s - %s %s HTTP/%d.%d - Missing header token", req.RemoteAddr, req.Method, req.URL.String(), req.ProtoMajor, req.ProtoMinor)
		return false
	}
	if td.IsServer {
		return true
	}
	if len(targetUserIDs) == 1 {
		targetUserID := targetUserIDs[0]
		if td.UserId == targetUserID {
			return true
		}
	}

	auth, err := a.perms.GetOpaAuth(req)
	if err != nil {
		log.Println(DataAPIPrefix, fmt.Sprintf("Opa authorization error [%v] ", err))
		return false
	}
	return auth.Result.Authorized
}
