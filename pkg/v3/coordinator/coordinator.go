package coordinator

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/util"
)

const (
	DefaultCacheClean           = time.Duration(30) * time.Second
	DefaultMinimumConfirmations = 1
)

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

func NewReportCoordinator(logs EventProvider, logger *log.Logger) *reportCoordinator {
	return &reportCoordinator{
		logger:            logger,
		events:            logs,
		activeKeys:        util.NewCache[bool](time.Hour), // 1 hour allows the cleanup routine to clear stale data
		activeKeysCleaner: util.NewIntervalCacheCleaner[bool](DefaultCacheClean),
		minConfs:          DefaultMinimumConfirmations,
		chStop:            make(chan struct{}, 1),
	}
}

func (rc *reportCoordinator) IsTransmissionConfirmed(upkeep ocr2keepers.ReportedUpkeep) bool {
	// key is confirmed if it both exists and has been confirmed by the log
	// poller
	confirmed, ok := rc.activeKeys.Get(upkeep.ID)
	return !ok || (ok && confirmed)
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

		rc.performEvent(evt)
	}

	return nil
}

func (rc *reportCoordinator) performEvent(evt ocr2keepers.TransmitEvent) {
	if confirmed, ok := rc.activeKeys.Get(evt.ID); ok {
		if !confirmed {
			// Process log if the key hasn't been confirmed yet
			rc.logger.Printf("Perform log found for key %s in transaction %s at block %s, with confirmations %d", evt.ID, evt.TransactionHash, evt.TransmitBlock, evt.Confirmations)

			// set state of key to indicate that the report was transmitted
			rc.activeKeys.Set(evt.ID, true, util.DefaultCacheExpiration)
		}

		/*
			if confirmed {
				// This can happen if we get a perform log for the same key again on a newer block in case of reorgs
				// In this case, no change to activeKeys is needed, but idBlocks is updated to the newer BlockNumber
			}
		*/
	} else {
		rc.activeKeys.Set(evt.ID, true, util.DefaultCacheExpiration)
	}
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
