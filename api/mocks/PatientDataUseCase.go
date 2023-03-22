// Code generated by mockery v2.12.3. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	httpreswriter "github.com/tidepool-org/tide-whisperer/api/httpreswriter"

	schema "github.com/tidepool-org/tide-whisperer/schema"
)

// PatientDataUseCase is an autogenerated mock type for the PatientDataUseCase type
type PatientDataUseCase struct {
	mock.Mock
}

// GetData provides a mock function with given fields: ctx, res, readBasalBucket
func (_m *PatientDataUseCase) GetData(ctx context.Context, res *httpreswriter.HttpResponseWriter, readBasalBucket bool) error {
	ret := _m.Called(ctx, res, readBasalBucket)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *httpreswriter.HttpResponseWriter, bool) error); ok {
		r0 = rf(ctx, res, readBasalBucket)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetDataRangeV1 provides a mock function with given fields: ctx, traceID, userID
func (_m *PatientDataUseCase) GetDataRangeV1(ctx context.Context, traceID string, userID string) (*schema.Date, error) {
	ret := _m.Called(ctx, traceID, userID)

	var r0 *schema.Date
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *schema.Date); ok {
		r0 = rf(ctx, traceID, userID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*schema.Date)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, traceID, userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type NewPatientDataUseCaseT interface {
	mock.TestingT
	Cleanup(func())
}

// NewPatientDataUseCase creates a new instance of PatientDataUseCase. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewPatientDataUseCase(t NewPatientDataUseCaseT) *PatientDataUseCase {
	mock := &PatientDataUseCase{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
