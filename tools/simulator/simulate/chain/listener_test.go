package chain_test

import (
	"context"
	"io"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/smartcontractkit/chainlink-automation/tools/simulator/config"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/simulate/chain"
)

func TestListener(t *testing.T) {
	t.Parallel()

	logger := log.New(io.Discard, "", 0)
	conf := config.Blocks{
		Genesis:  new(big.Int).SetInt64(1),
		Cadence:  config.Duration(100 * time.Millisecond),
		Jitter:   config.Duration(0),
		Duration: 10,
	}

	broadcaster := chain.NewBlockBroadcaster(conf, 1, logger, nil, loadTestUpkeep)
	listener := chain.NewListener(broadcaster, logger)

	deadline, ok := t.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}

	ctx, cancel := context.WithDeadline(context.Background(), deadline)

	broadcaster.Start()

	select {
	case <-ctx.Done():
		t.Log("context deadline was passed before upkeep was broadcast")
		t.Fail()
	case <-listener.Subscribe(chain.CreateUpkeepChannel):
	}

	cancel()

	broadcaster.Stop()
}

func loadTestUpkeep(block *chain.Block) {
	if block.Number.Cmp(new(big.Int).SetInt64(5)) >= 1 {
		block.Transactions = append(block.Transactions, chain.UpkeepCreatedTransaction{})
	}
}
