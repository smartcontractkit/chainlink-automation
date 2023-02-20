// Code generated by mockery v2.12.1. DO NOT EDIT.

package types

import (
	context "context"
	testing "testing"

	mock "github.com/stretchr/testify/mock"
)

// MockRegistry is an autogenerated mock type for the Registry type
type MockRegistry struct {
	mock.Mock
}

// CheckUpkeep provides a mock function with given fields: _a0, _a1
func (_m *MockRegistry) CheckUpkeep(_a0 context.Context, _a1 ...UpkeepKey) (UpkeepResults, error) {
	_va := make([]interface{}, len(_a1))
	for _i := range _a1 {
		_va[_i] = _a1[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _a0)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 UpkeepResults
	if rf, ok := ret.Get(0).(func(context.Context, ...UpkeepKey) UpkeepResults); ok {
		r0 = rf(_a0, _a1...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(UpkeepResults)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, ...UpkeepKey) error); ok {
		r1 = rf(_a0, _a1...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetActiveUpkeepIDs provides a mock function with given fields: _a0
func (_m *MockRegistry) GetActiveUpkeepIDs(_a0 context.Context) ([]UpkeepIdentifier, error) {
	ret := _m.Called(_a0)

	var r0 []UpkeepIdentifier
	if rf, ok := ret.Get(0).(func(context.Context) []UpkeepIdentifier); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]UpkeepIdentifier)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockRegistry creates a new instance of MockRegistry. It also registers the testing.TB interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockRegistry(t testing.TB) *MockRegistry {
	mock := &MockRegistry{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
