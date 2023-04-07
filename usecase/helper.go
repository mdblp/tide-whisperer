package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
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
		dates               common.Date
		user                string
		traceID             string
		includePumpSettings bool
		source              map[string]bool
		writer              writeFromIter
	}
)

func (p *PatientData) getDataV1Params(userID string, traceID string, startDate string, endDate string, withPumpSettings bool, readBasalBucket bool) (*apiDataParams, *common.DetailedError) {
	var err error

	dataSource := map[string]bool{
		"patientData": true,
		"basalBucket": readBasalBucket,
		"cbgBucket":   true,
	}

	// Check startDate & endDate parameter
	if startDate != "" || endDate != "" {
		var logError *common.DetailedError
		var startTime time.Time
		var endTime time.Time
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
	params := apiDataParams{
		dates: common.Date{
			Start: startDate,
			End:   endDate,
		},
		user:                userID,
		traceID:             traceID,
		includePumpSettings: withPumpSettings,
		source:              dataSource,
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
	pumpSettings *schemaV2.SettingsResult,
	iterData mongo.StorageIterator,
	Cbgs []schemaV2.CbgBucket,
	Basals []schemaV2.BasalBucket,
	writeParams *writeFromIter,
	convertToMgdl bool,
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
		err = writePumpSettings(&buff, writeParams)
		if err != nil {
			return nil, newWriteError(err)
		}
		err = writeDeviceParameterChanges(&buff, writeParams)
		if err != nil {
			return nil, newWriteError(err)
		}
	}

	common.TimeIt(ctx, "writeDataMain")
	writeParams.iter = iterData
	err = writeFromIterV1(ctx, &buff, writeParams)
	if err != nil {
		return nil, newWriteError(err)
	}
	common.TimeEnd(ctx, "writeDataMain")

	if len(Cbgs) > 0 {
		common.TimeIt(ctx, "WriteCbgs")
		writeParams.cbgs = Cbgs
		err = writeCbgs(ctx, convertToMgdl, &buff, writeParams)
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
			err = writeFromIterV1(ctx, &buff, writeParams)
			if err != nil {
				common.TimeEnd(ctx, "getUploads")
				return nil, newWriteError(err)
			}
		}
		common.TimeEnd(ctx, "getUploads")
	}

	// Silently failed theses error to the client, but record them to the log
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

func writeDeviceParameterChanges(res *bytes.Buffer, p *writeFromIter) error {
	settings := p.settings
	for _, paramChange := range settings.HistoryParameters {
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

func writePumpSettings(res *bytes.Buffer, p *writeFromIter) error {
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

func getMgdl(unit string, value float64) (string, float64) {
	if unit == MmolL {
		return MgdL, math.Round(value / MmolLToMgdLConversionFactor * MmolLToMgdLPrecisionFactor)
	}
	return unit, value
}

// Mapping V2 Bucket schema to expected V1 schema + write to output
func writeCbgs(ctx context.Context, convertToMgdl bool, res *bytes.Buffer, p *writeFromIter) error {
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
			if convertToMgdl {
				datum["units"], datum["value"] = getMgdl(sample.Units, sample.Value)
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
