package ocr

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"runtime"
	"sync"

	ocr2keepers "github.com/smartcontractkit/chainlink-common/pkg/types/automation"

	"github.com/smartcontractkit/chainlink-automation/tools/simulator/simulate/chain"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/util"
)

const (
	ReportTrackerBlockRange = 100
)

type ReportTracker struct {
	// provided dependencies
	listener *chain.Listener
	logger   *log.Logger

	// internal state props
	mu          sync.RWMutex
	blockEvents *util.SortedKeyMap[[]chain.TransmitEvent]
	latest      *chain.Block

	// service values
	chDone chan struct{}
}

// NewReportTracker ...
func NewReportTracker(listener *chain.Listener, logger *log.Logger) *ReportTracker {
	src := &ReportTracker{
		listener:    listener,
		logger:      log.New(logger.Writer(), "[report-tracker] ", log.Ldate|log.Ltime|log.Lshortfile),
		blockEvents: util.NewSortedKeyMap[[]chain.TransmitEvent](),
		chDone:      make(chan struct{}),
	}

	go src.run()

	runtime.SetFinalizer(src, func(srv *ReportTracker) { srv.stop() })

	return src
}

// GetLatestEvents returns a list of events within a lookback range. Events
// returned from this function should follow an 'at least once' delivery.
func (rt *ReportTracker) GetLatestEvents(_ context.Context) ([]ocr2keepers.TransmitEvent, error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if rt.latest == nil {
		rt.logger.Println("encountered nil block")

		return nil, nil
	}

	events := make([]ocr2keepers.TransmitEvent, 0)

	blockKeys := rt.blockEvents.Keys(ReportTrackerBlockRange)
	for _, blockKey := range blockKeys {
		evts, _ := rt.blockEvents.Get(blockKey)

		for _, event := range evts {
			transmits, err := createPluginTransmitEvents(event, *rt.latest)
			if err != nil {
				rt.logger.Println(err)
			}

			events = append(events, transmits...)
		}
	}

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

				rt.blockEvents.Set(event.BlockNumber.String(), evt.Transmits)
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
		events []ocr2keepers.TransmitEvent
	)

	for _, result := range results {
		event := ocr2keepers.TransmitEvent{
			Type:            ocr2keepers.PerformEvent,
			TransmitBlock:   ocr2keepers.BlockNumber(chainEvent.BlockNumber.Uint64()),
			Confirmations:   new(big.Int).Sub(latest.Number, chainEvent.BlockNumber).Int64(),
			TransactionHash: chainEvent.Hash,
			UpkeepID:        result.UpkeepID,
			WorkID:          result.WorkID,
			CheckBlock:      result.Trigger.BlockNumber,
		}

		events = append(events, event)
	}

	return events, nil
}
