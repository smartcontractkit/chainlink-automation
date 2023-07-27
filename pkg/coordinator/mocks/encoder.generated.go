// Code generated by mockery v2.32.0. DO NOT EDIT.

package mocks

import (
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	mock "github.com/stretchr/testify/mock"
)

// Encoder is an autogenerated mock type for the Encoder type
type Encoder struct {
	mock.Mock
}

// After provides a mock function with given fields: _a0, _a1
func (_m *Encoder) After(_a0 ocr2keepers.BlockKey, _a1 ocr2keepers.BlockKey) (bool, error) {
	ret := _m.Called(_a0, _a1)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(ocr2keepers.BlockKey, ocr2keepers.BlockKey) (bool, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(ocr2keepers.BlockKey, ocr2keepers.BlockKey) bool); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(ocr2keepers.BlockKey, ocr2keepers.BlockKey) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Increment provides a mock function with given fields: _a0
func (_m *Encoder) Increment(_a0 ocr2keepers.BlockKey) (ocr2keepers.BlockKey, error) {
	ret := _m.Called(_a0)

	var r0 ocr2keepers.BlockKey
	var r1 error
	if rf, ok := ret.Get(0).(func(ocr2keepers.BlockKey) (ocr2keepers.BlockKey, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(ocr2keepers.BlockKey) ocr2keepers.BlockKey); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(ocr2keepers.BlockKey)
	}

	if rf, ok := ret.Get(1).(func(ocr2keepers.BlockKey) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SplitUpkeepKey provides a mock function with given fields: _a0
func (_m *Encoder) SplitUpkeepKey(_a0 ocr2keepers.UpkeepKey) (ocr2keepers.BlockKey, ocr2keepers.UpkeepIdentifier, error) {
	ret := _m.Called(_a0)

	var r0 ocr2keepers.BlockKey
	var r1 ocr2keepers.UpkeepIdentifier
	var r2 error
	if rf, ok := ret.Get(0).(func(ocr2keepers.UpkeepKey) (ocr2keepers.BlockKey, ocr2keepers.UpkeepIdentifier, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(ocr2keepers.UpkeepKey) ocr2keepers.BlockKey); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(ocr2keepers.BlockKey)
	}

	if rf, ok := ret.Get(1).(func(ocr2keepers.UpkeepKey) ocr2keepers.UpkeepIdentifier); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(ocr2keepers.UpkeepIdentifier)
		}
	}

	if rf, ok := ret.Get(2).(func(ocr2keepers.UpkeepKey) error); ok {
		r2 = rf(_a0)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// NewEncoder creates a new instance of Encoder. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewEncoder(t interface {
	mock.TestingT
	Cleanup(func())
}) *Encoder {
	mock := &Encoder{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
