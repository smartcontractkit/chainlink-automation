// Code generated by mockery v2.22.1. DO NOT EDIT.

package mocks

import (
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	mock "github.com/stretchr/testify/mock"
)

// MockEncoder is an autogenerated mock type for the Encoder type
type MockEncoder struct {
	mock.Mock
}

// Encode provides a mock function with given fields: _a0
func (_m *MockEncoder) Encode(_a0 ...ocr2keepers.CheckResult) ([]byte, error) {
	_va := make([]interface{}, len(_a0))
	for _i := range _a0 {
		_va[_i] = _a0[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(...ocr2keepers.CheckResult) ([]byte, error)); ok {
		return rf(_a0...)
	}
	if rf, ok := ret.Get(0).(func(...ocr2keepers.CheckResult) []byte); ok {
		r0 = rf(_a0...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(...ocr2keepers.CheckResult) error); ok {
		r1 = rf(_a0...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Extract provides a mock function with given fields: _a0
func (_m *MockEncoder) Extract(_a0 []byte) ([]ocr2keepers.ReportedUpkeep, error) {
	ret := _m.Called(_a0)

	var r0 []ocr2keepers.ReportedUpkeep
	var r1 error
	if rf, ok := ret.Get(0).(func([]byte) ([]ocr2keepers.ReportedUpkeep, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func([]byte) []ocr2keepers.ReportedUpkeep); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]ocr2keepers.ReportedUpkeep)
		}
	}

	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMockEncoder interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockEncoder creates a new instance of MockEncoder. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockEncoder(t mockConstructorTestingTNewMockEncoder) *MockEncoder {
	mock := &MockEncoder{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}