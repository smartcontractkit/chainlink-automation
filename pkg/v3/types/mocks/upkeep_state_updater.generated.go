// Code generated by mockery v2.28.1. DO NOT EDIT.

package mocks

import (
	context "context"

	types "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	mock "github.com/stretchr/testify/mock"
)

// MockUpkeepStateUpdater is an autogenerated mock type for the UpkeepStateUpdater type
type MockUpkeepStateUpdater struct {
	mock.Mock
}

// SetUpkeepState provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockUpkeepStateUpdater) SetUpkeepState(_a0 context.Context, _a1 types.CheckResult, _a2 types.UpkeepState) error {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, types.CheckResult, types.UpkeepState) error); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewMockUpkeepStateUpdater interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockUpkeepStateUpdater creates a new instance of MockUpkeepStateUpdater. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockUpkeepStateUpdater(t mockConstructorTestingTNewMockUpkeepStateUpdater) *MockUpkeepStateUpdater {
	mock := &MockUpkeepStateUpdater{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}