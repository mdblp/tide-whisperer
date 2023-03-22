package schema

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStartEqual(t *testing.T) {
	start1 := time.Date(2023, time.March ,14, 0, 0, 0, 0, time.UTC)
	end1 := time.Date(2023, time.March ,15, 0, 0, 0, 0, time.UTC)
	start2 := time.Date(2023, time.March ,16, 0, 0, 0, 0, time.UTC)
	end2 := time.Date(2023, time.March ,17, 0, 0, 0, 0, time.UTC)

	range1 := TimeRange{
		Start: start1,
		End: &end1,
	}
	range2 := TimeRange{
		Start: start1,
		End: &end2,
	}
	assert.Equal(t, range1.StartEqual(range2), true, "Start equality was expected")
	assert.Equal(t, range2.StartEqual(range1), true, "Start equality was expected")
	
	range2.Start = start2
	assert.Equal(t, range1.StartEqual(range2), false, "Start equality wasn't expected")
	assert.Equal(t, range2.StartEqual(range1), false, "Start equality wasn't expected")
}

func TestEndEqual(t *testing.T) {
	start1 := time.Date(2023, time.March ,14, 0, 0, 0, 0, time.UTC)
	end1 := time.Date(2023, time.March ,15, 0, 0, 0, 0, time.UTC)
	start2 := time.Date(2023, time.March ,16, 0, 0, 0, 0, time.UTC)
	end2 := time.Date(2023, time.March ,17, 0, 0, 0, 0, time.UTC)

	range1 := TimeRange{
		Start: start1,
		End: &end1,
	}
	range2 := TimeRange{
		Start: start2,
		End: &end1,
	}
	assert.Equal(t, range1.EndEqual(range2), true, "End equality was expected")
	assert.Equal(t, range2.EndEqual(range1), true, "End equality was expected")
	
	range2.End = &end2
	assert.Equal(t, range1.EndEqual(range2), false, "End equality wasn't expected")
	assert.Equal(t, range2.EndEqual(range1), false, "End equality wasn't expected")
	
	range2.End = nil
	range1.End = nil
	assert.Equal(t, range1.EndEqual(range2), true, "End equality was expected")
	assert.Equal(t, range2.EndEqual(range1), true, "End equality was expected")

	range2.End = &end1
	assert.Equal(t, range1.EndEqual(range2), false, "End equality wasn't expected")
	assert.Equal(t, range2.EndEqual(range1), false, "End equality wasn't expected")
}

func TestEqual(t *testing.T) {
	start1 := time.Date(2023, time.March ,14, 0, 0, 0, 0, time.UTC)
	end1 := time.Date(2023, time.March ,15, 0, 0, 0, 0, time.UTC)
	start2 := time.Date(2023, time.March ,16, 0, 0, 0, 0, time.UTC)
	end2 := time.Date(2023, time.March ,17, 0, 0, 0, 0, time.UTC)

	range1 := TimeRange{
		Start: start1,
		End: &end1,
	}
	range2 := TimeRange{
		Start: start1,
		End: &end1,
	}
	assert.Equal(t, range1.Equal(range2), true, "Equality was expected")
	assert.Equal(t, range2.Equal(range1), true, "Equality was expected")
	
	inequalTests := TimeRanges{
		NewTimeRange(start1, &end2),
		NewTimeRange(start2, &end1),
		NewTimeRange(start1, nil),
		NewTimeRange(start2, nil),
	}
	for _, rangeToTest := range inequalTests {
		assert.Equal(t, range1.Equal(rangeToTest), false, "Equality wasn't expected")
		assert.Equal(t, rangeToTest.Equal(range1), false, "Equality wasn't expected")
	}
}

func TestOverLaps(t *testing.T) {
	start1 := time.Date(2023, time.March ,14, 0, 0, 0, 0, time.UTC)
	end1 := time.Date(2023, time.March ,15, 0, 0, 0, 0, time.UTC)
	start2 := time.Date(2023, time.March ,14, 12, 0, 0, 0, time.UTC)
	end2 := time.Date(2023, time.March ,14, 18, 0, 0, 0, time.UTC)
	start3 := time.Date(2023, time.March ,13, 0, 0, 0, 0, time.UTC)
	range1 := TimeRange{
		Start: start1,
		End: &end1,
	}

	overLappingTests := TimeRanges{
		NewTimeRange(start1, &end1),
		NewTimeRange(start1, &end2),
		NewTimeRange(start1, nil),
		NewTimeRange(start2, &end2),
		NewTimeRange(start2, nil),
		NewTimeRange(start3, &end2),
	}
	for _, rangeToTest := range overLappingTests {
		assert.Equal(t, range1.OverLaps(rangeToTest), true, "Overlap was expected")
		assert.Equal(t, rangeToTest.OverLaps(range1), true, "Overlap was expected")
	}
	notOverlappingTests := []TimeRange{
		NewTimeRange(end1, &start2),
		NewTimeRange(start3, &start1),
		NewTimeRange(end1, nil),
	}
	for _, rangeToTest := range notOverlappingTests {
		assert.Equal(t, range1.OverLaps(rangeToTest), false, "Overlap wasn't expected")
		assert.Equal(t, rangeToTest.OverLaps(range1), false, "Overlap wasn't expected")
	}
}

type testCaseUnion struct {
	Name                string
	InputRange          TimeRange
	Union				TimeRange
	ExpectedUnion       TimeRanges
	ExpectedFilledUnion TimeRanges
}

func TestUnion_Overlapping(t *testing.T) {
	start1 := time.Date(2023, time.March ,14, 0, 0, 0, 0, time.UTC)
	end1 := time.Date(2023, time.March ,15, 0, 0, 0, 0, time.UTC)
	start2 := time.Date(2023, time.March ,14, 12, 0, 0, 0, time.UTC)
	end2 := time.Date(2023, time.March ,14, 18, 0, 0, 0, time.UTC)
	start3 := time.Date(2023, time.March ,13, 0, 0, 0, 0, time.UTC)
	inputRange := TimeRange{
		Start: start1,
		End: &end1,
	}
	inputRangeWithNilEnd := TimeRange{
		Start: start1,
		End: nil,
	}

	overLappingTests := []testCaseUnion{
		{
			Name: "Self Union",
			InputRange: inputRange,
			Union: NewTimeRange(start1, &end1),
			ExpectedUnion: TimeRanges{NewTimeRange(start1, &end1)},
			ExpectedFilledUnion: TimeRanges{NewTimeRange(start1, &end1)}},
		{
			Name: "Same start",
			InputRange: inputRange,
			Union: NewTimeRange(start1, &end2),
			ExpectedUnion: TimeRanges{NewTimeRange(start1, &end2), NewTimeRange(end2, &end1)},
			ExpectedFilledUnion: TimeRanges{NewTimeRange(start1, &end2), NewTimeRange(end2, &end1)}},
		{
			Name: "Same start, nil end",
			InputRange: inputRange,
			Union: NewTimeRange(start1, nil),
			ExpectedUnion: TimeRanges{NewTimeRange(start1, &end1), NewTimeRange(end1, nil)},
			ExpectedFilledUnion: TimeRanges{NewTimeRange(start1, &end1), NewTimeRange(end1, nil)}},
		{
			Name: "Same end not nil",
			InputRange: inputRange,
			Union: NewTimeRange(start2, &end1),
			ExpectedUnion: TimeRanges{NewTimeRange(start1, &start2), NewTimeRange(start2, &end1)},
			ExpectedFilledUnion: TimeRanges{NewTimeRange(start1, &start2), NewTimeRange(start2, &end1)}},
		{
			Name: "Same end nil",
			InputRange: inputRangeWithNilEnd,
			Union: NewTimeRange(start2, nil),
			ExpectedUnion: TimeRanges{NewTimeRange(start1, &start2), NewTimeRange(start2, nil)},
			ExpectedFilledUnion: TimeRanges{NewTimeRange(start1, &start2), NewTimeRange(start2, nil)}},
		{
			Name: "Range totally included",
			InputRange: inputRange,
			Union: NewTimeRange(start2, &end2),
			ExpectedUnion: TimeRanges{NewTimeRange(start1, &start2), NewTimeRange(start2, &end2), NewTimeRange(end2, &end1)},
			ExpectedFilledUnion: TimeRanges{NewTimeRange(start1, &start2), NewTimeRange(start2, &end2), NewTimeRange(end2, &end1)}},
		{
			Name: "Start included, nil end",
			InputRange: inputRange,
			Union: NewTimeRange(start2, nil),
			ExpectedUnion: TimeRanges{NewTimeRange(start1, &start2), NewTimeRange(start2, &end1), NewTimeRange(end1, nil)},
			ExpectedFilledUnion: TimeRanges{NewTimeRange(start1, &start2), NewTimeRange(start2, &end1), NewTimeRange(end1, nil)}},
		{
			Name: "Start before, end included",
			InputRange: inputRange,
			Union: NewTimeRange(start3, &end2),
			ExpectedUnion: TimeRanges{NewTimeRange(start3, &start1), NewTimeRange(start1, &end2), NewTimeRange(end2, &end1)},
			ExpectedFilledUnion: TimeRanges{NewTimeRange(start3, &start1), NewTimeRange(start1, &end2), NewTimeRange(end2, &end1)}},
	}
	
	for _, tc := range overLappingTests {
		unionRange := tc.InputRange.Union(tc.Union, false)
		assert.Equal(t, unionRange, tc.ExpectedUnion, "Union is not as expected")
		unionRange = tc.Union.Union(tc.InputRange, false)
		assert.Equal(t, unionRange, tc.ExpectedUnion, "Union is not as expected")

		unionRange = tc.InputRange.Union(tc.Union, true)
		assert.Equal(t, unionRange, tc.ExpectedFilledUnion, "Union filled is not as expected")
		unionRange = tc.Union.Union(tc.InputRange, true)
		assert.Equal(t, unionRange, tc.ExpectedFilledUnion, "Union filled is not as expected")
	}
}

func TestUnion_NonOverlapping(t *testing.T) {
	start1 := time.Date(2023, time.March ,14, 0, 0, 0, 0, time.UTC)
	end1 := time.Date(2023, time.March ,15, 0, 0, 0, 0, time.UTC)
	start2 := time.Date(2023, time.March ,15, 12, 0, 0, 0, time.UTC)
	end2 := time.Date(2023, time.March ,15, 18, 0, 0, 0, time.UTC)
	range1 := TimeRange{
		Start: start1,
		End: &end1,
	}

	filledRange :=  NewTimeRange(end1, &start2)
	filledRange.Fill = true
	nonOverLappingTests := []testCaseUnion{
		{
			Name: "Non overlapping",
			InputRange: range1,
			Union: NewTimeRange(start2, &end2),
			ExpectedUnion: TimeRanges{NewTimeRange(start1, &end1), NewTimeRange(start2, &end2)},
			ExpectedFilledUnion: TimeRanges{NewTimeRange(start1, &end1), filledRange, NewTimeRange(start2, &end2)}},
		{
			Name: "non overlapping with nil end",
			InputRange: range1,
			Union: NewTimeRange(start2, nil),
			ExpectedUnion: TimeRanges{NewTimeRange(start1, &end1), NewTimeRange(start2, nil)},
			ExpectedFilledUnion: TimeRanges{NewTimeRange(start1, &end1), filledRange, NewTimeRange(start2, nil)}},
	}

	for _, tc := range nonOverLappingTests {
		unionRange := tc.InputRange.Union(tc.Union, false)
		assert.Equal(t, unionRange, tc.ExpectedUnion, "Union is not as expected")
		unionRange = tc.Union.Union(tc.InputRange, false)
		assert.Equal(t, unionRange, tc.ExpectedUnion, "Union is not as expected")

		unionRange = tc.InputRange.Union(tc.Union, true)
		assert.Equal(t, unionRange, tc.ExpectedFilledUnion, "Union filled is not as expected")
		unionRange = tc.Union.Union(tc.InputRange, true)
		assert.Equal(t, unionRange, tc.ExpectedFilledUnion, "Union filled is not as expected")
	}
}