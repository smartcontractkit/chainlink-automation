package ocr

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"math/big"
	"sync"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator/chain"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/plugin"
)

type OCR3TransmitLoader struct {
	// internal state values
	mu          sync.RWMutex
	queue       []chain.TransmitEvent
	transmitted map[string]chain.TransmitEvent
}

// NewOCR3TransmitLoader ...
func NewOCR3TransmitLoader(rb config.RunBook, logger *log.Logger) *OCR3TransmitLoader {
	eventLookup := make(map[*big.Int]config.ConfigEvent)

	for _, event := range rb.ConfigEvents {
		eventLookup[event.Block] = event
	}

	return &OCR3TransmitLoader{
		queue:       make([]chain.TransmitEvent, 0),
		transmitted: make(map[string]chain.TransmitEvent),
	}
}

func (tl *OCR3TransmitLoader) Load(block *chain.Block) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if len(tl.queue) == 0 {
		return
	}

	transmits := make([]chain.TransmitEvent, len(tl.queue))

	copy(transmits, tl.queue)

	tl.queue = []chain.TransmitEvent{}

	for _, r := range tl.queue {
		r.InBlock = block.Number.String()
	}

	block.Transactions = append(block.Transactions, chain.PerformUpkeepTransaction{
		Transmits: transmits,
	})
}

func (tl *OCR3TransmitLoader) Transmit(from string, reportBytes []byte, round uint64) error {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	reportHash := hash(reportBytes)

	if _, ok := tl.transmitted[reportHash]; ok {
		return fmt.Errorf("report already transmitted")
	}

	report := chain.TransmitEvent{
		InBlock:        "0",
		SendingAddress: from,
		Report:         reportBytes,
		Hash:           reportHash,
		Round:          round,
	}

	tl.queue = append(tl.queue, report)
	tl.transmitted[reportHash] = report

	return nil
}

func (tl *OCR3TransmitLoader) Results() []chain.TransmitEvent {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	events := []chain.TransmitEvent{}
	for _, r := range tl.transmitted {
		events = append(events, r)
	}

	return events
}

type OCR3Transmitter struct {
	// configured values
	transmitterID string
	loader        *OCR3TransmitLoader

	// internal state values
	mu sync.RWMutex
}

func NewOCR3Transmitter(id string, loader *OCR3TransmitLoader) *OCR3Transmitter {
	return &OCR3Transmitter{
		transmitterID: id,
		loader:        loader,
	}
}

func (tr *OCR3Transmitter) Transmit(
	ctx context.Context,
	digest types.ConfigDigest,
	v uint64,
	r ocr3types.ReportWithInfo[plugin.AutomationReportInfo],
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

func hash(b []byte) string {
	hasher := sha256.New()
	hasher.Write(b)

	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}
