package chain_test

import (
	"io"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/simulate/chain"
)

func TestBlockBroadcaster(t *testing.T) {
	t.Parallel()

	conf := config.Blocks{
		Genesis:    big.NewInt(10),
		Cadence:    config.Duration(50 * time.Millisecond),
		Jitter:     config.Duration(10 * time.Millisecond),
		Duration:   5,
		EndPadding: 0,
	}
	maxDelay := 10
	logger := log.New(io.Discard, "", 0)
	loader := new(mockBlockLoader)
	broadcaster := chain.NewBlockBroadcaster(conf, maxDelay, logger, loader.Load)

	sub1ID, chBlocks1 := broadcaster.Subscribe(true)
	sub2ID, chBlocks2 := broadcaster.Subscribe(true)
	_ = broadcaster.Start()

	<-chBlocks1
	<-chBlocks2

	broadcaster.Unsubscribe(sub1ID)
	broadcaster.Unsubscribe(sub2ID)
	broadcaster.Stop()

	assert.True(t, loader.called)
}

func TestBlockBroadcaster_Close_After_Limit(t *testing.T) {
	t.Parallel()

	conf := config.Blocks{
		Genesis:    big.NewInt(10),
		Cadence:    config.Duration(50 * time.Millisecond),
		Jitter:     config.Duration(10 * time.Millisecond),
		Duration:   5,
		EndPadding: 1,
	}
	maxDelay := 10
	logger := log.New(io.Discard, "", 0)
	broadcaster := chain.NewBlockBroadcaster(conf, maxDelay, logger)

	closed := broadcaster.Start()

	<-closed
}

type mockBlockLoader struct {
	called bool
}

func (_m *mockBlockLoader) Load(_ *chain.Block) {
	_m.called = true
}
