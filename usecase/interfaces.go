package usecase

import (
	"context"

	goComMgo "github.com/tidepool-org/go-common/clients/mongo"
	"github.com/tidepool-org/tide-whisperer/common"
	"github.com/tidepool-org/tide-whisperer/schema"
)

type PatientDataRepository interface {
	GetDataRangeV1(ctx context.Context, traceID string, userID string) (*common.Date, error)
	GetDataV1(ctx context.Context, traceID string, userID string, dates *common.Date, excludeTypes []string) (goComMgo.StorageIterator, error)
	GetLatestBasalSecurityProfile(ctx context.Context, traceID string, userID string) (*schema.DbProfile, error)
	GetUploadDataV1(ctx context.Context, traceID string, uploadIds []string) (goComMgo.StorageIterator, error)
	GetCbgForSummaryV1(ctx context.Context, traceID string, userID string, startDate string) (goComMgo.StorageIterator, error)
	GetLoopMode(ctx context.Context, traceID string, userID string, dates *common.Date) ([]schema.LoopModeEvent, error)
}

type DatabaseAdapter interface {
	goComMgo.Storage
}
