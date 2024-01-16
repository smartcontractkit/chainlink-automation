// Code generated by mockery v2.40.1. DO NOT EDIT.

package mocks

import (
	types "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	mock "github.com/stretchr/testify/mock"
)

// MockEncoder is an autogenerated mock type for the Encoder type
type MockEncoder struct {
	mock.Mock
}

// Encode provides a mock function with given fields: _a0
func (_m *MockEncoder) Encode(_a0 ...types.CheckResult) ([]byte, error) {
	_va := make([]interface{}, len(_a0))
	for _i := range _a0 {
		_va[_i] = _a0[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Encode")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(...types.CheckResult) ([]byte, error)); ok {
		return rf(_a0...)
	}
	if rf, ok := ret.Get(0).(func(...types.CheckResult) []byte); ok {
		r0 = rf(_a0...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(...types.CheckResult) error); ok {
		r1 = rf(_a0...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Extract provides a mock function with given fields: _a0
func (_m *MockEncoder) Extract(_a0 []byte) ([]types.ReportedUpkeep, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for Extract")
	}

	var r0 []types.ReportedUpkeep
	var r1 error
	if rf, ok := ret.Get(0).(func([]byte) ([]types.ReportedUpkeep, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func([]byte) []types.ReportedUpkeep); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.ReportedUpkeep)
		}
	}

	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockEncoder creates a new instance of MockEncoder. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockEncoder(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockEncoder {
	mock := &MockEncoder{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
