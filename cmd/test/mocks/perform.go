package mocks

import (
	"context"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/stretchr/testify/mock"
)

type MockPerformLogProvider struct {
	mock.Mock
}

func (_m *MockPerformLogProvider) PerformLogs(ctx context.Context) ([]types.PerformLog, error) {
	ret := _m.Mock.Called(ctx)

	var r0 []types.PerformLog
	if rf, ok := ret.Get(0).(func() []types.PerformLog); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.PerformLog)
		}
	}

	return r0, ret.Error(1)
}
