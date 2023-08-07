// Code generated by mockery v2.22.1. DO NOT EDIT.

package mocks

import (
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	pkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"

	"github.com/stretchr/testify/mock"
)

// MockResultViewer is an autogenerated mock type for the ResultViewer type
type MockResultViewer struct {
	mock.Mock
}

// View provides a mock function with given fields: _a0
func (_m *MockResultViewer) View(_a0 ...ocr2keepers.ViewOpt) ([]pkg.CheckResult, error) {
	_va := make([]interface{}, len(_a0))
	for _i := range _a0 {
		_va[_i] = _a0[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 []pkg.CheckResult
	var r1 error
	if rf, ok := ret.Get(0).(func(...ocr2keepers.ViewOpt) ([]pkg.CheckResult, error)); ok {
		return rf(_a0...)
	}
	if rf, ok := ret.Get(0).(func(...ocr2keepers.ViewOpt) []pkg.CheckResult); ok {
		r0 = rf(_a0...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]pkg.CheckResult)
		}
	}

	if rf, ok := ret.Get(1).(func(...ocr2keepers.ViewOpt) error); ok {
		r1 = rf(_a0...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMockResultViewer interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockResultViewer creates a new instance of MockResultViewer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockResultViewer(t mockConstructorTestingTNewMockResultViewer) *MockResultViewer {
	mock := &MockResultViewer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
