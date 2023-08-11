// Code generated by mockery v2.28.1. DO NOT EDIT.

package mocks

import (
	types "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	mock "github.com/stretchr/testify/mock"
)

// MockCoordinator is an autogenerated mock type for the Coordinator type
type MockCoordinator struct {
	mock.Mock
}

// FilterProposals provides a mock function with given fields: _a0
func (_m *MockCoordinator) FilterProposals(_a0 []types.CoordinatedProposal) ([]types.CoordinatedProposal, error) {
	ret := _m.Called(_a0)

	var r0 []types.CoordinatedProposal
	var r1 error
	if rf, ok := ret.Get(0).(func([]types.CoordinatedProposal) ([]types.CoordinatedProposal, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func([]types.CoordinatedProposal) []types.CoordinatedProposal); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.CoordinatedProposal)
		}
	}

	if rf, ok := ret.Get(1).(func([]types.CoordinatedProposal) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FilterResults provides a mock function with given fields: _a0
func (_m *MockCoordinator) FilterResults(_a0 []types.CheckResult) ([]types.CheckResult, error) {
	ret := _m.Called(_a0)

	var r0 []types.CheckResult
	var r1 error
	if rf, ok := ret.Get(0).(func([]types.CheckResult) ([]types.CheckResult, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func([]types.CheckResult) []types.CheckResult); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.CheckResult)
		}
	}

	if rf, ok := ret.Get(1).(func([]types.CheckResult) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ShouldAccept provides a mock function with given fields: _a0
func (_m *MockCoordinator) ShouldAccept(_a0 types.ReportedUpkeep) bool {
	ret := _m.Called(_a0)

	var r0 bool
	if rf, ok := ret.Get(0).(func(types.ReportedUpkeep) bool); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// ShouldTransmit provides a mock function with given fields: _a0
func (_m *MockCoordinator) ShouldTransmit(_a0 types.ReportedUpkeep) bool {
	ret := _m.Called(_a0)

	var r0 bool
	if rf, ok := ret.Get(0).(func(types.ReportedUpkeep) bool); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

type mockConstructorTestingTNewMockCoordinator interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockCoordinator creates a new instance of MockCoordinator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockCoordinator(t mockConstructorTestingTNewMockCoordinator) *MockCoordinator {
	mock := &MockCoordinator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}