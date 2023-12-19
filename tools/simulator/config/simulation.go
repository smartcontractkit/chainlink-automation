package config

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
)

var (
	ErrEncoding = fmt.Errorf("encoding/decoding failure")
)

// SimulationPlan is a collection of configurations with which to run a
// simulation.
type SimulationPlan struct {
	Node            Node                  `json:"node"`
	Network         Network               `json:"p2pNetwork"`
	RPC             RPC                   `json:"rpc"`
	Blocks          Blocks                `json:"blocks"`
	ConfigEvents    []OCR3ConfigEvent     `json:"-"`
	GenerateUpkeeps []GenerateUpkeepEvent `json:"-"`
	LogEvents       []LogTriggerEvent     `json:"-"`
}

// Encode applies JSON encoding of a simulation plan to bytes.
func (p SimulationPlan) Encode() ([]byte, error) {
	type encodedOutput struct {
		SimulationPlan
		Events []interface{} `json:"events"`
	}

	encodable := encodedOutput{
		SimulationPlan: p,
		Events:         make([]interface{}, len(p.ConfigEvents)+len(p.GenerateUpkeeps)),
	}

	for _, event := range p.ConfigEvents {
		// ensure the type is set properly
		event.Type = OCR3ConfigEventType
		encodable.Events = append(encodable.Events, event)
	}

	for _, event := range p.GenerateUpkeeps {
		// ensure the type is set properly
		event.Type = GenerateUpkeepEventType
		encodable.Events = append(encodable.Events, event)
	}

	for _, event := range p.LogEvents {
		// ensure the type is set properly
		event.Type = LogTriggerEventType
		encodable.Events = append(encodable.Events, event)
	}

	return json.Marshal(encodable)
}

// DecodeSimulationPlan uses JSON encoding to decode bytes to a simulation plan.
func DecodeSimulationPlan(encoded []byte) (SimulationPlan, error) {
	var plan SimulationPlan

	if err := json.Unmarshal(encoded, &plan); err != nil {
		return plan, fmt.Errorf("%w: failed to decode simulation plan: %s", ErrEncoding, err.Error())
	}

	plan.ConfigEvents = make([]OCR3ConfigEvent, 0)
	plan.GenerateUpkeeps = make([]GenerateUpkeepEvent, 0)
	plan.LogEvents = make([]LogTriggerEvent, 0)

	type eventCollection struct {
		Events []json.RawMessage `json:"events"`
	}

	var events eventCollection

	if err := json.Unmarshal(encoded, &events); err != nil {
		return plan, fmt.Errorf("%w: failed to decode events in simulation plan: %s", ErrEncoding, err.Error())
	}

	for idx, rawEvent := range events.Events {
		var event Event
		if err := json.Unmarshal(rawEvent, &event); err != nil {
			return plan, fmt.Errorf("%w: failed to decode event in simulation plan: %s", ErrEncoding, err.Error())
		}

		switch event.Type {
		case OCR3ConfigEventType:
			var configEvent OCR3ConfigEvent
			if err := json.Unmarshal(rawEvent, &configEvent); err != nil {
				return plan, fmt.Errorf("%w: failed to decode ocr3config event in simulation plan at index %d: %s", ErrEncoding, idx, err.Error())
			}

			plan.ConfigEvents = append(plan.ConfigEvents, configEvent)
		case GenerateUpkeepEventType:
			var generateEvent GenerateUpkeepEvent
			if err := json.Unmarshal(rawEvent, &generateEvent); err != nil {
				return plan, fmt.Errorf("%w: failed to decode generateUpkeep event in simulation plan at index %d: %s", ErrEncoding, idx, err.Error())
			}

			if generateEvent.Expected == "" {
				generateEvent.Expected = AllExpected
			}

			plan.GenerateUpkeeps = append(plan.GenerateUpkeeps, generateEvent)
		case LogTriggerEventType:
			var logEvent LogTriggerEvent
			if err := json.Unmarshal(rawEvent, &logEvent); err != nil {
				return plan, fmt.Errorf("%w: failed to decode logTrigger event in simulation plan at index %d: %s", ErrEncoding, idx, err.Error())
			}

			plan.LogEvents = append(plan.LogEvents, logEvent)
		default:
			return plan, fmt.Errorf("%w: unrecognized event at index %d", ErrEncoding, idx)
		}
	}

	return plan, nil
}

// Node is a configuration that applies to the simulated nodes.
type Node struct {
	// Count defines the total number of nodes added in the simulation.
	Count int `json:"totalNodeCount"`
	// MaxServiceWorkers is a configuration on the total number of go-routines
	// allowed to each node for running parallel pipeline calls.
	MaxServiceWorkers int `json:"maxNodeServiceWorkers"`
	// MaxQueueSize limits the queue size for incoming check pipeline requests.
	MaxQueueSize int `json:"maxNodeServiceQueueSize"`
}

// Network is a configuration for the simulated p2p network between simulated
// nodes.
type Network struct {
	// MaxLatency applies to the amout of time a message takes to be sent
	// between peers. This is intended to simulate delay due to physical
	// distance between nodes or other network delays.
	MaxLatency Duration `json:"maxLatency"`
}

// RPC is a configuration for a simulated RPC client. Each node recieves their
// own simulated rpc client which allows the configured values to be applied
// independently.
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

// Blocks is a configuration for simulated block production.
type Blocks struct {
	// Genesis is the starting block number.
	Genesis *big.Int `json:"genesisBlock"`
	// Cadence is how fast blocks are produced.
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
