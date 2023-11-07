package loader_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/ocr2keepers/tools/simulator/config"
	"github.com/smartcontractkit/ocr2keepers/tools/simulator/simulate/chain"
	"github.com/smartcontractkit/ocr2keepers/tools/simulator/simulate/loader"
)

func TestUpkeepConfigLoader(t *testing.T) {
	plan := config.SimulationPlan{
		Blocks: config.Blocks{
			Genesis:  big.NewInt(1),
			Cadence:  config.Duration(time.Second),
			Duration: 10,
		},
		GenerateUpkeeps: []config.GenerateUpkeepEvent{
			{
				Event: config.Event{
					TriggerBlock: big.NewInt(2),
				},
				Count:           10,
				StartID:         big.NewInt(1),
				EligibilityFunc: "2x",
				OffsetFunc:      "x",
				UpkeepType:      config.ConditionalUpkeepType,
			},
		},
	}
	telemetry := new(mockProgressTelemetry)

	telemetry.On("Register", loader.CreateUpkeepNamespace, int64(10)).Return(nil)
	telemetry.On("Increment", loader.CreateUpkeepNamespace, int64(10))

	loader, err := loader.NewUpkeepConfigLoader(plan, telemetry)

	require.NoError(t, err)

	block := chain.Block{
		Number:       big.NewInt(2),
		Transactions: []interface{}{},
	}

	loader.Load(&block)

	assert.Len(t, block.Transactions, 10)
}

type mockProgressTelemetry struct {
	mock.Mock
}

func (_m *mockProgressTelemetry) Register(namespace string, total int64) error {
	res := _m.Called(namespace, total)

	return res.Error(0)
}

func (_m *mockProgressTelemetry) Increment(namespace string, count int64) {
	_m.Called(namespace, count)
}
