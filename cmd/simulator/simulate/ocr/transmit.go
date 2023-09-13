package ocr

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"log"
	"sync"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"

	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/simulate/chain"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/plugin"
)

type OCR3TransmitLoader struct {
	// provided dependencies
	logger *log.Logger

	// internal state values
	mu          sync.RWMutex
	queue       []*chain.TransmitEvent
	transmitted map[string]*chain.TransmitEvent
}

// NewOCR3TransmitLoader ...
func NewOCR3TransmitLoader(_ config.RunBook, logger *log.Logger) *OCR3TransmitLoader {
	return &OCR3TransmitLoader{
		logger:      log.New(logger.Writer(), "[ocr3-transmit-loader] ", log.Ldate|log.Ltime|log.Lshortfile),
		queue:       make([]*chain.TransmitEvent, 0),
		transmitted: make(map[string]*chain.TransmitEvent),
	}
}

func (tl *OCR3TransmitLoader) Load(block *chain.Block) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if len(tl.queue) == 0 {
		return
	}

	transmits := make([]chain.TransmitEvent, 0, len(tl.queue))

	for i := range tl.queue {
		tl.queue[i].BlockNumber = block.Number
		tl.queue[i].BlockHash = block.Hash

		transmits = append(transmits, *tl.queue[i])
	}

	tl.queue = []*chain.TransmitEvent{}

	block.Transactions = append(block.Transactions, chain.PerformUpkeepTransaction{
		Transmits: transmits,
	})
}

func (tl *OCR3TransmitLoader) Transmit(from string, reportBytes []byte, round uint64) error {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	report := chain.TransmitEvent{
		Report: reportBytes,
		Round:  round,
	}

	var idHashBts bytes.Buffer
	if err := gob.NewEncoder(&idHashBts).Encode(report); err != nil {
		return err
	}

	transmitKey := hash(idHashBts.Bytes())

	report.SendingAddress = from

	var bts bytes.Buffer
	if err := gob.NewEncoder(&bts).Encode(report); err != nil {
		return err
	}

	report.Hash = rawHash(bts.Bytes())

	if _, ok := tl.transmitted[transmitKey]; ok {
		return fmt.Errorf("report already transmitted")
	}

	tl.logger.Printf("transmit sent from %s in round %d", from, round)

	tl.queue = append(tl.queue, &report)
	tl.transmitted[transmitKey] = &report

	return nil
}

func (tl *OCR3TransmitLoader) Results() []chain.TransmitEvent {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	events := []chain.TransmitEvent{}
	for _, r := range tl.transmitted {
		events = append(events, *r)
	}

	tl.logger.Printf("%d transmits returned in final results", len(events))

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

func rawHash(b []byte) [32]byte {
	hasher := sha256.New()
	hasher.Write(b)

	var sum [32]byte

	copy(sum[:], hasher.Sum(nil)[:])

	return sum
}

func hash(b []byte) string {
	hasher := sha256.New()
	hasher.Write(b)

	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}
