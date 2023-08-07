// Code generated by mockery v2.22.1. DO NOT EDIT.

package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

// MockUpkeepProvider is an autogenerated mock type for the UpkeepProvider type
type MockUpkeepProvider struct {
	mock.Mock
}

// GetActiveUpkeeps provides a mock function with given fields: _a0, _a1
func (_m *MockUpkeepProvider) GetActiveUpkeeps(_a0 context.Context, _a1 ocr2keepers.BlockKey) ([]ocr2keepers.UpkeepPayload, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []ocr2keepers.UpkeepPayload
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, ocr2keepers.BlockKey) ([]ocr2keepers.UpkeepPayload, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, ocr2keepers.BlockKey) []ocr2keepers.UpkeepPayload); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]ocr2keepers.UpkeepPayload)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, ocr2keepers.BlockKey) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMockUpkeepProvider interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockUpkeepProvider creates a new instance of MockUpkeepProvider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockUpkeepProvider(t mockConstructorTestingTNewMockUpkeepProvider) *MockUpkeepProvider {
	mock := &MockUpkeepProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
