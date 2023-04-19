package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mdblp/go-common/clients/status"
	orcaSchema "github.com/mdblp/orca/schema"
	schemaV2 "github.com/mdblp/tide-whisperer-v2/v2/schema"
	"github.com/tidepool-org/go-common/clients/mongo"
	internalSchema "github.com/tidepool-org/tide-whisperer/api/dto"
	"github.com/tidepool-org/tide-whisperer/common"
	"github.com/tidepool-org/tide-whisperer/schema"
)

type (
	apiDataParams struct {
		dates     common.Date
		source    map[string]bool
		writer    writeFromIter
		startTime time.Time
		endTime   time.Time
	}
)

func (p *PatientData) getDataV1Params(userID string, traceID string, startDate string, endDate string, readBasalBucket bool) (*apiDataParams, *common.DetailedError) {
	var err error

	dataSource := map[string]bool{
		"patientData": true,
		"basalBucket": readBasalBucket,
		"cbgBucket":   true,
	}

	var startTime, endTime time.Time
	// Check startDate & endDate parameter
	if startDate != "" || endDate != "" {
		var logError *common.DetailedError

		var timeRange int64 = 1 // endDate - startDate in seconds, initialized to 1 to avoid trigger an error, see below

		if startDate != "" {
			startTime, err = time.Parse(time.RFC3339Nano, startDate)
		}
		if err == nil && endDate != "" {
			endTime, err = time.Parse(time.RFC3339Nano, endDate)
		}

		if err == nil && startDate != "" && endDate != "" {
			timeRange = endTime.Unix() - startTime.Unix()
		}

		if timeRange <= 0 {
			err = fmt.Errorf("startDate is after endDate")
		}

		if err != nil {
			logError = &common.DetailedError{
				Status:          errorInvalidParameters.Status,
				Code:            errorInvalidParameters.Code,
				Message:         errorInvalidParameters.Message,
				InternalMessage: addContextToMessage("getDataV1Params", userID, traceID, err.Error()),
			}
			return nil, logError
		}
	}

	if endDate == "" {
		endTime = time.Now()
	}

	params := apiDataParams{
		dates: common.Date{
			Start: startDate,
			End:   endDate,
		},
		startTime: startTime,
		endTime:   endTime,
		source:    dataSource,
		writer: writeFromIter{
			uploadIDs: make([]string, 0, 16),
		},
	}
	return &params, nil
}

func (p *PatientData) getLatestPumpSettings(ctx context.Context, traceID string, userID string, writer *writeFromIter, token string) (*schemaV2.SettingsResult, *common.DetailedError) {
	common.TimeIt(ctx, "getLastPumpSettings")
	settings, err := p.tideV2Client.GetSettings(ctx, userID, token, true)
	if err != nil {
		logError := &common.DetailedError{
			Status:          errorRunningQuery.Status,
			Code:            errorRunningQuery.Code,
			Message:         errorRunningQuery.Message,
			InternalMessage: addContextToMessage("getLatestPumpSettings", userID, traceID, err.Error()),
		}

		switch v := err.(type) {
		case *status.StatusError:
			if v.Code != http.StatusNotFound {
				p.logger.Printf("{%s}", err.Error())
				common.TimeEnd(ctx, "getLastPumpSettings")
				return nil, logError
			}
			p.logger.Printf("{%s} - {getLatestPumpSettings: no pump settings found for user \"%s\"}", traceID, userID)
		default:
			p.logger.Printf("{%s}", err.Error())
			common.TimeEnd(ctx, "getLastPumpSettings")
			return nil, logError
		}
	}
	common.TimeEnd(ctx, "getLastPumpSettings")

	common.TimeIt(ctx, "getLatestBasalSecurityProfile")
	lastestProfile, err := p.patientDataRepository.GetLatestBasalSecurityProfile(ctx, traceID, userID)
	if err != nil {
		writer.basalSecurityProfile = nil
		p.logger.Printf("{%s} - {GetLatestBasalSecurityProfile:\"%s\"}", traceID, err)
	}
	writer.basalSecurityProfile = TransformToExposedModel(lastestProfile)
	common.TimeEnd(ctx, "getLatestBasalSecurityProfile")

	return settings, nil
}

func TransformToExposedModel(lastestProfile *schema.DbProfile) *internalSchema.Profile {
	var result *internalSchema.Profile

	if lastestProfile != nil {
		result = &internalSchema.Profile{}
		// Build start and end schedule
		// the BasalSchedule array is sorted on Start by the terminal
		for i, value := range lastestProfile.BasalSchedule {
			var elem internalSchema.Schedule
			elem.Rate = value.Rate
			elem.Start = value.Start
			if i == len(lastestProfile.BasalSchedule)-1 {
				elem.End = lastestProfile.BasalSchedule[0].Start
			} else {
				elem.End = lastestProfile.BasalSchedule[i+1].Start
			}
			result.BasalSchedule = append(result.BasalSchedule, elem)
		}
		result.Guid = lastestProfile.Guid
		result.Time = lastestProfile.Time
		result.Timezone = lastestProfile.Timezone
		result.Type = lastestProfile.Type
	}

	return result
}

func newWriteError(err error) *common.DetailedError {
	return &common.DetailedError{
		Status:          errorWriteBuffer.Status,
		Code:            errorWriteBuffer.Code,
		Message:         errorWriteBuffer.Message,
		InternalMessage: err.Error(),
	}
}
func (p *PatientData) writeDataToBuffer(
	ctx context.Context,
	traceID string,
	includePumpSettings bool,
	includeParameterChanges bool,
	pumpSettings *schemaV2.SettingsResult,
	iterData mongo.StorageIterator,
	Cbgs []schemaV2.CbgBucket,
	Basals []schemaV2.BasalBucket,
	writeParams *writeFromIter,
	bgUnit string,
	filteringParameterChanges bool,
	startTime time.Time,
	endTime time.Time,
) (*bytes.Buffer, *common.DetailedError) {
	buff := bytes.Buffer{}
	var iterUploads mongo.StorageIterator
	common.TimeIt(ctx, "writeData")
	defer common.TimeEnd(ctx, "writeData")
	// We return a JSON array, first character is: '['
	_, err := buff.WriteString("[\n")
	if err != nil {
		return nil, newWriteError(err)
	}

	if includePumpSettings && pumpSettings != nil {
		writeParams.settings = pumpSettings
		err = writePumpSettings(&buff, writeParams, bgUnit)
		if err != nil {
			return nil, newWriteError(err)
		}
	}

	if includeParameterChanges && pumpSettings != nil {
		writeParams.settings = pumpSettings
		err = writeDeviceParameterChanges(&buff, writeParams, filteringParameterChanges, bgUnit, startTime, endTime)
		if err != nil {
			return nil, newWriteError(err)
		}
	}

	common.TimeIt(ctx, "writeDataMain")
	writeParams.iter = iterData
	err = writeFromIterV1(ctx, &buff, bgUnit, writeParams)
	if err != nil {
		return nil, newWriteError(err)
	}
	common.TimeEnd(ctx, "writeDataMain")

	if len(Cbgs) > 0 {
		common.TimeIt(ctx, "WriteCbgs")
		writeParams.cbgs = Cbgs
		err = writeCbgs(ctx, bgUnit, &buff, writeParams)
		if err != nil {
			return nil, newWriteError(err)
		}
		common.TimeEnd(ctx, "WriteCbgs")
	}

	if len(Basals) > 0 {
		common.TimeIt(ctx, "writeBasals")
		writeParams.basals = Basals
		err = writeBasals(ctx, &buff, writeParams)
		if err != nil {
			return nil, newWriteError(err)
		}
		common.TimeEnd(ctx, "writeBasals")
	}

	// Fetch uploads
	if len(writeParams.uploadIDs) > 0 {
		common.TimeIt(ctx, "getUploads")
		iterUploads, err = p.patientDataRepository.GetUploadData(ctx, traceID, writeParams.uploadIDs)
		if err != nil {
			// Just log the problem, don't crash the query
			writeParams.parametersHistory = nil
			p.logger.Printf("{%s} - {GetUploadData:\"%s\"}", traceID, err)
		} else {
			defer iterUploads.Close(ctx)
			writeParams.iter = iterUploads
			err = writeFromIterV1(ctx, &buff, bgUnit, writeParams)
			if err != nil {
				common.TimeEnd(ctx, "getUploads")
				return nil, newWriteError(err)
			}
		}
		common.TimeEnd(ctx, "getUploads")
	}

	// Silently failed those error to the client, but record them to the log
	if writeParams.decode.firstError != nil {
		p.logger.Printf("{%s} - {nErrors:%d,MongoDecode:\"%s\"}", traceID, writeParams.decode.numErrors, writeParams.decode.firstError)
	}
	if writeParams.jsonError.firstError != nil {
		p.logger.Printf("{%s} - {nErrors:%d,jsonMarshall:\"%s\"}", traceID, writeParams.jsonError.numErrors, writeParams.jsonError.firstError)
	}

	// Last JSON array character:
	_, err = buff.WriteString("]\n")
	if err != nil {
		return nil, newWriteError(err)
	}
	return &buff, nil
}

func isConvertibleUnit(unit string) bool {
	return unit == MgdL || unit == MmolL
}

func writeDeviceParameterChanges(res *bytes.Buffer, p *writeFromIter, filteringParameterChanges bool, bgUnit string, startTime time.Time, endTime time.Time) error {
	settings := p.settings

	for _, paramChange := range settings.HistoryParameters {
		if filteringParameterChanges && (paramChange.Timestamp.Before(startTime) || paramChange.Timestamp.After(endTime)) {
			continue
		}
		datum := make(map[string]interface{})
		datum["id"] = uuid.New().String()
		datum["type"] = "deviceEvent"
		datum["subType"] = "deviceParameter"

		datum["time"] = paramChange.EffectiveDate
		datum["timezone"] = paramChange.Timezone
		datum["lastUpdateDate"] = paramChange.EffectiveDate

		datum["uploadId"] = uuid.New().String()
		datum["name"] = paramChange.Name
		datum["units"] = paramChange.Unit
		datum["value"] = paramChange.Value
		datum["level"] = paramChange.Level

		if paramChange.PreviousValue != "" {
			datum["previousValue"] = paramChange.PreviousValue
		}

		/*Handle conversion if bgUnit is not empty*/
		if bgUnit != "" {
			if datum["units"] != bgUnit && isConvertibleUnit(datum["units"].(string)) {
				val, err := convertToFloat64(datum["value"], datum["name"])
				if err != nil {
					return err
				}
				if datum["units"] == MgdL {
					datum["units"] = MmolL
					datum["value"] = fmt.Sprintf("%g", convertToMmol(val))
				} else {
					datum["units"] = MgdL
					datum["value"] = fmt.Sprintf("%g", convertToMgdl(val))
				}
			}

			if paramChange.PreviousUnit != bgUnit && paramChange.PreviousValue != "" && isConvertibleUnit(paramChange.PreviousUnit) {
				val, err := convertToFloat64(datum["previousValue"], datum["name"])
				if err != nil {
					return err
				}
				if paramChange.PreviousUnit == MgdL {
					datum["previousValue"] = fmt.Sprintf("%g", convertToMmol(val))
				} else {
					datum["previousValue"] = fmt.Sprintf("%g", convertToMgdl(val))
				}
			}
		}

		jsonDatum, err := json.Marshal(datum)
		if err != nil {
			if p.jsonError.firstError == nil {
				p.jsonError.firstError = err
			}
			p.jsonError.numErrors++
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
	}
	return nil
}

func convertToFloat64(value interface{}, name interface{}) (float64, error) {
	val, err := strconv.ParseFloat(value.(string), 64)
	if err != nil {
		return 0, fmt.Errorf("conversion failed because previousValue=%s for param %s is not a number", value.(string), name.(string))
	}
	return val, nil
}

func writePumpSettings(res *bytes.Buffer, p *writeFromIter, bgUnit string) error {
	settings := p.settings
	datum := make(map[string]interface{})
	datum["id"] = uuid.New().String()
	datum["type"] = "pumpSettings"
	datum["uploadId"] = uuid.New().String()
	datum["time"] = settings.Time
	datum["timezone"] = settings.Timezone
	/*TODO fetch from somewhere*/
	datum["activeSchedule"] = "Normal"
	datum["deviceId"] = settings.CurrentSettings.Device.DeviceID

	/* Perform conversion */
	if bgUnit != "" {
		for _, hp := range settings.HistoryParameters {
			if hp.Unit != bgUnit && isConvertibleUnit(hp.Unit) {
				val, err := convertToFloat64(datum["value"], datum["name"])
				if err != nil {
					/*log error but continue with existing values*/
				}
				if hp.Unit == MgdL {
					hp.Unit = MmolL
					hp.Value = fmt.Sprintf("%g", convertToMmol(val))
				} else {
					hp.Unit = MgdL
					hp.Value = fmt.Sprintf("%g", convertToMgdl(val))
				}
			}

			if hp.PreviousUnit != bgUnit && isConvertibleUnit(hp.PreviousUnit) {
				val, err := convertToFloat64(datum["previousValue"], datum["name"])
				if err != nil {
					return err
				}
				if hp.PreviousUnit == MgdL {
					hp.PreviousUnit = MmolL
					hp.PreviousValue = fmt.Sprintf("%g", convertToMmol(val))
				} else {
					hp.PreviousUnit = MgdL
					hp.PreviousValue = fmt.Sprintf("%g", convertToMgdl(val))
				}
			}
		}
		for _, p := range settings.CurrentSettings.Parameters {
			if p.Unit != bgUnit && isConvertibleUnit(p.Unit) {
				val, err := convertToFloat64(datum["value"], datum["name"])
				if err != nil {
					return err
				}
				if p.Unit == MgdL {
					p.Unit = MmolL
					p.Value = fmt.Sprintf("%g", convertToMmol(val))
				} else {
					p.Unit = MgdL
					p.Value = fmt.Sprintf("%g", convertToMgdl(val))
				}
			}
		}
	}

	groupedHistoryParameters := groupByChangeDate(settings.HistoryParameters)
	payload := map[string]interface{}{
		"basalsecurityprofile": p.basalSecurityProfile,
		"cgm":                  settings.CurrentSettings.Cgm,
		"device":               settings.CurrentSettings.Device,
		"pump":                 settings.CurrentSettings.Pump,
		"parameters":           settings.CurrentSettings.Parameters,
		"history":              groupedHistoryParameters,
	}
	datum["payload"] = payload

	jsonDatum, err := json.Marshal(datum)
	if err != nil {
		if p.jsonError.firstError == nil {
			p.jsonError.firstError = err
		}
		p.jsonError.numErrors++
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
	return nil
}

type GroupedHistoryParameters struct {
	ChangeDate time.Time                     `json:"changeDate"`
	Parameters []orcaSchema.HistoryParameter `json:"parameters"`
}

func groupByChangeDate(parameters []orcaSchema.HistoryParameter) []GroupedHistoryParameters {
	//Group parameters by Timestamp (corresponding to the moment where the request is sent to yourloops when leaving
	// param edition, vs EffectiveDate which is the time when the parameter is changed on the device)
	// Old implementation was grouping by same Timestamp -> max of EffectiveDate which is maybe not the best so we
	// decided to sort by Timestamp only (makes more sense).
	temporaryMap := make(map[string][]orcaSchema.HistoryParameter, 0)
	for _, p := range parameters {
		mapTime := p.Timestamp.Format("2006-01-02")
		if temporaryMap[mapTime] == nil {
			temporaryMap[mapTime] = []orcaSchema.HistoryParameter{p}
		} else {
			temporaryMap[mapTime] = append(temporaryMap[mapTime], p)
		}
	}
	finalArray := make([]GroupedHistoryParameters, 0)
	for _, p := range temporaryMap {
		finalArray = append(finalArray, GroupedHistoryParameters{
			ChangeDate: p[0].Timestamp,
			Parameters: p,
		})
	}
	return finalArray
}

func convertToMgdl(value float64) float64 {
	return math.Round(value * MmolLToMgdLConversionFactor)
}

func convertToMmol(value float64) float64 {
	roundedValue := math.Round(value / MmolLToMgdLConversionFactor * MmolLToMgdLPrecisionFactor)
	floatValue := roundedValue / MmolLToMgdLPrecisionFactor
	return floatValue
}

// Mapping V2 Bucket schema to expected V1 schema + write to output
func writeCbgs(ctx context.Context, bgUnit string, res *bytes.Buffer, p *writeFromIter) error {
	var err error
	for _, bucket := range p.cbgs {
		for i, sample := range bucket.Samples {
			datum := make(map[string]interface{})
			// Building a fake id (bucket.Id/range index)
			datum["id"] = fmt.Sprintf("cbg_%s_%d", bucket.Id, i)
			datum["type"] = "cbg"
			datum["time"] = sample.Timestamp
			datum["timezone"] = sample.Timezone
			datum["units"] = sample.Units
			datum["value"] = sample.Value
			if bgUnit != "" && datum["units"] != bgUnit {
				if datum["units"] == MmolL {
					datum["units"] = MgdL
					datum["value"] = convertToMgdl(sample.Value)
				} else {
					datum["units"] = MmolL
					datum["value"] = convertToMmol(sample.Value)
				}
			}
			jsonDatum, err := json.Marshal(datum)
			if err != nil {
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
		}
	}
	return err
}

// Mapping V2 Bucket schema to expected V1 schema + write to output
func writeBasals(ctx context.Context, res *bytes.Buffer, p *writeFromIter) error {
	var err error
	for _, bucket := range p.basals {
		for i, sample := range bucket.Samples {
			datum := make(map[string]interface{})
			// Building a fake id (bucket.Id/range index)
			datum["id"] = fmt.Sprintf("basal_%s_%d", bucket.Id, i)
			datum["type"] = "basal"
			datum["time"] = sample.Timestamp
			datum["timezone"] = sample.Timezone
			datum["deliveryType"] = sample.DeliveryType
			datum["rate"] = sample.Rate
			datum["duration"] = sample.Duration
			jsonDatum, err := json.Marshal(datum)
			if err != nil {
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
		}
	}
	return err
}
