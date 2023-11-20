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

func TestLogTriggerTracker(t *testing.T) {
	t.Parallel()

	logger := log.New(io.Discard, "", 0)
	conf := config.Blocks{
		Genesis:  new(big.Int).SetInt64(1),
		Cadence:  config.Duration(50 * time.Millisecond),
		Jitter:   config.Duration(0),
		Duration: 10,
	}

	upkeep1 := chain.SimulatedUpkeep{
		ID:          big.NewInt(10),
		UpkeepID:    [32]byte{},
		Type:        chain.LogTriggerType,
		TriggeredBy: "test_trigger",
	}

	chainLog1 := chain.Log{
		TxHash:       [32]byte{},
		BlockNumber:  big.NewInt(4),
		BlockHash:    [32]byte{},
		Idx:          uint32(1),
		TriggerValue: "test_trigger",
	}

	broadcaster := chain.NewBlockBroadcaster(conf, 1, logger, nil, loadUpkeepAt(upkeep1, 3), loadLogAt(chainLog1, 4))
	listener := chain.NewListener(broadcaster, logger)
	active := upkeep.NewActiveTracker(listener, logger)
	performs := upkeep.NewPerformTracker(listener, logger)

	logTracker := upkeep.NewLogTriggerTracker(listener, active, performs, logger)

	<-broadcaster.Start()

	assert.Len(t, logTracker.GetAfter(big.NewInt(3)), 0)
	assert.Len(t, logTracker.GetOnce(), 1)
	assert.Len(t, logTracker.GetOnce(), 0)
	assert.Len(t, logTracker.GetAfter(big.NewInt(5)), 1)
	assert.Len(t, logTracker.GetAfter(big.NewInt(3)), 0)

	broadcaster.Stop()
}

func TestLogTriggerTracker_NoUpkeeps(t *testing.T) {
	t.Parallel()

	logger := log.New(io.Discard, "", 0)
	conf := config.Blocks{
		Genesis:  new(big.Int).SetInt64(1),
		Cadence:  config.Duration(50 * time.Millisecond),
		Jitter:   config.Duration(0),
		Duration: 10,
	}

	chainLog1 := chain.Log{
		BlockNumber:  big.NewInt(4),
		Idx:          uint32(1),
		TriggerValue: "test_trigger",
	}

	broadcaster := chain.NewBlockBroadcaster(conf, 1, logger, nil, loadLogAt(chainLog1, 4))
	listener := chain.NewListener(broadcaster, logger)
	active := upkeep.NewActiveTracker(listener, logger)
	performs := upkeep.NewPerformTracker(listener, logger)

	logTracker := upkeep.NewLogTriggerTracker(listener, active, performs, logger)

	<-broadcaster.Start()

	assert.Len(t, logTracker.GetOnce(), 0)
	assert.Len(t, logTracker.GetOnce(), 0)
	assert.Len(t, logTracker.GetAfter(big.NewInt(5)), 0)
	assert.Len(t, logTracker.GetAfter(big.NewInt(3)), 0)

	broadcaster.Stop()
}

func TestLogTriggerTracker_Performed(t *testing.T) {
	t.Parallel()

	logger := log.New(io.Discard, "", 0)
	conf := config.Blocks{
		Genesis:  new(big.Int).SetInt64(1),
		Cadence:  config.Duration(50 * time.Millisecond),
		Jitter:   config.Duration(0),
		Duration: 10,
	}

	upkeep1 := chain.SimulatedUpkeep{
		ID:          big.NewInt(10),
		Type:        chain.LogTriggerType,
		TriggeredBy: "test_trigger",
	}

	chainLog1 := chain.Log{
		BlockNumber:  big.NewInt(4),
		Idx:          uint32(1),
		TriggerValue: "test_trigger",
	}

	report1, err := util.EncodeCheckResultsToReportBytes([]ocr2keepers.CheckResult{
		{
			UpkeepID: upkeep1.UpkeepID,
			WorkID: util.UpkeepWorkID(
				upkeep1.UpkeepID,
				ocr2keepers.NewLogTrigger(
					ocr2keepers.BlockNumber(5),
					[32]byte{},
					&ocr2keepers.LogTriggerExtension{
						TxHash:      chainLog1.TxHash,
						Index:       chainLog1.Idx,
						BlockHash:   chainLog1.BlockHash,
						BlockNumber: ocr2keepers.BlockNumber(chainLog1.BlockNumber.Uint64()),
					})),
		},
	})
	require.NoError(t, err)

	perform1 := []chain.TransmitEvent{
		{
			Report:      report1,
			BlockNumber: big.NewInt(5),
		},
	}

	broadcaster := chain.NewBlockBroadcaster(conf, 1, logger, nil, loadUpkeepAt(upkeep1, 3), loadLogAt(chainLog1, 4), loadPerformAt(perform1, 5))
	listener := chain.NewListener(broadcaster, logger)
	active := upkeep.NewActiveTracker(listener, logger)
	performs := upkeep.NewPerformTracker(listener, logger)

	logTracker := upkeep.NewLogTriggerTracker(listener, active, performs, logger)

	<-broadcaster.Start()

	assert.Len(t, logTracker.GetAfter(big.NewInt(3)), 0)
	assert.Len(t, logTracker.GetOnce(), 1)
	assert.Len(t, logTracker.GetOnce(), 0)
	assert.Len(t, logTracker.GetAfter(big.NewInt(5)), 0)
	assert.Len(t, logTracker.GetAfter(big.NewInt(3)), 0)

	broadcaster.Stop()
}

func loadLogAt(chainLog chain.Log, atBlock int64) func(*chain.Block) {
	return func(block *chain.Block) {
		if block.Number.Cmp(new(big.Int).SetInt64(atBlock)) == 0 {
			block.Transactions = append(block.Transactions, chainLog)
		}
	}
}
