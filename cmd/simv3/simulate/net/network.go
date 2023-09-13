package net

import (
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
)

type SimulatedBinaryNetworkEndpointFactory struct {
	PeerLookup map[uint8]string
	pID        string
	Network    *SimulatedNetwork
}

func (ef *SimulatedBinaryNetworkEndpointFactory) NewEndpoint(
	cd types.ConfigDigest,
	peerIDs []string,
	v2bootstrappers []commontypes.BootstrapperLocator,
	f int,
	limits types.BinaryNetworkEndpointLimits,
) (commontypes.BinaryNetworkEndpoint, error) {

	var thisID uint8
	for i, peer := range peerIDs {
		if peer == ef.pID {
			thisID = uint8(i)
		}
		ef.PeerLookup[uint8(i)] = peer
	}

	ch := ef.Network.RegisterEndpoint(ef.pID)
	ep := &SimulatedBinaryNetworkEndpoint{
		PeerLookup: ef.PeerLookup,
		ID:         thisID,
		Network:    ef.Network,
		chReceive:  ch,
	}

	return ep, nil
}

func (ef *SimulatedBinaryNetworkEndpointFactory) PeerID() string {
	return ef.pID
}

type SimulatedBinaryNetworkEndpoint struct {
	PeerLookup map[uint8]string
	ID         uint8
	Network    *SimulatedNetwork
	chReceive  chan commontypes.BinaryMessageWithSender
}

// SendTo(msg, to) sends msg to "to"
func (ne *SimulatedBinaryNetworkEndpoint) SendTo(payload []byte, to commontypes.OracleID) {
	peer, ok := ne.PeerLookup[uint8(to)]
	if ok {
		ne.Network.SendTo(ne.ID, payload, peer)
	}
}

// Broadcast(msg) sends msg to all oracles
func (ne *SimulatedBinaryNetworkEndpoint) Broadcast(payload []byte) {
	ne.Network.Broadcast(ne.ID, payload)
}

// Receive returns channel which carries all messages sent to this oracle.
func (ne *SimulatedBinaryNetworkEndpoint) Receive() <-chan commontypes.BinaryMessageWithSender {
	return ne.chReceive
}

// Start starts the endpoint
func (ne *SimulatedBinaryNetworkEndpoint) Start() error {
	return nil
}

// Close stops the endpoint. Calling this multiple times may return an
// error, but must not panic.
func (ne *SimulatedBinaryNetworkEndpoint) Close() error {
	return nil
}

type SimulatedNetwork struct {
	latency   int
	endpoints map[string]chan commontypes.BinaryMessageWithSender
}

func NewSimulatedNetwork(avgLatency time.Duration) *SimulatedNetwork {
	return &SimulatedNetwork{
		latency:   int(avgLatency / time.Millisecond),
		endpoints: make(map[string]chan commontypes.BinaryMessageWithSender),
	}
}

func (sn *SimulatedNetwork) NewFactory() *SimulatedBinaryNetworkEndpointFactory {
	f := &SimulatedBinaryNetworkEndpointFactory{
		PeerLookup: make(map[uint8]string),
		pID:        uuid.New().String(),
		Network:    sn,
	}

	return f
}

func (sn *SimulatedNetwork) RegisterEndpoint(id string) chan commontypes.BinaryMessageWithSender {
	ch := make(chan commontypes.BinaryMessageWithSender, 1000)
	sn.endpoints[id] = ch
	return ch
}

func (sn *SimulatedNetwork) SendTo(sender uint8, payload []byte, to string) {
	ch, ok := sn.endpoints[to]
	if ok {
		msg := commontypes.BinaryMessageWithSender{
			Msg:    payload,
			Sender: commontypes.OracleID(sender),
		}

		rn := rand.Intn(sn.latency)
		// simulate network delay
		<-time.After(time.Duration(rn) * time.Millisecond)

		ch <- msg
	}
}

func (sn *SimulatedNetwork) Broadcast(sender uint8, payload []byte) {
	rn := rand.Intn(100)
	// simulate network delay
	<-time.After(time.Duration(rn) * time.Millisecond)

	for _, ch := range sn.endpoints {
		/*
			if key == sender {
				continue
			}
		*/

		msg := commontypes.BinaryMessageWithSender{
			Msg:    payload,
			Sender: commontypes.OracleID(sender),
		}

		ch <- msg
	}
}
