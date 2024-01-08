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

	ocr2keepers "github.com/smartcontractkit/chainlink-common/pkg/types/automation"

	"github.com/smartcontractkit/chainlink-automation/tools/simulator/config"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/simulate/chain"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/simulate/upkeep"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/util"
)

func TestSource_GetActiveUpkeeps(t *testing.T) {
	t.Parallel()

	logger := log.New(io.Discard, "", 0)
	conf := config.Blocks{
		Genesis:  new(big.Int).SetInt64(1),
		Cadence:  config.Duration(50 * time.Millisecond),
		Jitter:   config.Duration(0),
		Duration: 5,
	}

	upkeep1 := chain.SimulatedUpkeep{
		ID:       big.NewInt(10),
		UpkeepID: [32]byte{},
		Type:     chain.ConditionalType,
	}

	broadcaster := chain.NewBlockBroadcaster(conf, 1, logger, nil, loadUpkeepAt(upkeep1, 2))
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

func TestSource_GetLatestPayloads(t *testing.T) {
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

	broadcaster := chain.NewBlockBroadcaster(conf, 1, logger, nil, loadUpkeepAt(upkeep1, 3), loadLogAt(chainLog1, 4))
	listener := chain.NewListener(broadcaster, logger)
	active := upkeep.NewActiveTracker(listener, logger)
	performs := upkeep.NewPerformTracker(listener, logger)
	triggered := upkeep.NewLogTriggerTracker(listener, active, performs, logger)

	src := upkeep.NewSource(active, triggered, 10, logger)

	// empty to start
	payloads, err := src.GetLatestPayloads(context.Background())

	require.NoError(t, err)
	assert.Len(t, payloads, 0)

	<-broadcaster.Start()

	// should have 1 upkeep after block source completes
	payloads, err = src.GetLatestPayloads(context.Background())

	require.NoError(t, err)
	assert.Len(t, payloads, 1)

	broadcaster.Stop()
}

func TestSource_GetRecoveryProposals(t *testing.T) {
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

	broadcaster := chain.NewBlockBroadcaster(conf, 1, logger, nil, loadUpkeepAt(upkeep1, 3), loadLogAt(chainLog1, 4))
	listener := chain.NewListener(broadcaster, logger)
	active := upkeep.NewActiveTracker(listener, logger)
	performs := upkeep.NewPerformTracker(listener, logger)
	triggered := upkeep.NewLogTriggerTracker(listener, active, performs, logger)

	src := upkeep.NewSource(active, triggered, 10, logger)

	// empty to start
	proposals, err := src.GetRecoveryProposals(context.Background())

	require.NoError(t, err)
	assert.Len(t, proposals, 0)

	<-broadcaster.Start()

	// call get once to move triggered upkeeps to read state
	_ = triggered.GetOnce()

	// should have 1 upkeep after block source completes
	proposals, err = src.GetRecoveryProposals(context.Background())

	require.NoError(t, err)
	assert.Len(t, proposals, 1)

	broadcaster.Stop()
}

func TestSource_BuildPayloads(t *testing.T) {
	t.Parallel()

	logger := log.New(io.Discard, "", 0)
	conf := config.Blocks{
		Genesis:  new(big.Int).SetInt64(1),
		Cadence:  config.Duration(50 * time.Millisecond),
		Jitter:   config.Duration(0),
		Duration: 5,
	}

	upkeep1 := chain.SimulatedUpkeep{
		ID:       big.NewInt(10),
		UpkeepID: [32]byte{},
		Type:     chain.ConditionalType,
	}

	trigger1 := ocr2keepers.NewLogTrigger(
		ocr2keepers.BlockNumber(5),
		[32]byte{},
		nil)

	workID := util.UpkeepWorkID(upkeep1.UpkeepID, trigger1)

	broadcaster := chain.NewBlockBroadcaster(conf, 1, logger, nil, loadUpkeepAt(upkeep1, 2))
	listener := chain.NewListener(broadcaster, logger)
	active := upkeep.NewActiveTracker(listener, logger)

	src := upkeep.NewSource(active, nil, 0, logger)

	<-broadcaster.Start()

	// should have 1 upkeep after block source completes
	payloads, err := src.BuildPayloads(context.Background(), ocr2keepers.CoordinatedBlockProposal{
		UpkeepID: upkeep1.UpkeepID,
		Trigger:  trigger1,
		WorkID:   workID,
	})

	require.NoError(t, err)
	require.Len(t, payloads, 1)

	assert.Equal(t, workID, payloads[0].WorkID)

	broadcaster.Stop()
}
