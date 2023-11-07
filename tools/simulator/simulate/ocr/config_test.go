package ocr_test

import (
	"context"
	"crypto/sha256"
	"io"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
	"github.com/smartcontractkit/ocr2keepers/tools/simulator/config"
	"github.com/smartcontractkit/ocr2keepers/tools/simulator/simulate/chain"
	"github.com/smartcontractkit/ocr2keepers/tools/simulator/simulate/ocr"
)

func TestOCR3ConfigTracker(t *testing.T) {
	t.Parallel()

	logger := log.New(io.Discard, "", 0)
	conf := config.Blocks{
		Genesis:  new(big.Int).SetInt64(1),
		Cadence:  config.Duration(100 * time.Millisecond),
		Jitter:   config.Duration(0),
		Duration: 5,
	}

	ocrConfig := types.ContractConfig{
		ConfigDigest: sha256.Sum256([]byte("some config data")),
	}

	broadcaster := chain.NewBlockBroadcaster(conf, 1, logger, nil, loadConfigAt(ocrConfig, 2))
	listener := chain.NewListener(broadcaster, logger)
	tracker := ocr.NewOCR3ConfigTracker(listener, logger)

	broadcaster.Start()

	<-tracker.Notify()

	broadcaster.Stop()

	changedInBlock, digest, err := tracker.LatestConfigDetails(context.Background())

	require.NoError(t, err)

	assert.Equal(t, uint64(2), changedInBlock, "changed in block should be equal to the block loaded at")
	assert.Equal(t, ocrConfig.ConfigDigest, digest, "config digest should match")

	latest, err := tracker.LatestConfig(context.Background(), 0)

	require.NoError(t, err)

	assert.Equal(t, ocrConfig, latest, "configs should match")

	blockHeight, err := tracker.LatestBlockHeight(context.Background())

	require.NoError(t, err)

	assert.Greater(t, blockHeight, uint64(1), "should advance at least higher than 2")
}

func loadConfigAt(ocrConfig types.ContractConfig, atBlock int64) func(*chain.Block) {
	return func(block *chain.Block) {
		if block.Number.Cmp(new(big.Int).SetInt64(atBlock)) == 0 {
			block.Transactions = append(block.Transactions, chain.OCR3ConfigTransaction{
				Config: ocrConfig,
			})
		}
	}
}
