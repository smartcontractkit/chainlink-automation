package ocr

import (
	"context"
	"log"
	"runtime"
	"sync"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator/chain"
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
	rt.mu.Lock()
	defer rt.mu.Unlock()

	events := make([]ocr2keepers.TransmitEvent, 0, len(rt.events))
	for _, event := range rt.events {
		events = append(events, createPluginTransmitEvents(event)...)
	}

	rt.read = append(rt.read, rt.events...)
	rt.events = []chain.TransmitEvent{}

	return events, nil
}

func (rt *ReportTracker) run() {
	chEvents := rt.listener.Subscribe(chain.PerformUpkeepChannel)

	for {
		select {
		case event := <-chEvents:
			switch evt := event.Event.(type) {
			case chain.PerformUpkeepTransaction:
				rt.saveTransmits(evt.Transmits)
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

// TODO: complete this
func createPluginTransmitEvents(chain.TransmitEvent) []ocr2keepers.TransmitEvent {

	return nil
}
