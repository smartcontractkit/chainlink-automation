package ocr

import (
	"context"
	"sync"

	ocr2keepers "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
)

type Transmitter interface {
	Transmit(string, []byte, uint64) error
}

type OCR3Transmitter struct {
	// configured values
	transmitterID string
	loader        Transmitter

	// internal state values
	mu sync.RWMutex
}

func NewOCR3Transmitter(id string, loader Transmitter) *OCR3Transmitter {
	return &OCR3Transmitter{
		transmitterID: id,
		loader:        loader,
	}
}

func (tr *OCR3Transmitter) Transmit(
	ctx context.Context,
	digest types.ConfigDigest,
	v uint64,
	r ocr3types.ReportWithInfo[ocr2keepers.AutomationReportInfo],
	s []types.AttributedOnchainSignature,
) error {
	return tr.loader.Transmit(tr.transmitterID, []byte(r.Report), v)
}

// Account from which the transmitter invokes the contract
func (tr *OCR3Transmitter) FromAccount() (types.Account, error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	return types.Account(tr.transmitterID), nil
}
