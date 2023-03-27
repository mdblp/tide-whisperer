package api

import (
	"bytes"
	"context"

	"github.com/tidepool-org/tide-whisperer/common"
)

type PatientDataUseCase interface {
	GetData(ctx context.Context, userID string, traceID string, startDate string, endDate string, withPumpSettings bool, readBasalBucket bool, buff *bytes.Buffer, res *common.HttpResponseWriter) error
	GetDataRangeLegacy(ctx context.Context, traceID string, userID string) (*common.Date, error)
}
