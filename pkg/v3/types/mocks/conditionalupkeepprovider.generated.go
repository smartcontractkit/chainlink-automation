// Code generated by mockery v2.43.2. DO NOT EDIT.

package mocks

import (
	context "context"

	automation "github.com/smartcontractkit/chainlink-common/pkg/types/automation"

	mock "github.com/stretchr/testify/mock"
)

// MockConditionalUpkeepProvider is an autogenerated mock type for the ConditionalUpkeepProvider type
type MockConditionalUpkeepProvider struct {
	mock.Mock
}

// GetActiveUpkeeps provides a mock function with given fields: _a0
func (_m *MockConditionalUpkeepProvider) GetActiveUpkeeps(_a0 context.Context) ([]automation.UpkeepPayload, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for GetActiveUpkeeps")
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

// NewMockConditionalUpkeepProvider creates a new instance of MockConditionalUpkeepProvider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockConditionalUpkeepProvider(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockConditionalUpkeepProvider {
	mock := &MockConditionalUpkeepProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
