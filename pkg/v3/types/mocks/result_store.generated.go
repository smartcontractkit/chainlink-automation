// Code generated by mockery v2.28.1. DO NOT EDIT.

package mocks

import (
	automation "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
	mock "github.com/stretchr/testify/mock"
)

// MockResultStore is an autogenerated mock type for the ResultStore type
type MockResultStore struct {
	mock.Mock
}

// Add provides a mock function with given fields: _a0
func (_m *MockResultStore) Add(_a0 ...automation.CheckResult) {
	_va := make([]interface{}, len(_a0))
	for _i := range _a0 {
		_va[_i] = _a0[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	_m.Called(_ca...)
}

// Remove provides a mock function with given fields: _a0
func (_m *MockResultStore) Remove(_a0 ...string) {
	_va := make([]interface{}, len(_a0))
	for _i := range _a0 {
		_va[_i] = _a0[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	_m.Called(_ca...)
}

// View provides a mock function with given fields:
func (_m *MockResultStore) View() ([]automation.CheckResult, error) {
	ret := _m.Called()

	var r0 []automation.CheckResult
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]automation.CheckResult, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []automation.CheckResult); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]automation.CheckResult)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMockResultStore interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockResultStore creates a new instance of MockResultStore. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockResultStore(t mockConstructorTestingTNewMockResultStore) *MockResultStore {
	mock := &MockResultStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
