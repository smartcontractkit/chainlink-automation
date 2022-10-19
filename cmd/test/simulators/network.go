package simulators

import (
	"github.com/google/uuid"
	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

type SimulatedBinaryNetworkEndpointFactory struct {
	ID       string
	OracleID uint8
	Network  *SimulatedNetwork
}

func (ef *SimulatedBinaryNetworkEndpointFactory) NewEndpoint(
	cd types.ConfigDigest,
	peerIDs []string,
	v2bootstrappers []commontypes.BootstrapperLocator,
	f int,
	limits types.BinaryNetworkEndpointLimits,
) (commontypes.BinaryNetworkEndpoint, error) {
	ch := ef.Network.RegisterEndpoint(ef.OracleID)
	ep := &SimulatedBinaryNetworkEndpoint{
		ID:        ef.OracleID,
		Network:   ef.Network,
		chReceive: ch,
	}

	return ep, nil
}

func (ef *SimulatedBinaryNetworkEndpointFactory) PeerID() string {
	return ef.ID
}

type SimulatedBinaryNetworkEndpoint struct {
	ID        uint8
	Network   *SimulatedNetwork
	chReceive chan commontypes.BinaryMessageWithSender
}

// SendTo(msg, to) sends msg to "to"
func (ne *SimulatedBinaryNetworkEndpoint) SendTo(payload []byte, to commontypes.OracleID) {
	ne.Network.SendTo(ne.ID, payload, uint8(to))
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
	nextOracle uint8
	endpoints  map[uint8]chan commontypes.BinaryMessageWithSender
}

func NewSimulatedNetwork() *SimulatedNetwork {
	return &SimulatedNetwork{
		endpoints: make(map[uint8]chan commontypes.BinaryMessageWithSender),
	}
}

func (sn *SimulatedNetwork) NewFactory() *SimulatedBinaryNetworkEndpointFactory {
	f := &SimulatedBinaryNetworkEndpointFactory{
		ID:       uuid.New().String(),
		OracleID: sn.nextOracle,
		Network:  sn,
	}

	sn.nextOracle++

	return f
}

func (sn *SimulatedNetwork) RegisterEndpoint(id uint8) chan commontypes.BinaryMessageWithSender {
	ch := make(chan commontypes.BinaryMessageWithSender, 1000)
	sn.endpoints[id] = ch
	return ch
}

func (sn *SimulatedNetwork) SendTo(sender uint8, payload []byte, to uint8) {
	ch, ok := sn.endpoints[to]
	if ok {
		msg := commontypes.BinaryMessageWithSender{
			Msg:    payload,
			Sender: commontypes.OracleID(sender),
		}

		ch <- msg
	}
}

func (sn *SimulatedNetwork) Broadcast(sender uint8, payload []byte) {
	for key, ch := range sn.endpoints {
		if key == sender {
			continue
		}

		msg := commontypes.BinaryMessageWithSender{
			Msg:    payload,
			Sender: commontypes.OracleID(sender),
		}

		ch <- msg
	}
}
