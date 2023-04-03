package basal

import (
	"testing"
	"time"

	"github.com/mdblp/tide-whisperer-v2/v2/schema"
	"github.com/stretchr/testify/assert"
	schemaV1 "github.com/tidepool-org/tide-whisperer/schema"
)


func Test_makeBasalSample(t *testing.T) {
	timeStart := time.Date(2023, time.March ,14, 0, 0, 0, 0, time.UTC)
	timeEnd := time.Date(2023, time.March, 14, 1, 0, 0, 0, time.UTC)
	realTzSample := schema.BasalSample{
		Sample: schema.Sample{
			Timestamp: time.Now(),
			Timezone: "Europe/Paris",
			TimezoneOffset: 0,
		},
		Guid: "Test",
		DeliveryType: "not a test",
		Rate: 1.5,
		Duration: 60000,
	}
	expectedRealTzSample := schema.BasalSample{
		Sample: schema.Sample{
			Timestamp: timeStart,
			Timezone: "Europe/Paris",
			TimezoneOffset: 60,
		},
		Guid: "Test",
		DeliveryType: "test",
		Rate: 1.5,
		Duration: 60 * 60 * 1000,
	}

	badTzSample := schema.BasalSample{
		Sample: schema.Sample{
			Timestamp: time.Now(),
			Timezone: "XXXX",
			TimezoneOffset: 90,
		},
		Guid: "Test",
		DeliveryType: "not a test",
		Rate: 1.5,
		Duration: 60000,
	}
	expectedBadTz := schema.BasalSample{
		Sample: schema.Sample{
			Timestamp: timeStart,
			Timezone: "UTC",
			TimezoneOffset: 0,
		},
		Guid: "Test",
		DeliveryType: "test",
		Rate: 1.5,
		Duration: 60 * 60 * 1000,
	}

	tests := []struct {
        inputSample schema.BasalSample
        expectedSample schema.BasalSample
    }{
        {inputSample: realTzSample, expectedSample: expectedRealTzSample},
        {inputSample: badTzSample, expectedSample: expectedBadTz},
    }

	for _, tc := range tests {
		builtBasal := makeBasalSample(timeStart, timeEnd, "test", tc.inputSample)
		assert.Equal(t, tc.expectedSample, builtBasal, "Unexpected basal buils")
	}
}

func Test_getSample(t *testing.T) {
	timeStart := time.Date(2023, time.March ,14, 0, 0, 0, 0, time.UTC)
	timeEnd30Min := time.Date(2023, time.March, 14, 0, 30, 0, 0, time.UTC)
	timeEnd60Min := time.Date(2023, time.March, 14, 1, 00, 0, 0, time.UTC)
	timeEndBeforeStart := time.Date(2023, time.March, 13, 23, 30, 0, 0, time.UTC)
	loopModeWithNilEnd := schemaV1.NewLoopModeEvent(timeStart, nil, "auto")
	loopModeWithEnd := schemaV1.NewLoopModeEvent(timeStart, &timeEnd30Min, "auto")
	timezone := "Europe/Paris"
	expected30Min := schema.BasalSample{
		Sample: schema.Sample{
			Timestamp: timeStart,
			Timezone: timezone,
			TimezoneOffset: 60,
		},
		Guid: "1234-10",
		DeliveryType: "auto",
		Rate: 1.5,
		Duration: 30 * 60 * 1000,
	}
	expected60Min := schema.BasalSample{
		Sample: schema.Sample{
			Timestamp: timeStart,
			Timezone: timezone,
			TimezoneOffset: 60,
		},
		Guid: "1234-10",
		DeliveryType: "auto",
		Rate: 1.5,
		Duration: 60 * 60 * 1000,
	}
	expectedSwitched := schema.BasalSample{
		Sample: schema.Sample{
			Timestamp: timeEndBeforeStart,
			Timezone: timezone,
			TimezoneOffset: 60,
		},
		Guid: "1234-10",
		DeliveryType: "auto",
		Rate: 1.5,
		Duration: 30 * 60 * 1000,
	}

	tests := []struct {
        inputEnd time.Time
		inputLoopMode schemaV1.LoopModeEvent
        expectedSample schema.BasalSample
    }{
        {inputEnd: timeEnd30Min, inputLoopMode: loopModeWithNilEnd, expectedSample: expected30Min},
		{inputEnd: timeEnd60Min, inputLoopMode: loopModeWithNilEnd, expectedSample: expected60Min},
		{inputEnd: timeEndBeforeStart, inputLoopMode: loopModeWithNilEnd, expectedSample: expectedSwitched},
        {inputEnd: timeEnd30Min, inputLoopMode: loopModeWithEnd, expectedSample: expected30Min},
		{inputEnd: timeEnd60Min, inputLoopMode: loopModeWithEnd, expectedSample: expected30Min},
		{inputEnd: timeEndBeforeStart, inputLoopMode: loopModeWithEnd, expectedSample: expected30Min},
    }
	for _, tc := range tests {
		sample := getSample("1234", 10, 1.5, timezone, tc.inputEnd, tc.inputLoopMode)
		assert.Equal(t, tc.expectedSample, sample, "Unexpected sample retrieved")
	}
}