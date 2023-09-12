package upkeep_test

import (
	"context"
	"io"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator/chain"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator/upkeep"
)

func TestSource_GetActiveUpkeeps(t *testing.T) {
	t.Parallel()

	logger := log.New(io.Discard, "", 0)
	conf := config.Blocks{
		Genesis:  new(big.Int).SetInt64(1),
		Cadence:  config.Duration(100 * time.Millisecond),
		Jitter:   config.Duration(0),
		Duration: 5,
	}

	upkeep1 := chain.SimulatedUpkeep{
		ID:       big.NewInt(10),
		UpkeepID: [32]byte{},
		Type:     chain.ConditionalType,
	}

	broadcaster := chain.NewBlockBroadcaster(conf, 1, logger, loadUpkeepAt(upkeep1, 2))
	listener := chain.NewListener(broadcaster, logger)
	active := upkeep.NewActiveTracker(listener, logger)

	src := upkeep.NewSource(active, nil, 0, logger)

	// empty to start
	payloads, err := src.GetActiveUpkeeps(context.Background())

	require.NoError(t, err)
	assert.Len(t, payloads, 0)

	<-broadcaster.Start()

	// should have 1 upkeep after block source completes
	payloads, err = src.GetActiveUpkeeps(context.Background())

	require.NoError(t, err)
	assert.Len(t, payloads, 1)

	broadcaster.Stop()
}
