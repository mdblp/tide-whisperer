// Code generated by mockery v2.12.3. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"
	api "github.com/tidepool-org/tide-whisperer/api"
)

// RequestLoggerFunc is an autogenerated mock type for the RequestLoggerFunc type
type RequestLoggerFunc struct {
	mock.Mock
}

// Execute provides a mock function with given fields: _a0
func (_m *RequestLoggerFunc) Execute(_a0 api.HandlerLoggerFunc) api.HandlerLoggerFunc {
	ret := _m.Called(_a0)

	var r0 api.HandlerLoggerFunc
	if rf, ok := ret.Get(0).(func(api.HandlerLoggerFunc) api.HandlerLoggerFunc); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.HandlerLoggerFunc)
		}
	}

	return r0
}

type NewRequestLoggerFuncT interface {
	mock.TestingT
	Cleanup(func())
}

// NewRequestLoggerFunc creates a new instance of RequestLoggerFunc. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewRequestLoggerFunc(t NewRequestLoggerFuncT) *RequestLoggerFunc {
	mock := &RequestLoggerFunc{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
