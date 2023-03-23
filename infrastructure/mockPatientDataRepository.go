package infrastructure

import (
	"context"
	"fmt"

	goComMgo "github.com/tidepool-org/go-common/clients/mongo"
	"github.com/tidepool-org/tide-whisperer/common"
	"github.com/tidepool-org/tide-whisperer/schema"
	"go.mongodb.org/mongo-driver/bson"
)

// MockPatientDataRepository use for unit tests
type MockPatientDataRepository struct {
	DeviceModel string

	ParametersHistory    bson.M
	BasalSecurityProfile *schema.DbProfile

	DataRangeV1    []string
	DataV1         []string
	DataIDV1       []string
	DataBGV1       []string
	DataPSV1       *string
	LoopModeEvents []schema.LoopModeEvent
}

func NewMockPatientDataRepository() *MockPatientDataRepository {
	return &MockPatientDataRepository{
		DeviceModel: "test",
	}
}

// GetDataRangeLegacy mock func, return nil,nil
func (c *MockPatientDataRepository) GetDataRangeLegacy(ctx context.Context, traceID string, userID string) (*common.Date, error) {
	if c.DataRangeV1 != nil && len(c.DataRangeV1) == 2 {
		return &common.Date{
			Start: c.DataRangeV1[0],
			End:   c.DataRangeV1[1],
		}, nil
	}
	return nil, fmt.Errorf("{%s} - [%s] - No data", traceID, userID)
}

// GetDataV1 v1 api mock call to fetch diabetes data
func (c *MockPatientDataRepository) GetDataInDeviceData(ctx context.Context, traceID string, userID string, dates *common.Date, excludedType []string) (goComMgo.StorageIterator, error) {
	if c.DataV1 != nil {
		return &MockDbAdapterIterator{
			numIter: -1,
			maxIter: len(c.DataV1),
			data:    c.DataV1,
		}, nil
	}
	return nil, fmt.Errorf("{%s} - [%s] - No data", traceID, userID)
}

func (c *MockPatientDataRepository) GetLatestBasalSecurityProfile(ctx context.Context, traceID string, userID string) (*schema.DbProfile, error) {
	if c.BasalSecurityProfile != nil {
		return c.BasalSecurityProfile, nil
	}
	return nil, nil
}

// GetUploadDataV1 Fetch upload data from theirs upload ids, using the $in query parameter
func (c *MockPatientDataRepository) GetUploadData(ctx context.Context, traceID string, uploadIds []string) (goComMgo.StorageIterator, error) {
	if c.DataIDV1 != nil {
		return &MockDbAdapterIterator{
			numIter: -1,
			maxIter: len(c.DataIDV1),
			data:    c.DataIDV1,
		}, nil
	}
	return nil, fmt.Errorf("{%s} - No data", traceID)
}

func (c *MockPatientDataRepository) GetLoopMode(ctx context.Context, traceID string, userID string, dates *common.Date) ([]schema.LoopModeEvent, error) {
	if c.LoopModeEvents != nil {
		return c.LoopModeEvents, nil
	}
	return nil, fmt.Errorf("{%s} - No data", traceID)
}
