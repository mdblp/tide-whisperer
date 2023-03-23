package schema

import (
	"time"
)

type DbSchedule struct {
	Rate  float64 `bson:"rate,omitempty"`
	Start int64   `bson:"start,omitempty"`
}

type DbProfile struct {
	Type          string       `bson:"type,omitempty"`
	Time          time.Time    `bson:"time,omitempty"`
	Timezone      string       `bson:"timezone,omitempty"`
	Guid          string       `bson:"guid,omitempty"`
	BasalSchedule []DbSchedule `bson:"basalSchedule,omitempty"`
}
