package basal

import (
	"fmt"
	"time"

	"github.com/mdblp/tide-whisperer-v2/v2/schema"
	schemaV1 "github.com/tidepool-org/tide-whisperer/schema"
)

func makeBasalSample(startTime time.Time, endTime time.Time, deliveryType string, sample schema.BasalSample) schema.BasalSample {
	tz := "UTC"
	tzOffset := 0
	loc, err := time.LoadLocation(sample.Timezone)
	if err == nil {
		localizedSampleTime := startTime.In(loc)
		_, tzOffset = localizedSampleTime.Zone()
		tz = sample.Timezone
		tzOffset = tzOffset / 60
	}
	return schema.BasalSample{
		Sample: schema.Sample{
			Timestamp: startTime,
			Timezone: tz,
			TimezoneOffset: tzOffset,
		},
		Guid: sample.Guid,
		DeliveryType: deliveryType,
		Rate: sample.Rate,
		Duration: int(endTime.UnixMilli() - startTime.UnixMilli()),
	}
}

func getSample(guid string, index int, rate float64, timezone string, endTime time.Time, loopMode schemaV1.LoopModeEvent) schema.BasalSample {
	guidFormatted := fmt.Sprintf("%s-%v", guid, index)
	endBasalTime := endTime
	if loopMode.End != nil {
		endBasalTime = *loopMode.End
	}
	startTime := loopMode.Start
	if startTime.After(endBasalTime) {
		// switching start & end times if needed
		tempTime := startTime
		startTime = endBasalTime
		endBasalTime = tempTime
	}
	return makeBasalSample(
		startTime,
		endBasalTime,
		loopMode.DeliveryType,
		schema.BasalSample{
			Sample: schema.Sample{
				Timezone: timezone,
			},
			Guid: guidFormatted,
			Rate: rate,
		},
	)
}
