package schema

import "time"

func TimeBeforeOrEqual(t *time.Time, t2 *time.Time) bool {
	if t == nil && t2 == nil {
		return true
	}
	if t == nil {
		return false
	}
	if t2 == nil {
		return true
	}
	return t.Before(*t2) || t.Equal(*t2)
}

func TimeAfterOrEqual(t *time.Time, t2 *time.Time) bool {
	if t == nil && t2 == nil {
		return true
	}
	if t == nil {
		return true
	}
	if t2 == nil {
		return false
	}
	return t.After(*t2) || t.Equal(*t2)
}

func TimeBetween(timeToTest time.Time, start *time.Time, end *time.Time) bool {
	if start == nil && end == nil {
		return true
	}
	if end == nil {
		return timeToTest.After(*start)
	}
	if start == nil {
		return TimeBeforeOrEqual(&timeToTest, end)
	}
	return timeToTest.After(*start) && TimeBeforeOrEqual(&timeToTest, end)
}

func TimeMin(time1 *time.Time, time2 *time.Time) *time.Time {
	if time1 == nil && time2 == nil {
		return nil
	}
	if time2 == nil {
		return time1
	}
	if time1 == nil {
		return time2
	}
	if time2.After(*time1) {
		return time1
	}
	return time2
}

func TimeMax(time1 *time.Time, time2 *time.Time) *time.Time {
	if time2 == nil || time1 == nil {
		return nil
	}
	if time2.After(*time1) {
		return time2
	}
	return time1
}
