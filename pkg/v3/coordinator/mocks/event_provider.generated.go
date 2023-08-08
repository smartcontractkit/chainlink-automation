// Code generated by mockery v2.28.1. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	types "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

// EventProvider is an autogenerated mock type for the EventProvider type
type EventProvider struct {
	mock.Mock
}

// Events provides a mock function with given fields: _a0
func (_m *EventProvider) Events(_a0 context.Context) ([]types.TransmitEvent, error) {
	ret := _m.Called(_a0)

	var r0 []types.TransmitEvent
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) ([]types.TransmitEvent, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(context.Context) []types.TransmitEvent); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.TransmitEvent)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewEventProvider interface {
	mock.TestingT
	Cleanup(func())
}

// NewEventProvider creates a new instance of EventProvider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewEventProvider(t mockConstructorTestingTNewEventProvider) *EventProvider {
	mock := &EventProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
