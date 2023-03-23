package api

import (
	"context"

	"github.com/tidepool-org/tide-whisperer/common"
)

type PatientDataUseCase interface {
	GetData(ctx context.Context, res *common.HttpResponseWriter, readBasalBucket bool) error
	GetDataRangeV1(ctx context.Context, traceID string, userID string) (*common.Date, error)
}
