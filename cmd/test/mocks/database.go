package mocks

import (
	"context"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/stretchr/testify/mock"
)

type MockDatabase struct {
	mock.Mock
}

func (_m *MockDatabase) ReadState(ctx context.Context, configDigest types.ConfigDigest) (*types.PersistentState, error) {
	ret := _m.Mock.Called(ctx, configDigest)

	var r0 *types.PersistentState
	if rf, ok := ret.Get(0).(func() *types.PersistentState); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.PersistentState)
		}
	}

	return r0, ret.Error(1)
}

func (_m *MockDatabase) WriteState(ctx context.Context, configDigest types.ConfigDigest, state types.PersistentState) error {
	return _m.Mock.Called(ctx, configDigest, state).Error(0)
}

func (_m *MockDatabase) ReadConfig(ctx context.Context) (*types.ContractConfig, error) {
	ret := _m.Mock.Called(ctx)

	var r0 *types.ContractConfig
	if rf, ok := ret.Get(0).(func() *types.ContractConfig); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ContractConfig)
		}
	}

	return r0, ret.Error(1)
}

func (_m *MockDatabase) WriteConfig(ctx context.Context, config types.ContractConfig) error {
	return _m.Mock.Called(ctx, config).Error(0)
}

func (_m *MockDatabase) StorePendingTransmission(ctx context.Context, ts types.ReportTimestamp, tr types.PendingTransmission) error {
	return _m.Mock.Called(ctx, ts, tr).Error(0)
}

func (_m *MockDatabase) PendingTransmissionsWithConfigDigest(ctx context.Context, digest types.ConfigDigest) (map[types.ReportTimestamp]types.PendingTransmission, error) {
	ret := _m.Mock.Called(ctx, digest)

	var r0 map[types.ReportTimestamp]types.PendingTransmission
	if rf, ok := ret.Get(0).(func() map[types.ReportTimestamp]types.PendingTransmission); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[types.ReportTimestamp]types.PendingTransmission)
		}
	}

	return r0, ret.Error(1)
}

func (_m *MockDatabase) DeletePendingTransmission(ctx context.Context, ts types.ReportTimestamp) error {
	return _m.Mock.Called(ctx, ts).Error(0)
}

func (_m *MockDatabase) DeletePendingTransmissionsOlderThan(ctx context.Context, tm time.Time) error {
	return _m.Mock.Called(ctx, tm).Error(0)
}
