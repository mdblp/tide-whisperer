package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	tideV2Client "github.com/mdblp/tide-whisperer-v2/v2/client/tidewhisperer"
	schemaV2 "github.com/mdblp/tide-whisperer-v2/v2/schema"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tidepool-org/go-common/clients/mongo"
	"github.com/tidepool-org/tide-whisperer/common"
	"github.com/tidepool-org/tide-whisperer/schema"
	"github.com/tidepool-org/tide-whisperer/usecase/basal"
)

var (
	errorRunningQuery      = common.DetailedError{Status: http.StatusInternalServerError, Code: "data_store_error", Message: "internal server error"}
	errorTideV2Http        = common.DetailedError{Status: http.StatusInternalServerError, Code: "tidev2_error", Message: "internal server error"}
	errorInvalidParameters = common.DetailedError{Status: http.StatusBadRequest, Code: "invalid_parameters", Message: "one or more parameters are invalid"}
)

type (
	// errorCounter to record only the first error to avoid spamming the log and takes too much time
	errorCounter struct {
		firstError error
		numErrors  int
	}
	// writeFromIter struct to pass to the function which write the http result from the mongo iterator for diabetes data
	writeFromIter struct {
		res      *common.HttpResponseWriter
		iter     mongo.StorageIterator
		settings *schemaV2.SettingsResult
		cbgs     []schemaV2.CbgBucket
		basals   []schemaV2.BasalBucket
		// parametersHistory fetched from portal database
		parametersHistory map[string]interface{}
		// basalSecurityProfile
		basalSecurityProfile interface{}
		// uploadIDs encountered during the operation
		uploadIDs []string
		// writeCount the number of data written
		writeCount int
		// datum decode errors
		decode errorCounter
		// datum JSON marshall errors
		jsonError errorCounter
	}
	// SummaryResultV1 returned by the summary v1 route
	SummaryResultV1 struct {
		// The userID of this summary
		UserID string `json:"userId"`
		// First upload data date (ISO-8601 datetime)
		RangeStart string `json:"rangeStart"`
		// Last upload data date (ISO-8601 datetime)
		RangeEnd string `json:"rangeEnd"`
		// Number of days used to compute the TIR & TBR
		ComputeDays int `json:"computeDays"`
		// % of cbg/smbg in range (TIR)
		PercentTimeInRange int `json:"percentTimeInRange"`
		// % of cbg/smbg below range (TBR)
		PercentTimeBelowRange int `json:"percentTimeBelowRange"`
		// Number of bg values used to compute the TIR & TBR (if 0, the percent values are meaningless)
		NumBgValues int `json:"numBgValues"`
		// The Hypo limit used to compute TIR & TBR
		GlyHypoLimit float64 `json:"glyHypoLimit"`
		// The Hyper limit used to compute TIR & TBR
		GlyHyperLimit float64 `json:"glyHyperLimit"`
		// The unit of hypo/hyper values
		GlyUnit string `json:"glyUnit"`
	}
	simplifiedBgDatum struct {
		Value float64 `json:"value" bson:"value"`
		Unit  string  `json:"units" bson:"units"`
	}
	deviceParameter struct {
		Level int    `json:"level" bson:"level"`
		Name  string `json:"name" bson:"name"`
		Unit  string `json:"unit" bson:"unit"`
		Value string `json:"value" bson:"value"`
	}
	pumpSettingsPayload struct {
		Parameters []deviceParameter `json:"parameters" bson:"parameters"`
		// Uncomment & fill if needed:
		// Device     map[string]string `json:"device" bson:"device"`
		// CGM        map[string]string `json:"cgm" bson:"cgm"`
		// Pump       map[string]string `json:"pump" bson:"pump"`
	}
	// PumpSettings datum to get a specific device parameter
	PumpSettings struct {
		ID      string              `json:"id" bson:"id"`
		Type    string              `json:"type" bson:"type"`
		Time    string              `json:"time" bson:"time"`
		Payload pumpSettingsPayload `json:"payload" bson:"payload"`
	}
)

// Parameters level to keep in api response
var parameterLevelFilter = [...]int{1, 2}

var dataFromStoreTimer = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:      "data_from_store_time",
	Help:      "A histogram for getDataFromStore execution time (ms)",
	Buckets:   prometheus.LinearBuckets(20, 20, 300),
	Subsystem: "tidewhisperer",
	Namespace: "dblp",
})

var dataFromTideV2Timer = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:      "data_from_tidev2_time",
	Help:      "A histogram for dataFromTideV2Timer execution time (ms)",
	Buckets:   prometheus.LinearBuckets(20, 20, 300),
	Subsystem: "tidewhisperer",
	Namespace: "dblp",
})

type PatientData struct {
	patientDataRepository PatientDataRepository
	tideV2Client          tideV2Client.ClientInterface
	logger                *log.Logger
}

func NewPatientDataUseCase(logger *log.Logger, tideV2Client tideV2Client.ClientInterface, patientDataRepository PatientDataRepository) *PatientData {
	return &PatientData{
		patientDataRepository: patientDataRepository,
		logger:                logger,
		tideV2Client:          tideV2Client,
	}
}

func (p *PatientData) getCbgFromTideV2(ctx context.Context, wg *sync.WaitGroup, userID string, sessionToken string, dates *common.Date, tideV2Data chan []schemaV2.CbgBucket, logErrorDataV2 chan *common.DetailedError) {
	defer wg.Done()
	start := time.Now()
	data, err := p.tideV2Client.GetCbgV2WithContext(ctx, userID, sessionToken, dates.Start, dates.End)
	if err != nil {
		logErrorDataV2 <- &common.DetailedError{
			Status:          errorTideV2Http.Status,
			Code:            errorTideV2Http.Code,
			Message:         errorTideV2Http.Message,
			InternalMessage: err.Error(),
		}
		tideV2Data <- nil
	} else {
		tideV2Data <- data
		logErrorDataV2 <- nil
	}
	elapsed_time := time.Now().Sub(start).Milliseconds()
	dataFromTideV2Timer.Observe(float64(elapsed_time))
}

func (p *PatientData) getBasalFromTideV2(ctx context.Context, wg *sync.WaitGroup, userID string, sessionToken string, dates *common.Date, v2Data chan []schemaV2.BasalBucket, logErrorDataV2 chan *common.DetailedError) {
	defer wg.Done()
	start := time.Now()
	data, err := p.tideV2Client.GetBasalV2WithContext(ctx, userID, sessionToken, dates.Start, dates.End)
	if err != nil {
		logErrorDataV2 <- &common.DetailedError{
			Status:          errorTideV2Http.Status,
			Code:            errorTideV2Http.Code,
			Message:         errorTideV2Http.Message,
			InternalMessage: err.Error(),
		}
		v2Data <- nil
	} else {
		v2Data <- data
		logErrorDataV2 <- nil
	}
	elapsed_time := time.Now().Sub(start).Milliseconds()
	dataFromTideV2Timer.Observe(float64(elapsed_time))
}

func (p *PatientData) getLoopModeData(ctx context.Context, wg *sync.WaitGroup, traceID string, userID string, dates *common.Date, loopModeData chan []schema.LoopModeEvent, logError chan *common.DetailedError) {
	defer wg.Done()
	start := time.Now()
	loopModes, err := p.patientDataRepository.GetLoopMode(ctx, traceID, userID, dates)
	if err != nil {
		logError <- &common.DetailedError{
			Status:          errorRunningQuery.Status,
			Code:            errorRunningQuery.Code,
			Message:         errorRunningQuery.Message,
			InternalMessage: err.Error(),
		}
		loopModeData <- loopModes
	} else {
		loopModeData <- loopModes
		logError <- nil
	}
	elapsed_time := time.Since(start).Milliseconds()
	dataFromStoreTimer.Observe(float64(elapsed_time))
}

func (p *PatientData) GetDataRangeLegacy(ctx context.Context, traceID string, userID string) (*common.Date, error) {
	if userID == "" {
		return nil, errors.New("user id is missing")
	}
	return p.patientDataRepository.GetDataRangeLegacy(ctx, traceID, userID)
}

func (p *PatientData) GetData(ctx context.Context, res *common.HttpResponseWriter, readBasalBucket bool) error {

	params, logError := p.getDataV1Params(readBasalBucket, res)
	if logError != nil {
		return res.WriteError(logError)
	}
	// Mongo iterators
	var pumpSettings *schemaV2.SettingsResult
	var iterUploads mongo.StorageIterator
	var chanApiCbgs chan []schemaV2.CbgBucket
	var chanApiBasals chan []schemaV2.BasalBucket
	var chanLoopMode chan []schema.LoopModeEvent

	var chanApiCbgError, chanApiBasalError chan *common.DetailedError
	var logErrorDataV2 *common.DetailedError
	var wg sync.WaitGroup

	var exclusions = map[string]string{
		"cbgBucket":   "cbg",
		"basalBucket": "basal",
	}
	var exclusionList []string
	groups := 0
	for key, value := range params.source {
		if value {
			groups++
			if _, ok := exclusions[key]; ok {
				exclusionList = append(exclusionList, exclusions[key])
			}
			if key == "basalBucket" {
				// Adding one group to retrieve loopModes
				groups++
			}
		}
	}
	dates := &params.dates

	writeParams := &params.writer

	sessionToken := getSessionToken(res)

	if params.includePumpSettings {
		pumpSettings, logError = p.getLatestPumpSettings(ctx, res.TraceID, params.user, writeParams, sessionToken)
		if logError != nil {
			return res.WriteError(logError)
		}
	}

	// Fetch data from patientData and V2 API (for cbg)
	chanStoreError := make(chan *common.DetailedError, 1)
	chanMongoIter := make(chan mongo.StorageIterator, 1)

	// Parallel routines
	wg.Add(groups)
	go p.getDataFromStore(ctx, &wg, res.TraceID, params.user, dates, exclusionList, chanMongoIter, chanStoreError)

	if params.source["cbgBucket"] {
		chanApiCbgs = make(chan []schemaV2.CbgBucket, 1)
		chanApiCbgError = make(chan *common.DetailedError, 1)
		go p.getCbgFromTideV2(ctx, &wg, params.user, sessionToken, dates, chanApiCbgs, chanApiCbgError)
	}
	if params.source["basalBucket"] {
		chanApiBasals = make(chan []schemaV2.BasalBucket, 1)
		chanApiBasalError = make(chan *common.DetailedError, 1)
		go p.getBasalFromTideV2(ctx, &wg, params.user, sessionToken, dates, chanApiBasals, chanApiBasalError)
		chanLoopMode = make(chan []schema.LoopModeEvent, 1)
		go p.getLoopModeData(ctx, &wg, res.TraceID, params.user, dates, chanLoopMode, chanStoreError)
	}
	go func() {
		wg.Wait()
		close(chanStoreError)
		close(chanMongoIter)
		if params.source["cbgBucket"] {
			close(chanApiCbgs)
			close(chanApiCbgError)
		}
		if params.source["basalBucket"] {
			close(chanApiBasals)
			close(chanApiBasalError)
			close(chanLoopMode)
		}
	}()

	logErrorStore := <-chanStoreError
	if logErrorStore != nil {
		return res.WriteError(logErrorStore)
	}
	if params.source["cbgBucket"] {
		logErrorDataV2 = <-chanApiCbgError
		if logErrorDataV2 != nil {
			return res.WriteError(logErrorDataV2)
		}
	}
	if params.source["basalBucket"] {
		logErrorDataV2 = <-chanApiBasalError
		if logErrorDataV2 != nil {
			return res.WriteError(logErrorDataV2)
		}
	}
	iterData := <-chanMongoIter
	var Cbgs []schemaV2.CbgBucket
	if params.source["cbgBucket"] {
		Cbgs = <-chanApiCbgs
	}
	var Basals []schemaV2.BasalBucket
	if params.source["basalBucket"] {
		Basals = <-chanApiBasals
		loopModes := <-chanLoopMode
		if len(loopModes) > 0 {
			loopModes = schema.FillLoopModeEvents(loopModes)
			Basals = basal.CleanUpBasals(Basals, loopModes)
		}
	}

	defer iterData.Close(ctx)

	return p.writeDataV1(
		ctx,
		res,
		params.includePumpSettings,
		pumpSettings,
		iterUploads,
		iterData,
		Cbgs,
		Basals,
		writeParams,
	)
}

func (p *PatientData) getDataFromStore(ctx context.Context, wg *sync.WaitGroup, traceID string, userID string, dates *common.Date, excludes []string, iterData chan mongo.StorageIterator, logError chan *common.DetailedError) {
	defer wg.Done()
	start := time.Now()
	data, err := p.patientDataRepository.GetDataInDeviceData(ctx, traceID, userID, dates, excludes)
	if err != nil {
		logError <- &common.DetailedError{
			Status:          errorRunningQuery.Status,
			Code:            errorRunningQuery.Code,
			Message:         errorRunningQuery.Message,
			InternalMessage: err.Error(),
		}
		iterData <- nil
	} else {
		logError <- nil
		iterData <- data
	}
	elapsed_time := time.Now().Sub(start).Milliseconds()
	dataFromStoreTimer.Observe(float64(elapsed_time))
}

// get session token (for history the header is found in the response and not in the request because of the v1 middelware)
// to be change of course, but for now keep it
func getSessionToken(res *common.HttpResponseWriter) string {
	// first look if old token are provided in the request
	sessionToken := res.Header.Get("x-tidepool-session-token")
	if sessionToken != "" {
		return sessionToken
	}
	// if not then
	sessionToken = strings.Trim(res.Header.Get("Authorization"), " ")
	if sessionToken != "" && strings.HasPrefix(sessionToken, "Bearer ") {
		tokenParts := strings.Split(sessionToken, " ")
		sessionToken = tokenParts[1]
	}
	return sessionToken
}

// writeFromIterV1 Common code to write
func writeFromIterV1(ctx context.Context, p *writeFromIter) error {
	var err error

	iter := p.iter
	p.iter = nil

	for iter.Next(ctx) {
		var jsonDatum []byte
		var datum map[string]interface{}

		err = iter.Decode(&datum)
		if err != nil {
			p.decode.numErrors++
			if p.decode.firstError == nil {
				p.decode.firstError = err
			}
			continue
		}
		if len(datum) > 0 {
			datumID, haveID := datum["id"].(string)
			if !haveID {
				// Ignore datum with no id, should never happend
				continue
			}

			// temp code for a DBGL1 release, allow to no change the front
			datumGuid, haveGuId := datum["guid"].(string)
			if haveGuId {
				datum["eventId"] = datumGuid
			}

			datumType, haveType := datum["type"].(string)
			if !haveType {
				// Ignore datum with no type, should never happend
				continue
			}
			uploadID, haveUploadID := datum["uploadId"].(string)
			if !haveUploadID {
				// No upload ID, abnormal situation
				continue
			}
			if datumType == "deviceEvent" {
				datumSubType, haveSubType := datum["subType"].(string)
				if haveSubType && datumSubType == "deviceParameter" {
					datumLevel, haveLevel := datum["level"]
					if haveLevel {
						intLevel, err := strconv.Atoi(fmt.Sprintf("%v", datumLevel))
						if err == nil && !common.ContainsInt(parameterLevelFilter[:], intLevel) {
							continue
						}
					}
				}
			}
			// Record the uploadID
			if !(datumType == "upload" && uploadID == datumID) {
				if !common.Contains(p.uploadIDs, uploadID) {
					p.uploadIDs = append(p.uploadIDs, uploadID)
				}
			}

			if datumType == "pumpSettings" && (p.parametersHistory != nil || p.basalSecurityProfile != nil) {
				payload := datum["payload"].(map[string]interface{})

				// Add the parameter history to the pump settings
				if p.parametersHistory != nil {
					payload["history"] = p.parametersHistory["history"]
				}

				// Add the basal security profile to the pump settings
				if p.basalSecurityProfile != nil {
					payload["basalsecurityprofile"] = p.basalSecurityProfile
				}

				datum["payload"] = payload
			}

			// Create the JSON string for this datum
			if jsonDatum, err = json.Marshal(datum); err != nil {
				if p.jsonError.firstError == nil {
					p.jsonError.firstError = err
				}
				p.jsonError.numErrors++
				continue
			}

			if p.writeCount > 0 {
				// Add the coma and line return (for readability)
				err = p.res.WriteString(",\n")
				if err != nil {
					return err
				}
			}
			err = p.res.Write(jsonDatum)
			if err != nil {
				return err
			}
			p.writeCount++
		} // else ignore
	}
	return nil
}
