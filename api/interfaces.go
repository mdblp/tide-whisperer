package api

import (
	"bytes"
	"context"

	"github.com/tidepool-org/tide-whisperer/common"
)

type PatientDataUseCase interface {
	GetData(ctx context.Context, userID string, traceID string, startDate string, endDate string, withPumpSettings bool, readBasalBucket bool, sessionToken string, buff *bytes.Buffer) *common.DetailedError
	GetDataRangeLegacy(ctx context.Context, traceID string, userID string) (*common.Date, error)
}

type UploaderUseCase interface {
	Upload(ctx context.Context, filename string, buffer *bytes.Buffer) error
}
