package loader_test

import (
	"io"
	"log"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/ocr2keepers/tools/simulator/config"
	"github.com/smartcontractkit/ocr2keepers/tools/simulator/simulate/chain"
	"github.com/smartcontractkit/ocr2keepers/tools/simulator/simulate/loader"
)

func TestOCR3TransmitLoader(t *testing.T) {
	t.Parallel()

	logger := log.New(io.Discard, "", 0)
	loader, err := loader.NewOCR3TransmitLoader(config.SimulationPlan{}, nil, logger)

	require.NoError(t, err)

	block := chain.Block{
		Number:       big.NewInt(1),
		Transactions: []interface{}{},
	}

	// nothing to load
	loader.Load(&block)

	require.Len(t, block.Transactions, 0, "nothing should be added to the block on first load")
	require.NoError(t, loader.Transmit("test", []byte("message1"), 10_000))
	require.NoError(t, loader.Transmit("test", []byte("message2"), 10_000))
	require.NotNil(t, loader.Transmit("test", []byte("message2"), 10_000), "cannot transmit the same report")

	loader.Load(&block)

	require.Len(t, block.Transactions, 1, "both transmitted transactions should be included")

	trx, ok := block.Transactions[0].(chain.PerformUpkeepTransaction)
	require.True(t, ok, "transaction should be perform type")
	assert.Len(t, trx.Transmits, 2, "transaction should contain expected number of transmits")

	require.NoError(t, loader.Transmit("test", []byte("message3"), 10_000))
	require.Len(t, loader.Results(), 3, "return all transmitted results")
}
