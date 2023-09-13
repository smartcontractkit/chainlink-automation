package ocr_test

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"math/big"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/simulate/chain"
	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/simulate/ocr"
)

func TestOCR3ConfigLoader(t *testing.T) {
	t.Parallel()

	logger := log.New(io.Discard, "", 0)

	digester := new(mockDigester)
	runbook := config.RunBook{
		ConfigEvents: []config.ConfigEvent{{Block: big.NewInt(2)}},
	}

	loader := ocr.NewOCR3ConfigLoader(runbook, digester, logger)
	block := chain.Block{
		Number: big.NewInt(1),
	}

	digester.On("ConfigDigest", mock.Anything).Return(nil, nil)

	loader.Load(&block)
	require.Len(t, block.Transactions, 0, "no transactions at block 1")

	onKey, _ := config.NewEVMKeyring(rand.Reader)
	offKey, _ := config.NewOffchainKeyring(rand.Reader, rand.Reader)
	loader.AddSigner("signer", onKey, offKey)

	block.Number = big.NewInt(2)

	loader.Load(&block)
	require.Len(t, block.Transactions, 1, "1 transaction at block 2")
}

type mockDigester struct {
	mock.Mock
}

func (_m *mockDigester) ConfigDigest(config types.ContractConfig) (types.ConfigDigest, error) {
	req := _m.Called(config)

	hash := sha256.Sum256([]byte(fmt.Sprintf("%+v", config)))

	return hash, req.Error(1)
}
