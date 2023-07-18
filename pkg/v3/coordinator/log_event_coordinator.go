package coordinator

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/config"
	"github.com/smartcontractkit/ocr2keepers/pkg/util"
)

const (
	DefaultCacheClean = time.Duration(30) * time.Second
)

//go:generate mockery --name EventProvider --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/coordinator" --case underscore --filename event_provider.generated.go
type EventProvider interface {
	Events(context.Context) ([]ocr2keepers.TransmitEvent, error)
}

type reportCoordinator struct {
	// injected dependencies
	logger *log.Logger
	events EventProvider

	// initialised by the constructor
	activeKeys        *util.Cache[bool]
	activeKeysCleaner *util.IntervalCacheCleaner[bool]
	minConfs          int

	// run state data
	running atomic.Bool
	chStop  chan struct{}
}

func NewReportCoordinator(logs EventProvider, conf config.OffchainConfig, logger *log.Logger) *reportCoordinator {
	return &reportCoordinator{
		logger:            logger,
		events:            logs,
		activeKeys:        util.NewCache[bool](time.Hour), // 1 hour allows the cleanup routine to clear stale data
		activeKeysCleaner: util.NewIntervalCacheCleaner[bool](DefaultCacheClean),
		minConfs:          conf.MinConfirmations,
		chStop:            make(chan struct{}, 1),
	}
}

func (rc *reportCoordinator) isLogEventUpkeep(upkeep ocr2keepers.ReportedUpkeep) bool {
	// Checking if Extension is a map
	extension, ok := upkeep.Trigger.Extension.(map[string]interface{})
	if !ok {
		return false
	}

	// Checking if "txHash" exists and is a string
	if _, ok := extension["txHash"].(string); !ok {
		return false
	}

	// Return true if all checks pass
	return true
}

func (rc *reportCoordinator) Accept(upkeep ocr2keepers.ReportedUpkeep) error {
	if !rc.isLogEventUpkeep(upkeep) {
		return fmt.Errorf("Upkeep is not log event based, skipping ID: %s", upkeep.ID)
	}

	if _, ok := rc.activeKeys.Get(upkeep.ID); !ok {
		rc.activeKeys.Set(upkeep.ID, false, util.DefaultCacheExpiration)
	}

	return nil
}

func (rc *reportCoordinator) IsTransmissionConfirmed(upkeep ocr2keepers.ReportedUpkeep) bool {
	// if non-exist in cache, return true
	// if exist in cache and confirmed by log poller, return true
	confirmed, ok := rc.activeKeys.Get(upkeep.ID)
	return !ok || (ok && confirmed)
}

func (rc *reportCoordinator) checkEvents(ctx context.Context) error {
	var (
		events []ocr2keepers.TransmitEvent
		err    error
	)

	events, err = rc.events.Events(ctx)
	if err != nil {
		return err
	}

	for _, evt := range events {
		if evt.Confirmations < int64(rc.minConfs) {
			rc.logger.Printf("Skipping perform log in transaction %s as confirmations (%d) is less than min confirmations (%d)", evt.TransactionHash, evt.Confirmations, rc.minConfs)
			continue
		}
		switch evt.Type {
		case ocr2keepers.PerformEvent, ocr2keepers.StaleReportEvent:
			rc.performEvent(evt)
		case ocr2keepers.ReorgReportEvent, ocr2keepers.InsufficientFundsReportEvent:
			rc.activeKeys.Delete(evt.ID)
			// TODO: push to recovery flow
		}
	}

	return nil
}

func (rc *reportCoordinator) performEvent(evt ocr2keepers.TransmitEvent) {
	rc.activeKeys.Set(evt.ID, true, util.DefaultCacheExpiration)
}

// isPending returns true if a key should be filtered out.
func (rc *reportCoordinator) isPending(payload ocr2keepers.UpkeepPayload) bool {
	if _, ok := rc.activeKeys.Get(payload.ID); ok {
		// If the payload already exists, return true
		return true
	}
	return false
}

func (rc *reportCoordinator) PreProcess(_ context.Context, payloads []ocr2keepers.UpkeepPayload) ([]ocr2keepers.UpkeepPayload, error) {
	var filteredPayloads []ocr2keepers.UpkeepPayload

	for _, payload := range payloads {
		if !rc.isPending(payload) {
			// If the payload is not pending, add it to the filteredPayloads slice
			filteredPayloads = append(filteredPayloads, payload)
		}
	}

	return filteredPayloads, nil
}

func (rc *reportCoordinator) run() {
	cadence := time.Second
	timer := time.NewTimer(cadence)

	for {
		select {
		case <-timer.C:
			startTime := time.Now()

			if err := rc.checkEvents(context.Background()); err != nil {
				rc.logger.Printf("failed to check perform and stale report logs: %s", err)
			}

			// attempt to adhere to a cadence of at least every second
			// a slow DB will cause the cadence to increase. these cases are logged
			diff := time.Since(startTime)
			if diff > cadence {
				rc.logger.Printf("log poll took %dms to complete; expected cadence is %dms; check database indexes and other performance improvements", diff/time.Millisecond, cadence/time.Millisecond)
				// start again immediately
				timer.Reset(time.Microsecond)
			} else {
				// wait the difference between the cadence and the time taken
				timer.Reset(cadence - diff)
			}
		case <-rc.chStop:
			timer.Stop()
			return
		}
	}
}

// Start starts all subprocesses
func (rc *reportCoordinator) Start(_ context.Context) error {
	if rc.running.Load() {
		return fmt.Errorf("process already running")
	}

	go rc.activeKeysCleaner.Run(rc.activeKeys)

	rc.running.Store(true)
	rc.run()

	return nil
}

// Close terminates all subprocesses
func (rc *reportCoordinator) Close() error {
	if !rc.running.Load() {
		return fmt.Errorf("process not running")
	}

	rc.activeKeysCleaner.Stop()
	rc.chStop <- struct{}{}
	rc.running.Store(false)

	return nil
}
