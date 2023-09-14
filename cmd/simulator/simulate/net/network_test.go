package net_test

import (
	"testing"
	"time"

	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/simulate/net"
)

func TestSimulatedNetwork(t *testing.T) {
	t.Parallel()

	latency := 50 * time.Millisecond
	network := net.NewSimulatedNetwork(latency)

	bMsg := []byte("message")
	peerIDs := []string{}

	f1 := network.NewFactory()
	peerIDs = append(peerIDs, f1.PeerID())

	f2 := network.NewFactory()
	peerIDs = append(peerIDs, f2.PeerID())

	f3 := network.NewFactory()
	peerIDs = append(peerIDs, f3.PeerID())

	f1ep1, err := f1.NewEndpoint([32]byte{}, peerIDs, []commontypes.BootstrapperLocator{}, 0, types.BinaryNetworkEndpointLimits{})

	assert.Nil(t, f1ep1.Start())
	assert.Nil(t, f1ep1.Close())

	require.NoError(t, err)

	f2ep1, err := f2.NewEndpoint([32]byte{}, peerIDs, []commontypes.BootstrapperLocator{}, 0, types.BinaryNetworkEndpointLimits{})

	require.NoError(t, err)

	f3ep1, err := f3.NewEndpoint([32]byte{}, peerIDs, []commontypes.BootstrapperLocator{}, 0, types.BinaryNetworkEndpointLimits{})

	require.NoError(t, err)

	f1ep1.SendTo(bMsg, 1)

	msg := <-f2ep1.Receive()

	assert.Equal(t, bMsg, msg.Msg)
	assert.Equal(t, commontypes.OracleID(0), msg.Sender)

	f2ep1.Broadcast(bMsg)

	<-f1ep1.Receive()
	<-f2ep1.Receive()
	<-f3ep1.Receive()
}

func TestSimulatedNetwork_NewFactory(t *testing.T) {
	t.Parallel()

	latency := 50 * time.Millisecond
	network := net.NewSimulatedNetwork(latency)
	factory := network.NewFactory()

	require.NotNil(t, factory)
	assert.NotNil(t, factory.Network)

	// test that peer lookup map has been initialized
	factory.PeerLookup[0] = "test"
}

func TestSimulatedNetwork_SendTo(t *testing.T) {
	t.Parallel()

	latency := 50 * time.Millisecond
	network := net.NewSimulatedNetwork(latency)

	bMsg := []byte("message")
	chMsg := network.RegisterEndpoint("receiver")

	start := time.Now()
	network.SendTo(0, bMsg, "receiver")

	msg := <-chMsg

	tolerance := 20 * time.Millisecond
	elapsed := time.Since(start)

	assert.LessOrEqual(t, elapsed, latency+tolerance, "message send time should be less than or equal to configured latency")
	assert.Equal(t, bMsg, msg.Msg)
}

func TestSimulatedNetwork_Broadcast(t *testing.T) {
	t.Parallel()

	latency := 50 * time.Millisecond
	network := net.NewSimulatedNetwork(latency)

	bMsg := []byte("message")
	chReceiver1 := network.RegisterEndpoint("receiver1")
	chReceiver2 := network.RegisterEndpoint("receiver2")
	chReceiver3 := network.RegisterEndpoint("receiver3")

	network.Broadcast(0, bMsg)

	msg1 := <-chReceiver1
	msg2 := <-chReceiver2
	msg3 := <-chReceiver3

	assert.Equal(t, bMsg, msg1.Msg)
	assert.Equal(t, bMsg, msg2.Msg)
	assert.Equal(t, bMsg, msg3.Msg)
}
