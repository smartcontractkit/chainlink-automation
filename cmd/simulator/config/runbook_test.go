package config

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunbook_Encode(t *testing.T) {
	rb := RunBook{
		Nodes:             4,
		MaxServiceWorkers: 10,
		MaxQueueSize:      1000,
		AvgNetworkLatency: Duration(300 * time.Millisecond),
		RPCDetail:         RPC{},
		BlockCadence: Blocks{
			Genesis:    big.NewInt(3),
			Cadence:    Duration(1 * time.Second),
			Jitter:     Duration(200 * time.Millisecond),
			Duration:   20,
			EndPadding: 20,
		},
		ConfigEvents: []ConfigEvent{},
		Upkeeps:      []Upkeep{},
	}

	_, err := rb.Encode()

	assert.NoError(t, err, "no error expected from encoding the runbook")
}
