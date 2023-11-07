package config

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimulationPlan_EncodeDecode(t *testing.T) {
	plan := SimulationPlan{
		Node: Node{
			Count:             4,
			MaxServiceWorkers: 10,
			MaxQueueSize:      1000,
		},
		Network: Network{
			MaxLatency: Duration(300 * time.Millisecond),
		},
		RPC: RPC{},
		Blocks: Blocks{
			Genesis:    big.NewInt(3),
			Cadence:    Duration(1 * time.Second),
			Jitter:     Duration(200 * time.Millisecond),
			Duration:   20,
			EndPadding: 20,
		},
		ConfigEvents:    []OCR3ConfigEvent{},
		GenerateUpkeeps: []GenerateUpkeepEvent{},
		LogEvents:       []LogTriggerEvent{},
	}

	encoded, err := plan.Encode()

	require.NoError(t, err, "no error expected from encoding the simulation plan")

	decodedPlan, err := DecodeSimulationPlan(encoded)

	require.NoError(t, err, "no error expected from decoding the simulation plan")

	assert.Equal(t, plan, decodedPlan, "simulation plan should match after encoding and decoding")
}
