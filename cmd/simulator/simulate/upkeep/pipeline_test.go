package upkeep

import (
	"context"
	"io"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/simulate/chain"
	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/util"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func TestCheckPipeline(t *testing.T) {
	t.Parallel()

	logger := log.New(io.Discard, "", 0)
	conf := config.RunBook{
		BlockCadence: config.Blocks{
			Genesis:  new(big.Int).SetInt64(1),
			Cadence:  config.Duration(50 * time.Millisecond),
			Jitter:   config.Duration(0),
			Duration: 5,
		},
		RPCDetail: config.RPC{
			ErrorRate:          0.0,
			RateLimitThreshold: 10,
			AverageLatency:     10,
		},
	}

	upkeep1 := chain.SimulatedUpkeep{
		ID:   big.NewInt(10),
		Type: chain.LogTriggerType,
	}

	trigger1 := ocr2keepers.NewLogTrigger(
		ocr2keepers.BlockNumber(5),
		[32]byte{},
		nil)

	workID := util.UpkeepWorkID(upkeep1.UpkeepID, trigger1)

	netTel := new(mockNetTelemetry)
	conTel := new(mockCheckTelemetry)

	broadcaster := chain.NewBlockBroadcaster(conf.BlockCadence, 1, logger, loadUpkeepAt(upkeep1, 2))
	listener := chain.NewListener(broadcaster, logger)
	active := NewActiveTracker(listener, logger)
	performs := NewPerformTracker(listener, logger)

	<-broadcaster.Start()
	broadcaster.Stop()

	netTel.On("Register", "checkUpkeep", mock.Anything, nil)
	netTel.On("Register", "simulatePerform", mock.Anything, nil)
	netTel.On("AddRateDataPoint", mock.Anything)
	conTel.On("CheckID", ocr2keepers.UpkeepIdentifier(upkeep1.UpkeepID).String(), mock.Anything, mock.Anything)

	pipeline := NewCheckPipeline(conf, active, performs, netTel, conTel, logger)

	results, err := pipeline.CheckUpkeeps(context.Background(), ocr2keepers.UpkeepPayload{
		UpkeepID: upkeep1.UpkeepID,
		Trigger:  trigger1,
		WorkID:   workID,
	})

	require.NoError(t, err)
	assert.Len(t, results, 1)

	netTel.AssertExpectations(t)
	conTel.AssertExpectations(t)
}

func TestIsConditionalEligible(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		eligibles []*big.Int
		performs  []*big.Int
		block     *big.Int
		expected  bool
	}{
		{
			name:      "eligible no performs",
			eligibles: []*big.Int{big.NewInt(10), big.NewInt(20), big.NewInt(30)},
			performs:  []*big.Int{},
			block:     big.NewInt(15), // block is between first and second eligible
			expected:  true,
		},
		{
			name:      "not eligible no performs",
			eligibles: []*big.Int{big.NewInt(10), big.NewInt(20)},
			performs:  []*big.Int{},
			block:     big.NewInt(9), // block is before first eligible
			expected:  false,
		},
		{
			name:      "eligible with 1 perform",
			eligibles: []*big.Int{big.NewInt(10), big.NewInt(20), big.NewInt(30)},
			performs:  []*big.Int{big.NewInt(15)},
			block:     big.NewInt(21), // block is between second and third eligible
			expected:  true,
		},
		{
			name:      "not eligible with 1 perform",
			eligibles: []*big.Int{big.NewInt(10), big.NewInt(20), big.NewInt(30)},
			performs:  []*big.Int{big.NewInt(15)},
			block:     big.NewInt(18), // block is between first and second eligible
			expected:  false,
		},
		{
			name:      "eligible with 2 performs",
			eligibles: []*big.Int{big.NewInt(10), big.NewInt(20), big.NewInt(30)},
			performs:  []*big.Int{big.NewInt(15), big.NewInt(22)},
			block:     big.NewInt(31), // block is after third eligible
			expected:  true,
		},
		{
			name:      "not eligible with 2 performs",
			eligibles: []*big.Int{big.NewInt(10), big.NewInt(20), big.NewInt(30)},
			performs:  []*big.Int{big.NewInt(15), big.NewInt(22)},
			block:     big.NewInt(26), // block is between second and third eligible
			expected:  false,
		},
	}

	for i := range tests {
		eligible := isConditionalEligible(tests[i].eligibles, tests[i].performs, tests[i].block)

		if eligible != tests[i].expected {
			t.Logf("%s was %t but was not expected", tests[i].name, eligible)
			t.Fail()
		}
	}
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

type mockNetTelemetry struct {
	mock.Mock
}

func (_m *mockNetTelemetry) Register(callName string, duration time.Duration, err error) {
	_m.Called(callName, duration, err)
}

func (_m *mockNetTelemetry) AddRateDataPoint(rate int) {
	_m.Called(rate)
}

type mockCheckTelemetry struct {
	mock.Mock
}

func (_m *mockCheckTelemetry) CheckID(upkeepID string, blockNumber uint64, blockHash [32]byte) {
	_m.Called(upkeepID, blockNumber, blockHash)
}
