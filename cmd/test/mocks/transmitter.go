package mocks

import (
	"context"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/stretchr/testify/mock"
)

type MockContractTransmitter struct {
	mock.Mock
}

// Transmit sends the report to the on-chain OCR2Aggregator smart
// contract's Transmit method.
//
// In most cases, implementations of this function should store the
// transmission in a queue/database/..., but perform the actual
// transmission (and potentially confirmation) of the transaction
// asynchronously.
func (_m *MockContractTransmitter) Transmit(
	ctx context.Context,
	rc types.ReportContext,
	r types.Report,
	s []types.AttributedOnchainSignature,
) error {
	return _m.Mock.Called(ctx, rc, r, s).Error(0)
}

// LatestConfigDigestAndEpoch returns the logically latest configDigest and
// epoch for which a report was successfully transmitted.
func (_m *MockContractTransmitter) LatestConfigDigestAndEpoch(
	ctx context.Context,
) (
	configDigest types.ConfigDigest,
	epoch uint32,
	err error,
) {
	ret := _m.Mock.Called(ctx)

	var r0 types.ConfigDigest
	if rf, ok := ret.Get(0).(func() types.ConfigDigest); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.ConfigDigest)
		}
	}

	var r1 uint32
	if rf, ok := ret.Get(1).(func() uint32); ok {
		r1 = rf()
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(uint32)
		}
	}

	return r0, r1, ret.Error(2)
}

// Account from which the transmitter invokes the contract
func (_m *MockContractTransmitter) FromAccount() types.Account {
	ret := _m.Mock.Called()

	var r0 types.Account
	if rf, ok := ret.Get(0).(func() types.Account); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.Account)
		}
	}

	return r0
}
