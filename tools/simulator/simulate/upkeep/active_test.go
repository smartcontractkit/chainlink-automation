package upkeep_test

import (
	"io"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/config"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/simulate/chain"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/simulate/upkeep"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/util"
)

func TestActiveTracker(t *testing.T) {
	t.Parallel()

	logger := log.New(io.Discard, "", 0)
	conf := config.Blocks{
		Genesis:  new(big.Int).SetInt64(1),
		Cadence:  config.Duration(100 * time.Millisecond),
		Jitter:   config.Duration(0),
		Duration: 10,
	}

	upkeep1 := chain.SimulatedUpkeep{
		ID:       big.NewInt(8),
		UpkeepID: util.NewUpkeepID(big.NewInt(8).Bytes(), uint8(ocr2keepers.ConditionTrigger)),
		Type:     chain.ConditionalType,
	}

	upkeep2 := chain.SimulatedUpkeep{
		ID:       big.NewInt(10),
		UpkeepID: util.NewUpkeepID(big.NewInt(10).Bytes(), uint8(ocr2keepers.LogTrigger)),
		Type:     chain.LogTriggerType,
	}

	broadcaster := chain.NewBlockBroadcaster(conf, 1, logger, nil, loadUpkeepAt(upkeep1, 2), loadUpkeepAt(upkeep2, 2))
	listener := chain.NewListener(broadcaster, logger)

	tracker := upkeep.NewActiveTracker(listener, logger)

	<-broadcaster.Start()
	broadcaster.Stop()

	assert.Len(t, tracker.GetAllByType(chain.ConditionalType), 1, "should have 1 conditional upkeep")
	assert.Len(t, tracker.GetAllByType(chain.LogTriggerType), 1, "should have 1 log upkeeps")

	trackedUpkeep, ok := tracker.GetByUpkeepID(upkeep1.UpkeepID)

	require.True(t, ok)
	assert.Equal(t, upkeep1, trackedUpkeep)

	var otherID [32]byte
	otherID[5] = 1

	_, ok = tracker.GetByUpkeepID(otherID)

	assert.False(t, ok)

	bl := tracker.GetLatestBlock()

	assert.GreaterOrEqual(t, 1, bl.Number.Cmp(conf.Genesis))
}

func loadUpkeepAt(upkeep chain.SimulatedUpkeep, atBlock int64) func(*chain.Block) {
	return func(block *chain.Block) {
		if block.Number.Cmp(new(big.Int).SetInt64(atBlock)) == 0 {
			block.Transactions = append(block.Transactions, chain.UpkeepCreatedTransaction{
				Upkeep: upkeep,
			})
		}
	}
}
