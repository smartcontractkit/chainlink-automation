// Code generated by mockery v2.28.1. DO NOT EDIT.

package mocks

import (
	types "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	mock "github.com/stretchr/testify/mock"
)

// MockRetryer is an autogenerated mock type for the Retryer type
type MockRetryer struct {
	mock.Mock
}

// Retry provides a mock function with given fields: _a0
func (_m *MockRetryer) Retry(_a0 types.CheckResult) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(types.CheckResult) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewMockRetryer interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockRetryer creates a new instance of MockRetryer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockRetryer(t mockConstructorTestingTNewMockRetryer) *MockRetryer {
	mock := &MockRetryer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
