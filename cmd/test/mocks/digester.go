package mocks

import (
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/stretchr/testify/mock"
)

type MockOffchainConfigDigester struct {
	mock.Mock
}

// Compute ConfigDigest for the given ContractConfig. The first two bytes of the
// ConfigDigest must be the big-endian encoding of ConfigDigestPrefix!
func (_m *MockOffchainConfigDigester) ConfigDigest(config types.ContractConfig) (types.ConfigDigest, error) {
	ret := _m.Mock.Called(config)

	var r0 types.ConfigDigest
	if rf, ok := ret.Get(0).(func() types.ConfigDigest); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.ConfigDigest)
		}
	}

	return r0, ret.Error(1)
}

// This should return the same constant value on every invocation
func (_m *MockOffchainConfigDigester) ConfigDigestPrefix() types.ConfigDigestPrefix {
	ret := _m.Mock.Called()

	var r0 types.ConfigDigestPrefix
	if rf, ok := ret.Get(0).(func() types.ConfigDigestPrefix); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.ConfigDigestPrefix)
		}
	}

	return r0
}
