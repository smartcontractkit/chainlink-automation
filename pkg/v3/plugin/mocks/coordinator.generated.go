// Code generated by mockery v2.22.1. DO NOT EDIT.

package mocks

import (
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"

	"github.com/stretchr/testify/mock"
)

// MockCoordinator is an autogenerated mock type for the Coordinator type
type MockCoordinator struct {
	mock.Mock
}

// Accept provides a mock function with given fields: _a0
func (_m *MockCoordinator) Accept(_a0 ocr2keepers.ReportedUpkeep) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(ocr2keepers.ReportedUpkeep) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// IsTransmissionConfirmed provides a mock function with given fields: _a0
func (_m *MockCoordinator) IsTransmissionConfirmed(_a0 ocr2keepers.ReportedUpkeep) bool {
	ret := _m.Called(_a0)

	var r0 bool
	if rf, ok := ret.Get(0).(func(ocr2keepers.ReportedUpkeep) bool); ok {
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
