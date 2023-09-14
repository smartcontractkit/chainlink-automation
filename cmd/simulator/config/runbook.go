package config

import (
	"encoding/json"
	"math/big"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
)

// TODO: change the name of this to SimulationPlan
type RunBook struct {
	Nodes             int           `json:"nodes"`
	MaxServiceWorkers int           `json:"maxNodeServiceWorkers"`
	MaxQueueSize      int           `json:"maxNodeServiceQueueSize"`
	AvgNetworkLatency Duration      `json:"avgNetworkLatency"`
	RPCDetail         RPC           `json:"rpcDetail"`
	BlockCadence      Blocks        `json:"blockDetail"`
	ConfigEvents      []ConfigEvent `json:"configEvents"`
	Upkeeps           []Upkeep      `json:"upkeeps"`
}

func (rb RunBook) Encode() ([]byte, error) {
	return json.Marshal(rb)
}

func LoadRunBook(b []byte) (RunBook, error) {
	var rb RunBook

	err := json.Unmarshal(b, &rb)
	if err != nil {
		return rb, err
	}

	return rb, nil
}

type Blocks struct {
	Genesis *big.Int `json:"genesisBlock"`
	Cadence Duration `json:"blockCadence"`
	// Jitter is the average amount of variance applied to the cadence
	Jitter Duration `json:"blockCadenceJitter"`
	// Duration is the number of blocks to simulate before blocks should stop
	// broadcasting
	Duration int `json:"durationInBlocks"`
	// EndPadding is the number of blocks to add to the end of the process to
	// allow all transmits to close up for the simulated test
	EndPadding int `json:"endPadding"`
}

type RPC struct {
	// MaxBlockDelay is the maximum amount of time in ms that a block would take
	// to be viewed by the node
	MaxBlockDelay int `json:"maxBlockDelay"`
	// AverageLatency is the average amount of time in ms that an RPC network
	// call can take
	AverageLatency int `json:"averageLatency"`
	// ErrorRate is the chance that any RPC call will return an error. This helps
	// simulated heavily loaded RPC servers.
	ErrorRate float64 `json:"errorRate"`
	// RateLimitThreshold is the point at which rate limiting occurs for RPC calls.
	// this limit is calls per second
	RateLimitThreshold int `json:"rateLimitThreshold"`
}

// ConfigEvent is an event that indicates a new config should be broadcast
type ConfigEvent struct {
	// Block is the block number where this event is triggered
	Block *big.Int `json:"triggerBlockNumber"`
	// F is the configurable faulty number of nodes
	F int `json:"maxFaultyNodes"`
	// Offchain is the json encoded off chain config data
	Offchain string `json:"offchainConfigJSON"`
	// Rmax is the maximum number of rounds in an epoch
	Rmax uint64 `json:"maxRoundsPerEpoch"`
	// DeltaProgress is the OCR setting for round leader progress before forcing
	// a new epoch and leader
	DeltaProgress Duration `json:"deltaProgress"`
	// DeltaResend ...
	DeltaResend Duration `json:"deltaResend"`
	// DeltaInitial ...
	DeltaInitial Duration `json:"deltaInitial"`
	// DeltaRound is the approximate time a round should complete in
	DeltaRound Duration `json:"deltaRound"`
	// DeltaGrace ...
	DeltaGrace Duration `json:"deltaGrace"`
	// DeltaRequest ...
	DeltaRequest Duration `json:"deltaCertifiedCommitRequest"`
	// DeltaStage is the time OCR waits before attempting a followup transmit
	DeltaStage Duration `json:"deltaStage"`
	// MaxQuery ...
	MaxQuery Duration `json:"maxQueryTime"`
	// MaxObservation is the maximum amount of time to provide observation to complete
	MaxObservation Duration `json:"maxObservationTime"`
	// MaxAccept ...
	MaxAccept Duration `json:"maxShouldAcceptTime"`
	// MaxTransmit ...
	MaxTransmit Duration `json:"maxShouldTransmitTime"`
}

type Upkeep struct {
	Count        int      `json:"count"`
	StartID      *big.Int `json:"startID"`
	GenerateFunc string   `json:"generateFunc"`
	OffsetFunc   string   `json:"offsetFunc"`
}

type SymBlock struct {
	BlockNumber     *big.Int
	TransmittedData [][]byte
	LatestEpoch     *uint32
	Config          *types.ContractConfig
}