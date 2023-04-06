package usecase

import (
	"bytes"
	"context"

	goComMgo "github.com/tidepool-org/go-common/clients/mongo"
	"github.com/tidepool-org/tide-whisperer/common"
	"github.com/tidepool-org/tide-whisperer/schema"
)

type PatientDataRepository interface {
	GetDataRangeLegacy(ctx context.Context, traceID string, userID string) (*common.Date, error)
	GetDataInDeviceData(ctx context.Context, traceID string, userID string, dates *common.Date, excludeTypes []string) (goComMgo.StorageIterator, error)
	GetLatestBasalSecurityProfile(ctx context.Context, traceID string, userID string) (*schema.DbProfile, error)
	GetUploadData(ctx context.Context, traceID string, uploadIds []string) (goComMgo.StorageIterator, error)
	GetLoopMode(ctx context.Context, traceID string, userID string, dates *common.Date) ([]schema.LoopModeEvent, error)
}

type DatabaseAdapter interface {
	goComMgo.Storage
}

type PatientDataUseCase interface {
	GetData(args GetDataArgs) (*bytes.Buffer, *common.DetailedError)
}
type Uploader interface {
	Upload(ctx context.Context, filename string, buffer *bytes.Buffer) error
}
