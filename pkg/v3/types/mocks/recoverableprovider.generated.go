// Code generated by mockery v2.28.1. DO NOT EDIT.

package mocks

import (
	types "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	mock "github.com/stretchr/testify/mock"
)

// MockRecoverableProvider is an autogenerated mock type for the RecoverableProvider type
type MockRecoverableProvider struct {
	mock.Mock
}

// GetRecoveryProposals provides a mock function with given fields:
func (_m *MockRecoverableProvider) GetRecoveryProposals() ([]types.UpkeepPayload, error) {
	ret := _m.Called()

	var r0 []types.UpkeepPayload
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]types.UpkeepPayload, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []types.UpkeepPayload); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.UpkeepPayload)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMockRecoverableProvider interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockRecoverableProvider creates a new instance of MockRecoverableProvider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockRecoverableProvider(t mockConstructorTestingTNewMockRecoverableProvider) *MockRecoverableProvider {
	mock := &MockRecoverableProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}