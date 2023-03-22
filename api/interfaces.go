package api

import (
	"context"

	"github.com/tidepool-org/tide-whisperer/api/httpreswriter"
	"github.com/tidepool-org/tide-whisperer/schema"
)

type PatientDataUseCase interface {
	GetData(ctx context.Context, res *httpreswriter.HttpResponseWriter, readBasalBucket bool) error
	GetDataRangeV1(ctx context.Context, traceID string, userID string) (*schema.Date, error)
}
