package mocks

import (
	"context"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/stretchr/testify/mock"
)

type MockContractConfigTracker struct {
	mock.Mock
}

// Notify may optionally emit notification events when the contract's
// configuration changes. This is purely used as an optimization reducing
// the delay between a configuration change and its enactment. Implementors
// who don't care about this may simply return a nil channel.
//
// The returned channel should never be closed.
func (_m *MockContractConfigTracker) Notify() <-chan struct{} {
	ret := _m.Mock.Called()

	var r0 <-chan struct{}
	if rf, ok := ret.Get(0).(func() <-chan struct{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan struct{})
		}
	}

	return r0
}

// LatestConfigDetails returns information about the latest configuration,
// but not the configuration itself.
func (_m *MockContractConfigTracker) LatestConfigDetails(ctx context.Context) (changedInBlock uint64, configDigest types.ConfigDigest, err error) {
	ret := _m.Mock.Called(ctx)

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(uint64)
		}
	}

	var r1 types.ConfigDigest
	if rf, ok := ret.Get(1).(func() types.ConfigDigest); ok {
		r1 = rf()
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(types.ConfigDigest)
		}
	}

	return r0, r1, ret.Error(2)
}

// LatestConfig returns the latest configuration.
func (_m *MockContractConfigTracker) LatestConfig(ctx context.Context, changedInBlock uint64) (types.ContractConfig, error) {
	ret := _m.Mock.Called(ctx, changedInBlock)

	var r0 types.ContractConfig
	if rf, ok := ret.Get(0).(func() types.ContractConfig); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.ContractConfig)
		}
	}

	return r0, ret.Error(1)
}

// LatestBlockHeight returns the height of the most recent block in the chain.
func (_m *MockContractConfigTracker) LatestBlockHeight(ctx context.Context) (blockHeight uint64, err error) {
	ret := _m.Mock.Called(ctx)

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(uint64)
		}
	}

	return r0, ret.Error(1)
}
