package schema

import "time"

type TimeRange struct{
	Start time.Time `bson:"startTime"`
	End *time.Time  `bson:"endTime"`
	Fill bool
}
type TimeRanges []TimeRange

func NewTimeRange(start time.Time, end *time.Time) TimeRange {
	return TimeRange{
		Start: start,
		End: end,
		Fill: false,
	}
}

func (d TimeRange) StartEqual(d2 TimeRange) bool {
	return d.Start.Equal(d2.Start) 
}

func (d TimeRange) EndEqual(d2 TimeRange) bool {
	if d.End != nil && d2.End != nil {
		return d.End.Equal(*d2.End) 
	}
	return d.End == nil && d2.End == nil
}

func (d TimeRange) Equal(d2 TimeRange) bool {
	return d.StartEqual(d2) && d.EndEqual(d2)
}

func (d TimeRange) OverLaps(d2 TimeRange) bool {
	if d.StartEqual(d2) || d.EndEqual(d2) {
		return true
	}
	firstRange := d
	secondRange := d2
	if TimeBeforeOrEqual(&d2.Start, &d.Start) {
		firstRange = d2
		secondRange = d
	}
	separatedRanges := TimeAfterOrEqual(&secondRange.Start, firstRange.End)
	return !separatedRanges
}

func (d TimeRange) Union(d2 TimeRange, fill bool) TimeRanges {
	if !d.OverLaps(d2) {
		firstRange := d
		secondRange := d2
		if TimeBeforeOrEqual(&d2.Start, &d.Start) {
			firstRange = d2
			secondRange = d
		}
		if fill {
			filler := NewTimeRange(*firstRange.End, &secondRange.Start)
			filler.Fill = true
			return TimeRanges{firstRange, filler, secondRange}
		}
		return TimeRanges{firstRange, secondRange}
	}
	if d.Equal(d2) {
		return TimeRanges{d}
	}
	if d.StartEqual(d2) {
		minEnd := TimeMin(d.End, d2.End)
		maxEnd := TimeMax(d.End, d2.End)
		range1 := NewTimeRange(d.Start, minEnd)
		range2 := NewTimeRange(*minEnd, maxEnd)
		return TimeRanges{range1, range2}
	}
	if d.EndEqual(d2) {
		minStart := TimeMin(&d.Start, &d2.Start)
		maxStart := TimeMax(&d.Start, &d2.Start)
		range1 := NewTimeRange(*minStart, maxStart)
		range2 := NewTimeRange(*maxStart, d.End)
		return TimeRanges{range1, range2}
	}
	
	minStart := TimeMin(&d.Start, &d2.Start)
	maxStart := TimeMax(&d.Start, &d2.Start)
	minEnd := TimeMin(d.End, d2.End)
	maxEnd := TimeMax(d.End, d2.End)
	range1 := NewTimeRange(*minStart, TimeMin(maxStart, minEnd))
	range2 := NewTimeRange(*TimeMin(maxStart, minEnd), TimeMax(maxStart, minEnd))
	range3 := NewTimeRange(*TimeMax(maxStart, minEnd), maxEnd)
	return TimeRanges{range1, range2, range3}
}
