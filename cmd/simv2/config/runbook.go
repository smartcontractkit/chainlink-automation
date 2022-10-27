package config

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

type RunBook struct {
	Nodes             int
	AvgNetworkLatency Duration
	RPCDetail         RPC
	BlockCadence      Blocks
	ConfigEvents      []ConfigEvent
	Upkeeps           []Upkeep
}

type Blocks struct {
	Genesis *big.Int
	Cadence time.Duration
	// Duration is the number of blocks to simulate before blocks should stop
	// broadcasting
	Duration int
}

type RPC struct {
	// MaxBlockDelay is the maximum amount of time in ms that a block would take
	// to be viewed by the node
	MaxBlockDelay int
	// AverageLatency is the average amount of time in ms that an RPC network
	// call can take
	AverageLatency int
}

// ConfigEvent is an event that indicates a new config should be broadcast
type ConfigEvent struct {
	// Block is the block number where this event is triggered
	Block *big.Int
	// F is the configurable faulty number of nodes
	F int
	// Offchain is the json encoded off chain config data
	Offchain []byte
	// Rmax is the maximum number of rounds in an epoch
	Rmax uint8
	// DeltaProgress is the OCR setting for round leader progress before forcing
	// a new epoch and leader
	DeltaProgress Duration
	// DeltaResend ...
	DeltaResend Duration
	// DeltaRound is the approximate time a round should complete in
	DeltaRound Duration
	// DeltaGrace ...
	DeltaGrace Duration
	// DeltaStage is the time OCR waits before attempting a followup transmit
	DeltaStage Duration
	// MaxObservation is the maximum amount of time to provide observation to complete
	MaxObservation Duration
	// MaxReport is the maximum amount of time to provide report to complete
	MaxReport Duration
	// MaxAccept ...
	MaxAccept Duration
	// MaxTransmit ...
	MaxTransmit Duration
}

type SymBlock struct {
	BlockNumber     *big.Int
	TransmittedData [][]byte
	LatestEpoch     *uint32
	Config          *types.ContractConfig
}

type Upkeep struct {
	Count        int
	StartID      *big.Int
	GenerateFunc string
	OffsetFunc   string
}

type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	p, err := time.ParseDuration(raw)
	if err != nil {
		return err
	}

	*d = Duration(p)
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(time.Duration(d).String()), nil
}

func (d Duration) Value() time.Duration {
	return time.Duration(d)
}
