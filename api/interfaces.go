package api

import (
	"bytes"
	"context"

	"github.com/tidepool-org/tide-whisperer/common"
	"github.com/tidepool-org/tide-whisperer/usecase"
)

type PatientDataUseCase interface {
	GetData(args usecase.GetDataArgs) (*bytes.Buffer, *common.DetailedError)
	GetDataRangeLegacy(ctx context.Context, traceID string, userID string) (*common.Date, error)
}

type ExporterUseCase interface {
	Export(args usecase.ExportArgs)
}
