package basal

import (
	"testing"
	"time"

	"github.com/mdblp/tide-whisperer-v2/v2/schema"
	"github.com/stretchr/testify/assert"
	schemaV1 "github.com/tidepool-org/tide-whisperer/schema"
)

func TestCleanUpBasals_LoopModeJoin(t *testing.T) {
	startLoop1 := time.Date(2023, time.March ,11, 0, 0, 0, 0, time.UTC)
	endLoop1 := time.Date(2023, time.March ,12, 0, 5, 0, 0, time.UTC)
	startLoop2 := time.Date(2023, time.March ,12, 0, 15, 0, 0, time.UTC)
	endLoop2 := time.Date(2023, time.March ,12, 12, 0, 0, 0, time.UTC)
	startLoop3 := time.Date(2023, time.March ,13, 8, 0, 0, 0, time.UTC)
	loopModes := []schemaV1.LoopModeEvent{
		schemaV1.NewLoopModeEvent(startLoop1, &endLoop1, "automated"),
		schemaV1.NewLoopModeEvent(endLoop1, &startLoop2, "scheduled"),
		schemaV1.NewLoopModeEvent(startLoop2, &endLoop2, "automated"),
		schemaV1.NewLoopModeEvent(endLoop2, &startLoop3, "scheduled"),
		schemaV1.NewLoopModeEvent(startLoop3, nil, "automated"),
	}

	basals := []schema.BasalBucket{
		{
			Day: time.Date(2023, time.March, 12, 0, 0, 0, 0, time.UTC),
			Samples: []schema.BasalSample{
				{
					Sample: schema.Sample{
						Timestamp: time.Date(2023, time.March, 12, 0, 0, 0, 0, time.UTC),
						Timezone: "Europe/Paris",
					},
					Duration: 10 * 60 * 1000,
					Rate: 1,
					Guid: "id1",
				},
				{
					Sample: schema.Sample{
						Timestamp: time.Date(2023, time.March, 12, 0, 10, 0, 0, time.UTC),
						Timezone: "Europe/Paris",
					},
					Duration: 10 * 60 * 1000,
					Rate: 0.5,
					Guid: "id2",
				},
				{
					Sample: schema.Sample{
						Timestamp: time.Date(2023, time.March, 12, 0, 30, 0, 0, time.UTC),
						Timezone: "Europe/Paris",
					},
					Duration: 10 * 60 * 1000,
					Rate: 0.1,
					Guid: "id3",
				},
			},
		},
		{
			Day: time.Date(2023, time.March, 13, 0, 0, 0, 0, time.UTC),
			Samples: []schema.BasalSample{
				{
					Sample: schema.Sample{
						Timestamp: time.Date(2023, time.March, 13, 0, 0, 0, 0, time.UTC),
						Timezone: "Europe/Paris",
					},
					Duration: 10 * 60 * 1000,
					Rate: 1,
					Guid: "id4",
				},
				{
					Sample: schema.Sample{
						Timestamp: time.Date(2023, time.March, 13, 0, 10, 0, 0, time.UTC),
						Timezone: "Europe/Paris",
					},
					Duration: 10 * 60 * 1000,
					Rate: 0.5,
					Guid: "id5",
				},
				{
					Sample: schema.Sample{
						Timestamp: time.Date(2023, time.March, 13, 0, 30, 0, 0, time.UTC),
						Timezone: "Europe/Paris",
					},
					Duration: 10 * 60 * 1000,
					Rate: 0.1,
					Guid: "id6",
				},
			},
		},
	}

	finalBasals := CleanUpBasals(basals, loopModes)
	expected := []schema.BasalBucket{
			{
				Day: time.Date(2023, time.March, 12, 0, 0, 0, 0, time.UTC),
				Samples: []schema.BasalSample{
					{
						Sample: schema.Sample{
							Timestamp: time.Date(2023, time.March, 12, 0, 0, 0, 0, time.UTC),
							Timezone: "Europe/Paris",
							TimezoneOffset: 60,
						},
						Duration: 5 * 60 * 1000,
						Rate: 1,
						Guid: "id1-0",
						DeliveryType: "automated",
					},
					{
						Sample: schema.Sample{
							Timestamp: endLoop1,
							Timezone: "Europe/Paris",
							TimezoneOffset: 60,
						},
						Duration: 5 * 60 * 1000,
						Rate: 1,
						Guid: "id1-1",
						DeliveryType: "scheduled",
					},
					{
						Sample: schema.Sample{
							Timestamp: time.Date(2023, time.March, 12, 0, 10, 0, 0, time.UTC),
							Timezone: "Europe/Paris",
							TimezoneOffset: 60,
						},
						Duration: 5 * 60 * 1000,
						Rate: 0.5,
						Guid: "id2-2",
						DeliveryType: "scheduled",
					},
					{
						Sample: schema.Sample{
							Timestamp: time.Date(2023, time.March, 12, 0, 15, 0, 0, time.UTC),
							Timezone: "Europe/Paris",
							TimezoneOffset: 60,
						},
						Duration: 5 * 60 * 1000,
						Rate: 0.5,
						Guid: "id2-3",
						DeliveryType: "automated",
					},
					{
						Sample: schema.Sample{
							Timestamp: time.Date(2023, time.March, 12, 0, 20, 0, 0, time.UTC),
							Timezone: "Europe/Paris",
							TimezoneOffset: 60,
						},
						Duration: 10 * 60 * 1000,
						Rate: 0,
						Guid: "id3-4",
						DeliveryType: "automated",
					},
					{
						Sample: schema.Sample{
							Timestamp: time.Date(2023, time.March, 12, 0, 30, 0, 0, time.UTC),
							Timezone: "Europe/Paris",
							TimezoneOffset: 60,
						},
						Duration: 10 * 60 * 1000,
						Rate: 0.1,
						Guid: "id3-5",
						DeliveryType: "automated",
					},
				},
			},
			{
				Day: time.Date(2023, time.March, 13, 0, 0, 0, 0, time.UTC),
				Samples: []schema.BasalSample{
					{
						Sample: schema.Sample{
							Timestamp: time.Date(2023, time.March, 12, 0, 40, 0, 0, time.UTC),
							Timezone: "Europe/Paris",
							TimezoneOffset: 60,
						},
						Duration: 20 * 60 * 1000 + (11 * 60 * 60 *1000),
						Rate: 0,
						Guid: "id4-6",
						DeliveryType: "automated",
					},
					{
						Sample: schema.Sample{
							Timestamp: time.Date(2023, time.March, 12, 12, 0, 0, 0, time.UTC),
							Timezone: "Europe/Paris",
							TimezoneOffset: 60,
						},
						Duration: (12 * 60 * 60 *1000),
						Rate: 0,
						Guid: "id4-7",
						DeliveryType: "scheduled",
					},
					{
						Sample: schema.Sample{
							Timestamp: time.Date(2023, time.March, 13, 0, 0, 0, 0, time.UTC),
							Timezone: "Europe/Paris",
							TimezoneOffset: 60,
						},
						Duration: 10 * 60 * 1000,
						Rate: 1,
						Guid: "id4-8",
						DeliveryType: "scheduled",
					},
					{
						Sample: schema.Sample{
							Timestamp: time.Date(2023, time.March, 13, 0, 10, 0, 0, time.UTC),
							Timezone: "Europe/Paris",
							TimezoneOffset: 60,
						},
						Duration: 10 * 60 * 1000,
						Rate: 0.5,
						Guid: "id5-9",
						DeliveryType: "scheduled",
					},
					{
						Sample: schema.Sample{
							Timestamp: time.Date(2023, time.March, 13, 0, 20, 0, 0, time.UTC),
							Timezone: "Europe/Paris",
							TimezoneOffset: 60,
						},
						Duration: 10 * 60 * 1000,
						Rate: 0,
						Guid: "id6-10",
						DeliveryType: "scheduled",
					},
					{
						Sample: schema.Sample{
							Timestamp: time.Date(2023, time.March, 13, 0, 30, 0, 0, time.UTC),
							Timezone: "Europe/Paris",
							TimezoneOffset: 60,
						},
						Duration: 10 * 60 * 1000,
						Rate: 0.1,
						Guid: "id6-11",
						DeliveryType: "scheduled",
					},
				},
			},
	}
	assert.Equal(t, expected, finalBasals, "Unexpected basals cleanup")
}


func TestCleanUpBasals_BasalOverlap(t *testing.T) {
	startLoop1 := time.Date(2023, time.March ,11, 0, 0, 0, 0, time.UTC)
	endLoop1 := time.Date(2023, time.March ,12, 0, 5, 0, 0, time.UTC)
	startLoop2 := time.Date(2023, time.March ,12, 0, 15, 0, 0, time.UTC)
	endLoop2 := time.Date(2023, time.March ,12, 12, 0, 0, 0, time.UTC)
	loopModes := []schemaV1.LoopModeEvent{
		schemaV1.NewLoopModeEvent(startLoop1, &endLoop1, "automated"),
		schemaV1.NewLoopModeEvent(endLoop1, &startLoop2, "scheduled"),
		schemaV1.NewLoopModeEvent(startLoop2, &endLoop2, "automated"),
	}

	basals := []schema.BasalBucket{
		{
			Day: time.Date(2023, time.March, 12, 0, 0, 0, 0, time.UTC),
			Samples: []schema.BasalSample{
				{
					Sample: schema.Sample{
						Timestamp: time.Date(2023, time.March, 12, 0, 0, 0, 0, time.UTC),
						Timezone: "Europe/Paris",
					},
					Duration: 10 * 60 * 1000,
					Rate: 1,
					Guid: "id1",
				},
				{
					Sample: schema.Sample{
						Timestamp: time.Date(2023, time.March, 12, 0, 5, 0, 0, time.UTC),
						Timezone: "Europe/Paris",
					},
					Duration: 10 * 60 * 1000,
					Rate: 0.5,
					Guid: "id2",
				},
				{
					Sample: schema.Sample{
						Timestamp: time.Date(2023, time.March, 12, 0, 10, 0, 0, time.UTC),
						Timezone: "Europe/Paris",
					},
					Duration: 10 * 60 * 1000,
					Rate: 0.1,
					Guid: "id3",
				},
			},
		},
	}

	finalBasals := CleanUpBasals(basals, loopModes)
	expected := []schema.BasalBucket{
			{
				Day: time.Date(2023, time.March, 12, 0, 0, 0, 0, time.UTC),
				Samples: []schema.BasalSample{
					{
						Sample: schema.Sample{
							Timestamp: time.Date(2023, time.March, 12, 0, 0, 0, 0, time.UTC),
							Timezone: "Europe/Paris",
							TimezoneOffset: 60,
						},
						Duration: 5 * 60 * 1000,
						Rate: 1,
						Guid: "id1-0",
						DeliveryType: "automated",
					},
					{
						Sample: schema.Sample{
							Timestamp: time.Date(2023, time.March, 12, 0, 5, 0, 0, time.UTC),
							Timezone: "Europe/Paris",
							TimezoneOffset: 60,
						},
						Duration: 5 * 60 * 1000,
						Rate: 0.5,
						Guid: "id2-2",
						DeliveryType: "scheduled",
					},
					{
						Sample: schema.Sample{
							Timestamp: time.Date(2023, time.March, 12, 0, 10, 0, 0, time.UTC),
							Timezone: "Europe/Paris",
							TimezoneOffset: 60,
						},
						Duration: 5 * 60 * 1000,
						Rate: 0.1,
						Guid: "id3-3",
						DeliveryType: "scheduled",
					},
					{
						Sample: schema.Sample{
							Timestamp: time.Date(2023, time.March, 12, 0, 15, 0, 0, time.UTC),
							Timezone: "Europe/Paris",
							TimezoneOffset: 60,
						},
						Duration: 5 * 60 * 1000,
						Rate: 0.1,
						Guid: "id3-4",
						DeliveryType: "automated",
					},
				},
			},
	}
	assert.Equal(t, expected, finalBasals, "Unexpected basals cleanup")
}


func TestFillMissingBasal(t *testing.T) {
	startLoop1 := time.Date(2023, time.March ,11, 0, 0, 0, 0, time.UTC)
	endLoop1 := time.Date(2023, time.March ,12, 0, 5, 0, 0, time.UTC)
	startLoop2 := time.Date(2023, time.March ,12, 0, 15, 0, 0, time.UTC)
	endLoop2 := time.Date(2023, time.March ,12, 12, 0, 0, 0, time.UTC)
	loopModes := []schemaV1.LoopModeEvent{
		schemaV1.NewLoopModeEvent(startLoop1, &endLoop1, "automated"),
		schemaV1.NewLoopModeEvent(endLoop1, &startLoop2, "scheduled"),
		schemaV1.NewLoopModeEvent(startLoop2, &endLoop2, "automated"),
	}
	currentBasal := schema.BasalSample{
		Sample: schema.Sample{
			Timestamp: time.Date(2023, time.March, 12, 1, 0, 0, 0, time.UTC),
			Timezone: "Europe/Paris",
		},
		Duration: 10 * 60 * 1000,
		Rate: 0.5,
		Guid: "id2",
	}
	previousEndBasalTime := time.Date(2023, time.March ,12, 0, 0, 0, 0, time.UTC)
	newEndTime, newCount, filledBasals := fillMissingBasal(currentBasal, previousEndBasalTime, 1, loopModes)
	expectedNewEndTime := currentBasal.Timestamp
	assert.Equal(t, expectedNewEndTime, newEndTime, "Unexpected endTime returned")
	expectedFilledBasal := []schema.BasalSample{
		{
			Sample: schema.Sample{
				Timestamp: previousEndBasalTime,
				Timezone: "Europe/Paris",
				TimezoneOffset: 60,
			},
			Duration: 5 * 60 * 1000,
			Rate: 0,
			Guid: "id2-1",
			DeliveryType: "automated",
		},
		{
			Sample: schema.Sample{
				Timestamp: endLoop1,
				Timezone: "Europe/Paris",
				TimezoneOffset: 60,
			},
			Duration: 10 * 60 * 1000,
			Rate: 0,
			Guid: "id2-2",
			DeliveryType: "scheduled",
		},
		{
			Sample: schema.Sample{
				Timestamp: startLoop2,
				Timezone: "Europe/Paris",
				TimezoneOffset: 60,
			},
			Duration: 45 * 60 * 1000,
			Rate: 0,
			Guid: "id2-3",
			DeliveryType: "automated",
		},
	}
	assert.Equal(t, len(expectedFilledBasal) + 1, newCount, "Unexpected basal count returned")
	assert.Equal(t, expectedFilledBasal, filledBasals, "Unexpected filled basal returned")

}


func TestFixOverlappingBasal(t *testing.T) {
	previousSamples := []schema.BasalSample{
		{
			Sample: schema.Sample{
				Timestamp: time.Date(2023, time.March, 12, 0, 0, 0, 0, time.UTC),
				Timezone: "Europe/Paris",
				TimezoneOffset: 60,
			},
			Duration: 15 * 60 * 1000,
			Rate: 0,
			Guid: "id1-1",
			DeliveryType: "automated",
		},
		{
			Sample: schema.Sample{
				Timestamp: time.Date(2023, time.March, 12, 0, 15, 0, 0, time.UTC),
				Timezone: "Europe/Paris",
				TimezoneOffset: 60,
			},
			Duration: 35 * 60 * 1000,
			Rate: 0,
			Guid: "id2-2",
			DeliveryType: "automated",
		},
		{
			Sample: schema.Sample{
				Timestamp: time.Date(2023, time.March, 12, 0, 50, 0, 0, time.UTC),
				Timezone: "Europe/Paris",
				TimezoneOffset: 60,
			},
			Duration: 15 * 60 * 1000, // ends at 01:05:00
			Rate: 0,
			Guid: "id3-3",
			DeliveryType: "automated",
		},
	}
	currentBasal := schema.BasalSample{
		Sample: schema.Sample{
			Timestamp: time.Date(2023, time.March, 12, 0, 45, 0, 0, time.UTC),
			Timezone: "Europe/Paris",
			TimezoneOffset: 60,
		},
		Duration: 10 * 60 * 1000,
		Rate: 0.5,
		Guid: "id2",
	}
	fixedBasals := fixOverlappingBasal(previousSamples, currentBasal)

	expectedFixedBasal := []schema.BasalSample{
		{
			Sample: schema.Sample{
				Timestamp: time.Date(2023, time.March, 12, 0, 0, 0, 0, time.UTC),
				Timezone: "Europe/Paris",
				TimezoneOffset: 60,
			},
			Duration: 15 * 60 * 1000,
			Rate: 0,
			Guid: "id1-1",
			DeliveryType: "automated",
		},
		{
			Sample: schema.Sample{
				Timestamp: time.Date(2023, time.March, 12, 0, 15, 0, 0, time.UTC),
				Timezone: "Europe/Paris",
				TimezoneOffset: 60,
			},
			Duration: 30 * 60 * 1000, // ends at 00:45:00
			Rate: 0,
			Guid: "id2-2",
			DeliveryType: "automated",
		},
	}
	assert.Equal(t, expectedFixedBasal, fixedBasals, "Unexpected fixed basal count returned")

}


func TestCutBasalWithLoopModes(t *testing.T) {
	startLoop1 := time.Date(2023, time.March ,12, 0, 0, 0, 0, time.UTC)
	endLoop1 := time.Date(2023, time.March ,12, 0, 47, 0, 0, time.UTC)
	startLoop2 := time.Date(2023, time.March ,12, 0, 52, 0, 0, time.UTC)
	endLoop2 := time.Date(2023, time.March ,12, 12, 0, 0, 0, time.UTC)
	loopModes := []schemaV1.LoopModeEvent{
		schemaV1.NewLoopModeEvent(startLoop1, &endLoop1, "automated"),
		schemaV1.NewLoopModeEvent(endLoop1, &startLoop2, "scheduled"),
		schemaV1.NewLoopModeEvent(startLoop2, &endLoop2, "automated"),
	}
	currentBasal := schema.BasalSample{
		Sample: schema.Sample{
			Timestamp: time.Date(2023, time.March, 12, 0, 45, 0, 0, time.UTC),
			Timezone: "Europe/Paris",
			TimezoneOffset: 60,
		},
		Duration: 10 * 60 * 1000,
		Rate: 0.5,
		Guid: "id1",
	}
	newEnd, newCount, cutBasals := cutBasalWithLoopModes(currentBasal, 1, loopModes)
	expectedEnd := time.Date(2023, time.March, 12, 0, 55, 0, 0, time.UTC)
	assert.Equal(t, expectedEnd, newEnd, "Unexpected cut basal count returned")
	expectedCutBasal := []schema.BasalSample{
		{
			Sample: schema.Sample{
				Timestamp: currentBasal.Timestamp,
				Timezone: "Europe/Paris",
				TimezoneOffset: 60,
			},
			Duration: 2 * 60 * 1000,
			Rate: currentBasal.Rate,
			Guid: "id1-1",
			DeliveryType: "automated",
		},
		{
			Sample: schema.Sample{
				Timestamp: endLoop1,
				Timezone: "Europe/Paris",
				TimezoneOffset: 60,
			},
			Duration: 5 * 60 * 1000,
			Rate: currentBasal.Rate,
			Guid: "id1-2",
			DeliveryType: "scheduled",
		},
		{
			Sample: schema.Sample{
				Timestamp: startLoop2,
				Timezone: "Europe/Paris",
				TimezoneOffset: 60,
			},
			Duration: 3 * 60 * 1000,
			Rate: currentBasal.Rate,
			Guid: "id1-3",
			DeliveryType: "automated",
		},
	}
	assert.Equal(t, len(expectedCutBasal) + 1, newCount, "Unexpected cut basal count returned")
	assert.Equal(t, expectedCutBasal, cutBasals, "Unexpected cut basal count returned")
}
