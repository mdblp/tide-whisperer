package api

import (
	"bytes"
	"context"

	"github.com/tidepool-org/tide-whisperer/common"
)

type PatientDataUseCase interface {
	GetData(ctx context.Context, userID string, traceID string, startDate string, endDate string, withPumpSettings bool, sessionToken string, buff *bytes.Buffer) *common.DetailedError
	GetDataRangeLegacy(ctx context.Context, traceID string, userID string) (*common.Date, error)
}

type ExporterUseCase interface {
	Export(userID string, traceID string, startDate string, endDate string, withPumpSettings bool, sessionToken string)
}
