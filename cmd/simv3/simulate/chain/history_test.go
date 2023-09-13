package chain_test

import (
	"context"
	"io"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulate/chain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockHistoryTracker(t *testing.T) {
	t.Parallel()

	logger := log.New(io.Discard, "", 0)
	conf := config.Blocks{
		Genesis:  new(big.Int).SetInt64(1),
		Cadence:  config.Duration(100 * time.Millisecond),
		Jitter:   config.Duration(0),
		Duration: 10,
	}

	broadcaster := chain.NewBlockBroadcaster(conf, 1, logger, loadTestUpkeep)
	listener := chain.NewListener(broadcaster, logger)

	deadline, ok := t.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}

	tracker := chain.NewBlockHistoryTracker(listener, logger)

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	idx, chHistory, err := tracker.Subscribe()

	require.NoError(t, err, "no error expected from subscribe to tracker")

	broadcaster.Start()
	select {
	case <-ctx.Done():
		t.Log("context deadline was passed before upkeep was broadcast")
		t.Fail()
	case <-chHistory:
	}

	err = tracker.Unsubscribe(idx)

	assert.NoError(t, err, "no error expected when unsubscribing from tracker")

	cancel()

	broadcaster.Stop()
}
