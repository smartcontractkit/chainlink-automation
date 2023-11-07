package upkeep_test

import (
	"io"
	"log"
	"math/big"
	"testing"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	"github.com/smartcontractkit/ocr2keepers/tools/simulator/config"
	"github.com/smartcontractkit/ocr2keepers/tools/simulator/simulate/chain"
	"github.com/smartcontractkit/ocr2keepers/tools/simulator/simulate/upkeep"
	"github.com/smartcontractkit/ocr2keepers/tools/simulator/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerformTracker_LogTrigger(t *testing.T) {
	t.Parallel()

	logger := log.New(io.Discard, "", 0)
	conf := config.Blocks{
		Genesis:  new(big.Int).SetInt64(1),
		Cadence:  config.Duration(50 * time.Millisecond),
		Jitter:   config.Duration(0),
		Duration: 10,
	}

	upkeepID := util.NewUpkeepID(big.NewInt(8).Bytes(), uint8(ocr2keepers.LogTrigger))

	workID := util.UpkeepWorkID(
		upkeepID,
		ocr2keepers.NewLogTrigger(
			ocr2keepers.BlockNumber(5),
			[32]byte{},
			&ocr2keepers.LogTriggerExtension{
				BlockNumber: ocr2keepers.BlockNumber(5),
			}))

	report1, err := util.EncodeCheckResultsToReportBytes([]ocr2keepers.CheckResult{
		{
			UpkeepID: upkeepID,
			WorkID:   workID,
		},
	})
	require.NoError(t, err)

	perform1 := []chain.TransmitEvent{
		{
			Report:      report1,
			BlockNumber: big.NewInt(5),
		},
	}

	broadcaster := chain.NewBlockBroadcaster(conf, 1, logger, nil, loadPerformAt(perform1, 5))
	listener := chain.NewListener(broadcaster, logger)
	tracker := upkeep.NewPerformTracker(listener, logger)

	<-broadcaster.Start()

	// should return 0 because the reported upkeep is log trigger type
	assert.Len(t, tracker.PerformsForUpkeepID(ocr2keepers.UpkeepIdentifier(upkeepID).String()), 0)

	// should return true because the work id was encountered
	assert.True(t, tracker.IsWorkIDPerformed(workID))

	broadcaster.Stop()
}

func TestPerformTracker_Conditional(t *testing.T) {
	t.Parallel()

	logger := log.New(io.Discard, "", 0)
	conf := config.Blocks{
		Genesis:  new(big.Int).SetInt64(1),
		Cadence:  config.Duration(50 * time.Millisecond),
		Jitter:   config.Duration(0),
		Duration: 10,
	}

	upkeepID := util.NewUpkeepID(big.NewInt(8).Bytes(), uint8(ocr2keepers.ConditionTrigger))
	workID := util.UpkeepWorkID(
		upkeepID,
		ocr2keepers.NewLogTrigger(
			ocr2keepers.BlockNumber(5),
			[32]byte{},
			nil))

	report1, err := util.EncodeCheckResultsToReportBytes([]ocr2keepers.CheckResult{
		{
			UpkeepID: upkeepID,
			WorkID:   workID,
		},
	})

	require.NoError(t, err)

	perform1 := []chain.TransmitEvent{
		{
			Report:      report1,
			BlockNumber: big.NewInt(5),
		},
	}

	broadcaster := chain.NewBlockBroadcaster(conf, 1, logger, nil, loadPerformAt(perform1, 5))
	listener := chain.NewListener(broadcaster, logger)
	tracker := upkeep.NewPerformTracker(listener, logger)

	<-broadcaster.Start()

	// should return 1 because the reported upkeep is conditional trigger type
	assert.Len(t, tracker.PerformsForUpkeepID(ocr2keepers.UpkeepIdentifier(upkeepID).String()), 1)

	// should return true because the work id was encountered
	assert.True(t, tracker.IsWorkIDPerformed(workID))

	broadcaster.Stop()
}

func TestPerformTracker_DecodeReportFailure(t *testing.T) {
	t.Parallel()

	logger := log.New(io.Discard, "", 0)
	conf := config.Blocks{
		Genesis:  new(big.Int).SetInt64(1),
		Cadence:  config.Duration(50 * time.Millisecond),
		Jitter:   config.Duration(0),
		Duration: 10,
	}

	upkeepID := util.NewUpkeepID(big.NewInt(8).Bytes(), uint8(ocr2keepers.ConditionTrigger))
	workID := util.UpkeepWorkID(
		upkeepID,
		ocr2keepers.NewLogTrigger(
			ocr2keepers.BlockNumber(5),
			[32]byte{},
			nil))

	perform1 := []chain.TransmitEvent{
		{
			Report:      []byte("0x"),
			BlockNumber: big.NewInt(5),
		},
	}

	broadcaster := chain.NewBlockBroadcaster(conf, 1, logger, nil, loadPerformAt(perform1, 5))
	listener := chain.NewListener(broadcaster, logger)
	tracker := upkeep.NewPerformTracker(listener, logger)

	<-broadcaster.Start()

	// should return 0 because the report should not be decodable
	assert.Len(t, tracker.PerformsForUpkeepID(ocr2keepers.UpkeepIdentifier(upkeepID).String()), 0)

	// should return false because the report should not be decodable
	assert.False(t, tracker.IsWorkIDPerformed(workID))

	broadcaster.Stop()
}

func loadPerformAt(events []chain.TransmitEvent, atBlock int64) func(*chain.Block) {
	return func(block *chain.Block) {
		if block.Number.Cmp(new(big.Int).SetInt64(atBlock)) == 0 {
			block.Transactions = append(block.Transactions, chain.PerformUpkeepTransaction{
				Transmits: events,
			})
		}
	}
}
