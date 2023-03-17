package schema

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLoopModeEvent(t *testing.T) {
	start := time.Date(2023, time.March ,14, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, time.March ,15, 0, 0, 0, 0, time.UTC)
	delivery := "test"
	loopMode := NewLoopModeEvent(start, &end, delivery)
	expected := LoopModeEvent{
		TimeRange: TimeRange{
			Start: start,
			End: &end,
		},
		DeliveryType: "test",
	}
	if !reflect.DeepEqual(loopMode, expected) {
		t.Fatalf("Expected loopMode:%v to equal %v", loopMode, expected)
	}
	loopMode = NewLoopModeEvent(start, nil, delivery)
	expected = LoopModeEvent{
		TimeRange: TimeRange{
			Start: start,
			End: nil,
		},
		DeliveryType: "test",
	}
	if !reflect.DeepEqual(loopMode, expected) {
		t.Fatalf("Expected loopMode:%v to equal %v", loopMode, expected)
	}
}

func TestFillLoopModeEvents_NotOverlapping(t *testing.T) {
	start1 := time.Date(2023, time.March ,14, 0, 0, 0, 0, time.UTC)
	end1 := time.Date(2023, time.March ,15, 0, 0, 0, 0, time.UTC)
	start2 := time.Date(2023, time.March ,16, 0, 0, 0, 0, time.UTC)
	end2 := time.Date(2023, time.March ,17, 0, 0, 0, 0, time.UTC)
	start3 := time.Date(2023, time.March ,19, 0, 0, 0, 0, time.UTC)
	delivery := ""
	loopModes := []LoopModeEvent{
		NewLoopModeEvent(start1, &end1, delivery),
		NewLoopModeEvent(start2, &end2, delivery),
		NewLoopModeEvent(start3, nil, delivery),
	}
	filledLoopModes := FillLoopModeEvents(loopModes)
	expected := []LoopModeEvent{
		NewLoopModeEvent(start1, &end1, "automated"),
		NewLoopModeEvent(end1, &start2, "scheduled"),
		NewLoopModeEvent(start2, &end2, "automated"),
		NewLoopModeEvent(end2, &start3, "scheduled"),
		NewLoopModeEvent(start3, nil, "automated"),
	}
	assert.Equal(t, expected, filledLoopModes, "Loop Modes are not filled as expected")
}


func TestFillLoopModeEvents_WithNilEnds(t *testing.T) {
	start1 := time.Date(2023, time.March ,14, 0, 0, 0, 0, time.UTC)
	end1 := time.Date(2023, time.March ,15, 0, 0, 0, 0, time.UTC)
	start2 := time.Date(2023, time.March ,16, 0, 0, 0, 0, time.UTC)
	end2 := time.Date(2023, time.March ,17, 0, 0, 0, 0, time.UTC)
	start3 := time.Date(2023, time.March ,19, 0, 0, 0, 0, time.UTC)
	delivery := ""
	loopModes := []LoopModeEvent{
		NewLoopModeEvent(start1, &end1, delivery),
		NewLoopModeEvent(start2, nil, delivery),
		NewLoopModeEvent(start2, &end2, delivery),
		NewLoopModeEvent(start3, nil, delivery),
		NewLoopModeEvent(start3, nil, delivery),
	}
	filledLoopModes := FillLoopModeEvents(loopModes)
	expected := []LoopModeEvent{
		NewLoopModeEvent(start1, &end1, "automated"),
		NewLoopModeEvent(end1, &start2, "scheduled"),
		NewLoopModeEvent(start2, &end2, "automated"),
		NewLoopModeEvent(end2, &start3, "scheduled"),
		NewLoopModeEvent(start3, nil, "automated"),
	}
	assert.Equal(t, expected, filledLoopModes, "Loop Modes are not filled as expected")
}

func TestFillLoopModeEvents_Overlapping(t *testing.T) {
	start1 := time.Date(2023, time.March ,14, 0, 0, 0, 0, time.UTC)
	end1 := time.Date(2023, time.March ,15, 0, 0, 0, 0, time.UTC)
	start2 := time.Date(2023, time.March ,14, 10, 0, 0, 0, time.UTC)
	end2 := time.Date(2023, time.March ,17, 0, 0, 0, 0, time.UTC)
	start3 := time.Date(2023, time.March ,16, 0, 0, 0, 0, time.UTC)
	delivery := ""
	loopModes := []LoopModeEvent{
		NewLoopModeEvent(start1, &end1, delivery),
		NewLoopModeEvent(start2, &end2, delivery),
		NewLoopModeEvent(start3, nil, delivery),
	}
	filledLoopModes := FillLoopModeEvents(loopModes)
	expected := []LoopModeEvent{
		NewLoopModeEvent(start1, &start2, "automated"),
		NewLoopModeEvent(start2, &end1, "automated"),
		NewLoopModeEvent(end1, &start3, "automated"),
		NewLoopModeEvent(start3, &end2, "automated"),
		NewLoopModeEvent(end2, nil, "automated"),
	}
	assert.Equal(t, expected, filledLoopModes, "Loop Modes are not filled as expected")
}


func TestGetLoopModeEventsBetween_Overlapping(t *testing.T) {
	start1 := time.Date(2023, time.March ,14, 0, 0, 0, 0, time.UTC)
	end1 := time.Date(2023, time.March ,15, 0, 0, 0, 0, time.UTC)
	start2 := time.Date(2023, time.March ,15, 10, 0, 0, 0, time.UTC)
	end2 := time.Date(2023, time.March ,15, 18, 0, 0, 0, time.UTC)
	start3 := time.Date(2023, time.March ,16, 0, 0, 0, 0, time.UTC)

	loopModes := []LoopModeEvent{
		NewLoopModeEvent(start1, &end1, "automated"),
		NewLoopModeEvent(end1, &start2, "scheduled"),
		NewLoopModeEvent(start2, &end2, "automated"),
		NewLoopModeEvent(end2, &start3, "scheduled"),
		NewLoopModeEvent(start3, nil, "automated"),
	}
	testStartWithin := time.Date(2023, time.March ,14, 10, 0, 0, 0, time.UTC)
	testEndWithin := time.Date(2023, time.March ,14, 20, 0, 0, 0, time.UTC)
	events := GetLoopModeEventsBetween(testStartWithin, &testEndWithin, loopModes)
	expected := []LoopModeEvent{
		NewLoopModeEvent(testStartWithin, &testEndWithin, "automated"),
	}
	assert.Equal(t, expected, events, "Loop Modes are not retrieved as expected")

	testEndAcrossRanges := time.Date(2023, time.March ,16, 10, 0, 0, 0, time.UTC)
	events = GetLoopModeEventsBetween(testStartWithin, &testEndAcrossRanges, loopModes)
	expected = []LoopModeEvent{
		NewLoopModeEvent(testStartWithin, &end1, "automated"),
		NewLoopModeEvent(end1, &start2, "scheduled"),
		NewLoopModeEvent(start2, &end2, "automated"),
		NewLoopModeEvent(end2, &start3, "scheduled"),
		NewLoopModeEvent(start3, &testEndAcrossRanges, "automated"),
	}
	assert.Equal(t, expected, events, "Loop Modes are not retrieved as expected")
}

func TestGetLoopModeEventsBetween_NonOverlapping(t *testing.T) {
	start1 := time.Date(2023, time.March ,14, 0, 0, 0, 0, time.UTC)
	end1 := time.Date(2023, time.March ,15, 0, 0, 0, 0, time.UTC)
	start2 := time.Date(2023, time.March ,14, 10, 0, 0, 0, time.UTC)
	end2 := time.Date(2023, time.March ,17, 0, 0, 0, 0, time.UTC)
	start3 := time.Date(2023, time.March ,16, 0, 0, 0, 0, time.UTC)

	loopModes := []LoopModeEvent{
		NewLoopModeEvent(start1, &end1, "automated"),
		NewLoopModeEvent(end1, &start2, "scheduled"),
		NewLoopModeEvent(start2, &end2, "automated"),
		NewLoopModeEvent(end2, &start3, "scheduled"),
		NewLoopModeEvent(start3, nil, "automated"),
	}
	testStartBeforeAll := time.Date(2023, time.March ,12, 0, 0, 0, 0, time.UTC)
	testEndBeforeAll := time.Date(2023, time.March ,12, 20, 0, 0, 0, time.UTC)
	events := GetLoopModeEventsBetween(testStartBeforeAll, &testEndBeforeAll, loopModes)
	expected := []LoopModeEvent{
		NewLoopModeEvent(testStartBeforeAll, &testEndBeforeAll, "scheduled"),
	}
	assert.Equal(t, expected, events, "Loop Modes are not retrieved as expected")

}