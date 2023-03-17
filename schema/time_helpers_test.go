package schema

import (
	"testing"
	"time"
)

func Test_TimeBeforeOrEqual(t *testing.T) {
	time1 := time.Date(2023, time.March ,14, 0, 0, 0, 0, time.UTC)
	time2 := time.Date(2023, time.March, 15, 0, 0, 0, 0, time.UTC)
	test := TimeBeforeOrEqual(&time1, &time1)
	if !test {
		t.Fatalf("Expected t1:%v to be before or equal t2:%v", time1, time1)
	}
	test = TimeBeforeOrEqual(&time1, &time2)
	if !test {
		t.Fatalf("Expected t1:%v to be before or equal t2:%v", time1, time2)
	}
	test = TimeBeforeOrEqual(&time2, &time1)
	if test {
		t.Fatalf("Expected t1:%v not to be before or equal t2:%v", time2, time1)
	}
}

func Test_TimeAfterOrEqual(t *testing.T) {
	time1 := time.Date(2023, time.March ,14, 0, 0, 0, 0, time.UTC)
	time2 := time.Date(2023, time.March, 15, 0, 0, 0, 0, time.UTC)
	test := TimeAfterOrEqual(&time1, &time1)
	if !test {
		t.Fatalf("Expected t1:%v to be after or equal t2:%v", time1, time1)
	}
	test = TimeAfterOrEqual(&time2, &time1)
	if !test {
		t.Fatalf("Expected t1:%v to be after or equal t2:%v", time1, time2)
	}
	test = TimeAfterOrEqual(&time1, &time2)
	if test {
		t.Fatalf("Expected t1:%v not to be after or equal t2:%v", time1, time2)
	}
}

func Test_TimeBetween(t *testing.T) {
	rangeStart := time.Date(2023, time.March ,14, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2023, time.March, 15, 0, 0, 0, 0, time.UTC)
	inRangeDate := time.Date(2023, time.March, 14, 12, 0, 0, 0, time.UTC)
	outOfRangeDate := time.Date(2023, time.March, 15, 12, 0, 0, 0, time.UTC)
	test := TimeBetween(inRangeDate, &rangeStart, &rangeEnd)
	if !test {
		t.Fatalf("Expected t1:%v to be between ]%v,%v]", inRangeDate, rangeStart, rangeEnd)
	}
	test = TimeBetween(rangeStart, &rangeStart, &rangeEnd)
	if test {
		t.Fatalf("Expected t1:%v not to be between ]%v,%v]", rangeStart, rangeStart, rangeEnd)
	}
	test = TimeBetween(rangeEnd, &rangeStart, &rangeEnd)
	if !test {
		t.Fatalf("Expected t1:%v to be between ]%v,%v]", rangeEnd, rangeStart, rangeEnd)
	}
	test = TimeBetween(outOfRangeDate, &rangeEnd, &rangeStart)
	if test {
		t.Fatalf("Expected t1:%v not to be between ]%v,%v]",outOfRangeDate, rangeStart, rangeEnd)
	}
}