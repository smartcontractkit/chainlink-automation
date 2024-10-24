// Code generated by mockery v2.43.2. DO NOT EDIT.

package mocks

import (
	context "context"

	automation "github.com/smartcontractkit/chainlink-common/pkg/types/automation"

	mock "github.com/stretchr/testify/mock"
)

// MockLogEventProvider is an autogenerated mock type for the LogEventProvider type
type MockLogEventProvider struct {
	mock.Mock
}

// Close provides a mock function with given fields:
func (_m *MockLogEventProvider) Close() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Close")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetLatestPayloads provides a mock function with given fields: _a0
func (_m *MockLogEventProvider) GetLatestPayloads(_a0 context.Context) ([]automation.UpkeepPayload, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for GetLatestPayloads")
	}

	var r0 []automation.UpkeepPayload
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) ([]automation.UpkeepPayload, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(context.Context) []automation.UpkeepPayload); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]automation.UpkeepPayload)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetConfig provides a mock function with given fields: _a0
func (_m *MockLogEventProvider) SetConfig(_a0 automation.LogEventProviderConfig) {
	_m.Called(_a0)
}

// Start provides a mock function with given fields: _a0
func (_m *MockLogEventProvider) Start(_a0 context.Context) error {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for Start")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMockLogEventProvider creates a new instance of MockLogEventProvider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockLogEventProvider(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockLogEventProvider {
	mock := &MockLogEventProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
