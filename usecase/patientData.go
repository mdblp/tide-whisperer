package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	tideV2Client "github.com/mdblp/tide-whisperer-v2/v2/client/tidewhisperer"
	schemaV2 "github.com/mdblp/tide-whisperer-v2/v2/schema"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tidepool-org/go-common/clients/mongo"
	goComMgo "github.com/tidepool-org/go-common/clients/mongo"
	"github.com/tidepool-org/tide-whisperer/common"
	"github.com/tidepool-org/tide-whisperer/schema"
	"github.com/tidepool-org/tide-whisperer/usecase/basal"
)

const (
	MmolL = "mmol/L"
	MgdL  = "mg/dL"

	MmolLToMgdLConversionFactor float64 = 18.01577
	MmolLToMgdLPrecisionFactor  float64 = 10.0
)

var (
	errorRunningQuery      = common.DetailedError{Status: http.StatusInternalServerError, Code: "data_store_error", Message: "internal server error"}
	errorTideV2Http        = common.DetailedError{Status: http.StatusInternalServerError, Code: "tidev2_error", Message: "internal server error"}
	errorWriteBuffer       = common.DetailedError{Status: http.StatusInternalServerError, Code: "write_error", Message: "internal server error"}
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
	readBasalBucket       bool
}

func NewPatientDataUseCase(logger *log.Logger, tideV2Client tideV2Client.ClientInterface, patientDataRepository PatientDataRepository, readBasalBucket bool) *PatientData {
	return &PatientData{
		patientDataRepository: patientDataRepository,
		logger:                logger,
		tideV2Client:          tideV2Client,
		readBasalBucket:       readBasalBucket,
	}
}

func (p *PatientData) getCbgFromTideV2(ctx context.Context, wg *sync.WaitGroup, traceID string, userID string, sessionToken string, dates *common.Date, channel chan interface{}) {
	defer wg.Done()
	start := time.Now()
	data, err := p.tideV2Client.GetCbgV2WithContext(ctx, userID, sessionToken, dates.Start, dates.End)
	if err != nil {
		channel <- &common.DetailedError{
			Status:          errorTideV2Http.Status,
			Code:            errorTideV2Http.Code,
			Message:         errorTideV2Http.Message,
			InternalMessage: addContextToMessage("getCbgFromTideV2", userID, traceID, err.Error()),
		}
	} else {
		channel <- data
	}
	elapsedTime := time.Since(start).Milliseconds()
	dataFromTideV2Timer.Observe(float64(elapsedTime))
}

func (p *PatientData) getBasalFromTideV2(ctx context.Context, wg *sync.WaitGroup, traceID string, userID string, sessionToken string, dates *common.Date, channel chan interface{}) {
	defer wg.Done()
	start := time.Now()
	data, err := p.tideV2Client.GetBasalV2WithContext(ctx, userID, sessionToken, dates.Start, dates.End)
	if err != nil {
		channel <- &common.DetailedError{
			Status:          errorTideV2Http.Status,
			Code:            errorTideV2Http.Code,
			Message:         errorTideV2Http.Message,
			InternalMessage: addContextToMessage("getBasalFromTideV2", userID, traceID, err.Error()),
		}
	} else {
		channel <- data
	}
	elapsedTime := time.Since(start).Milliseconds()
	dataFromTideV2Timer.Observe(float64(elapsedTime))
}

func (p *PatientData) getLoopModeData(ctx context.Context, wg *sync.WaitGroup, traceID string, userID string, dates *common.Date, channel chan interface{}) {
	defer wg.Done()
	start := time.Now()
	loopModes, err := p.patientDataRepository.GetLoopMode(ctx, traceID, userID, dates)
	if err != nil {
		channel <- &common.DetailedError{
			Status:          errorRunningQuery.Status,
			Code:            errorRunningQuery.Code,
			Message:         errorRunningQuery.Message,
			InternalMessage: addContextToMessage("getBasalFromTideV2", userID, traceID, err.Error()),
		}
	} else {
		channel <- loopModes
	}
	elapsedTime := time.Since(start).Milliseconds()
	dataFromStoreTimer.Observe(float64(elapsedTime))
}

func (p *PatientData) GetDataRangeLegacy(ctx context.Context, traceID string, userID string) (*common.Date, error) {
	if userID == "" {
		return nil, errors.New("user id is missing")
	}
	return p.patientDataRepository.GetDataRangeLegacy(ctx, traceID, userID)
}

/*Temporary hack until we remove DetailedError
TODO : refactor DetailedError to stop using it and use go-common v2 errors
By doing this we will use the error interface and we will be able to wrap errors with additional
context using fmt.Errorf("context: %w", err) */
func addContextToMessage(methodName string, userID string, traceID string, message string) string {
	return fmt.Sprintf("%s failed: user=[%s], traceID=[%s] : %v", methodName, userID, traceID, message)
}

type GetDataArgs struct {
	Ctx                   context.Context
	UserID                string
	TraceID               string
	StartDate             string
	EndDate               string
	WithPumpSettings      bool
	WithParametersChanges bool
	SessionToken          string
	ConvertToMgdl         bool
}

func (p *PatientData) GetData(args GetDataArgs) (*bytes.Buffer, *common.DetailedError) {
	params, err := p.getDataV1Params(args.UserID, args.TraceID, args.StartDate, args.EndDate, p.readBasalBucket)
	if err != nil {
		return nil, err
	}
	var pumpSettings *schemaV2.SettingsResult

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

	if args.WithPumpSettings || args.WithParametersChanges {
		pumpSettings, err = p.getLatestPumpSettings(args.Ctx, args.TraceID, args.UserID, writeParams, args.SessionToken)
		if err != nil {
			return nil, err
		}
	}

	// Fetch data from patientData and V2 API (for cbg)
	channel := make(chan interface{})

	// Parallel routines
	wg.Add(groups)
	go p.getDataFromStore(args.Ctx, &wg, args.TraceID, args.UserID, dates, exclusionList, channel)

	if params.source["cbgBucket"] {
		go p.getCbgFromTideV2(args.Ctx, &wg, args.TraceID, args.UserID, args.SessionToken, dates, channel)
	}
	if params.source["basalBucket"] {
		go p.getBasalFromTideV2(args.Ctx, &wg, args.TraceID, args.UserID, args.SessionToken, dates, channel)
		go p.getLoopModeData(args.Ctx, &wg, args.TraceID, args.UserID, dates, channel)
	}

	/*To stop the range loop reading channels once all data are read from it*/
	/*This is due to the fact that writing into a channel will terminate once a read is done
	with unbuffered channels.*/
	go func() {
		wg.Wait()
		close(channel)
	}()

	var iterData goComMgo.StorageIterator
	var cbgs []schemaV2.CbgBucket
	var basals []schemaV2.BasalBucket
	var loopModes []schema.LoopModeEvent

	for chanData := range channel {
		switch d := chanData.(type) {
		case *common.DetailedError:
			return nil, d
		case goComMgo.StorageIterator:
			iterData = d
		case []schemaV2.CbgBucket:
			cbgs = d
		case []schemaV2.BasalBucket:
			basals = d
		case []schema.LoopModeEvent:
			loopModes = d
		}
	}

	if len(loopModes) > 0 {
		loopModes = schema.FillLoopModeEvents(loopModes)
		basals = basal.CleanUpBasals(basals, loopModes)
	}

	defer iterData.Close(args.Ctx)

	return p.writeDataToBuffer(
		args.Ctx,
		args.TraceID,
		args.WithPumpSettings,
		args.WithParametersChanges,
		pumpSettings,
		iterData,
		cbgs,
		basals,
		writeParams,
		args.ConvertToMgdl,
	)
}

func (p *PatientData) getDataFromStore(ctx context.Context, wg *sync.WaitGroup, traceID string, userID string, dates *common.Date, excludes []string, channel chan interface{}) {
	defer wg.Done()
	start := time.Now()
	data, err := p.patientDataRepository.GetDataInDeviceData(ctx, traceID, userID, dates, excludes)
	if err != nil {
		channel <- &common.DetailedError{
			Status:          errorRunningQuery.Status,
			Code:            errorRunningQuery.Code,
			Message:         errorRunningQuery.Message,
			InternalMessage: addContextToMessage("getDataFromStore", userID, traceID, err.Error()),
		}
	} else {
		channel <- data
	}
	elapsedTime := time.Since(start).Milliseconds()
	dataFromStoreTimer.Observe(float64(elapsedTime))
}

// writeFromIterV1 Common code to write
func writeFromIterV1(ctx context.Context, res *bytes.Buffer, p *writeFromIter) error {
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

			/*perform mmol -> mgdl conversion if needed*/
			switch datum["type"] {
			case "smbg", "deviceEvent":
				/*verify units is mmol, mainly for deviceEvent*/
				if datum["units"] == MmolL {
					datum["units"], datum["value"] = getMgdl(datum["units"].(string), datum["value"].(float64))
				}
			case "wizard":
				/*For wizard, we don't have anymore fields in mmol, so we're changing the unit but no conversion is done.
				The associated bolus is separated and will be converted in another function.*/
				if datum["units"] == MmolL {
					datum["units"] = MgdL
				}
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
				_, err = res.WriteString(",\n")
				if err != nil {
					return err
				}
			}
			_, err = res.Write(jsonDatum)
			if err != nil {
				return err
			}
			p.writeCount++
		} // else ignore
	}
	return nil
}
