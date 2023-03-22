package schema

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// SchemaVersion struct
type SchemaVersion struct {
	Minimum int
	Maximum int
}

// Params struct
type Params struct {
	UserID   string
	Types    []string
	SubTypes []string
	Date
	*SchemaVersion
	Carelink           bool
	Dexcom             bool
	DexcomDataSource   bson.M
	DeviceID           string
	Latest             bool
	Medtronic          bool
	MedtronicDate      string
	MedtronicUploadIds []string
	UploadID           string
	LevelFilter        []int
}

// Date struct
type Date struct {
	Start string
	End   string
}

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
