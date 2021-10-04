package schema

import "time"

type (
	CbgBucket struct {
		Id               string    `bson:"_id,omitempty"`
		CreatedTimestamp time.Time `bson:"createdTimestamp,omitempty"`
		UserId           string    `bson:"userId,omitempty"`
		Day              time.Time `bson:"day,omitempty"` // ie: 2021-09-28
		Measurements     []Sample  `bson:"measurements"`
	}

	Sample struct {
		Value            float64   `bson:"value,omitempty"`
		Units            string    `bson:"units,omitempty"`
		CreatedTimestamp time.Time `bson:"createdTimestamp,omitempty"`
		TimeStamp        time.Time `bson:"timestamp,omitempty"`
		Timezone         string    `bson:"timezone,omitempty"`
		TimezoneOffset   int       `bson:"timezoneOffset,omitempty"`
	}
)
