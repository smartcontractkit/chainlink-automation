package upkeep_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulate/chain"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulate/upkeep"
)

func TestUpkeepConfigLoader(t *testing.T) {
	runbook := config.RunBook{
		BlockCadence: config.Blocks{
			Genesis:  big.NewInt(1),
			Cadence:  config.Duration(time.Second),
			Duration: 10,
		},
		Upkeeps: []config.Upkeep{
			{
				Count:        1,
				StartID:      big.NewInt(1),
				GenerateFunc: "2x",
				OffsetFunc:   "x",
			},
		},
	}

	loader, err := upkeep.NewUpkeepConfigLoader(runbook)

	require.NoError(t, err)

	block := chain.Block{
		Number:       runbook.BlockCadence.Genesis,
		Transactions: []interface{}{},
	}

	loader.Load(&block)

	assert.Len(t, block.Transactions, 1)
}
