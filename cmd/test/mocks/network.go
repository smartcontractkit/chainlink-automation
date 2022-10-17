package mocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	protocoltypes "github.com/smartcontractkit/libocr/offchainreporting2/types"
)

type MockBinaryNetworkEndpointFactory struct {
	mock.Mock
}

func (_m *MockBinaryNetworkEndpointFactory) NewEndpoint(
	cd protocoltypes.ConfigDigest,
	peerIDs []string,
	v2bootstrappers []commontypes.BootstrapperLocator,
	f int,
	limits types.BinaryNetworkEndpointLimits,
) (commontypes.BinaryNetworkEndpoint, error) {
	ret := _m.Mock.Called(cd, peerIDs, v2bootstrappers, f, limits)

	var r0 commontypes.BinaryNetworkEndpoint
	if rf, ok := ret.Get(0).(func() commontypes.BinaryNetworkEndpoint); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(commontypes.BinaryNetworkEndpoint)
		}
	}

	return r0, ret.Error(1)
}

func (_m *MockBinaryNetworkEndpointFactory) PeerID() string {
	ret := _m.Mock.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(string)
		}
	}

	return r0
}

type MockBinaryNetworkEndpoint struct {
	mock.Mock
}

// SendTo(msg, to) sends msg to "to"
func (_m *MockBinaryNetworkEndpoint) SendTo(payload []byte, to commontypes.OracleID) {
	_m.Mock.Called(payload, to)
}

// Broadcast(msg) sends msg to all oracles
func (_m *MockBinaryNetworkEndpoint) Broadcast(payload []byte) {
	_m.Mock.Called(payload)
}

// Receive returns channel which carries all messages sent to this oracle.
func (_m *MockBinaryNetworkEndpoint) Receive() <-chan commontypes.BinaryMessageWithSender {
	ret := _m.Mock.Called()

	var r0 <-chan commontypes.BinaryMessageWithSender
	if rf, ok := ret.Get(0).(func() <-chan commontypes.BinaryMessageWithSender); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan commontypes.BinaryMessageWithSender)
		}
	}

	return r0
}

// Start starts the endpoint
func (_m *MockBinaryNetworkEndpoint) Start() error {
	return _m.Mock.Called().Error(0)
}

// Close stops the endpoint. Calling this multiple times may return an
// error, but must not panic.
func (_m *MockBinaryNetworkEndpoint) Close() error {
	return _m.Mock.Called().Error(0)
}
