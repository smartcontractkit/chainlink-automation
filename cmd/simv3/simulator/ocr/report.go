package ocr

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"runtime"
	"sync"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator/chain"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/util"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type ReportTracker struct {
	// provided dependencies
	listener *chain.Listener
	logger   *log.Logger

	// internal state props
	mu     sync.RWMutex
	events []chain.TransmitEvent
	read   []chain.TransmitEvent
	latest *chain.Block

	// service values
	chDone chan struct{}
}

// NewReportTracker ...
func NewReportTracker(listener *chain.Listener, logger *log.Logger) *ReportTracker {
	src := &ReportTracker{
		listener: listener,
		logger:   log.New(logger.Writer(), "[report-tracker]", log.LstdFlags),
		events:   make([]chain.TransmitEvent, 0),
		read:     make([]chain.TransmitEvent, 0),
		chDone:   make(chan struct{}),
	}

	go src.run()

	runtime.SetFinalizer(src, func(srv *ReportTracker) { srv.stop() })

	return src
}

// GetLatestEvents returns a list of events that are after a specified block
// threshold. Returns each event exactly once.
func (rt *ReportTracker) GetLatestEvents(_ context.Context) ([]ocr2keepers.TransmitEvent, error) {
	if rt.latest == nil {
		return nil, nil
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()

	events := make([]ocr2keepers.TransmitEvent, 0, len(rt.events))
	for _, event := range rt.events {
		transmits, err := createPluginTransmitEvents(event, *rt.latest)
		if err != nil {
			rt.logger.Println(err)
		}

		events = append(events, transmits...)
	}

	rt.read = append(rt.read, rt.events...)
	rt.events = []chain.TransmitEvent{}

	return events, nil
}

func (rt *ReportTracker) run() {
	chEvents := rt.listener.Subscribe(chain.PerformUpkeepChannel, chain.BlockChannel)

	for {
		select {
		case event := <-chEvents:
			switch evt := event.Event.(type) {
			case chain.PerformUpkeepTransaction:
				rt.logger.Printf("%d transmit events detected on block %s", len(evt.Transmits), event.BlockNumber)

				rt.saveTransmits(evt.Transmits)
			case chain.Block:
				rt.updateBlock(evt)
			}
		case <-rt.chDone:
			return
		}
	}
}

func (rt *ReportTracker) stop() {
	close(rt.chDone)
}

func (rt *ReportTracker) saveTransmits(transmits []chain.TransmitEvent) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rt.events = append(rt.events, transmits...)
}

func (rt *ReportTracker) updateBlock(block chain.Block) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rt.latest = &block
}

func createPluginTransmitEvents(chainEvent chain.TransmitEvent, latest chain.Block) ([]ocr2keepers.TransmitEvent, error) {
	var results []ocr2keepers.CheckResult

	results, err := util.DecodeCheckResultsFromReportBytes(chainEvent.Report)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal transmitted report: %w", err)
	}

	var (
		trHash [32]byte
		events []ocr2keepers.TransmitEvent
	)

	copy(trHash[:], chainEvent.Hash[:32])

	for _, result := range results {
		event := ocr2keepers.TransmitEvent{
			Type:            ocr2keepers.PerformEvent,
			TransmitBlock:   ocr2keepers.BlockNumber(chainEvent.BlockNumber.Uint64()),
			Confirmations:   new(big.Int).Sub(latest.Number, chainEvent.BlockNumber).Int64(),
			TransactionHash: trHash,
			UpkeepID:        result.UpkeepID,
			WorkID:          result.WorkID,
			CheckBlock:      result.Trigger.BlockNumber,
		}

		events = append(events, event)
	}

	return events, nil
}
