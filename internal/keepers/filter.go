package keepers

import (
	"context"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type reportCoordinator struct {
	logger       *log.Logger
	registry     types.Registry
	logs         types.PerformLogProvider
	idBlocks     *cache[bool] // should clear out when the next perform with this id occurs
	activeKeys   *cache[bool] // should clear when next perform with blocknumber and id occurs
	cacheCleaner *intervalCacheCleaner[bool]
	starter      sync.Once
	chStop       chan struct{}
}

func newReportCoordinator(r types.Registry, s time.Duration, cacheClean time.Duration, logs types.PerformLogProvider, logger *log.Logger) *reportCoordinator {
	c := &reportCoordinator{
		logger:     logger,
		registry:   r,
		logs:       logs,
		idBlocks:   newCache[bool](s),
		activeKeys: newCache[bool](time.Hour), // 1 hour allows the cleanup routine to clear stale data
		chStop:     make(chan struct{}),
	}

	cl := &intervalCacheCleaner[bool]{
		Interval: cacheClean,
		stop:     make(chan struct{}),
	}

	c.cacheCleaner = cl

	runtime.SetFinalizer(c, func(srv *reportCoordinator) { srv.stop() })

	c.start()

	return c
}

// Filter returns a filter function that removes upkeep keys that apply to this
// filter
func (rc *reportCoordinator) Filter() func(types.UpkeepKey) bool {
	return func(key types.UpkeepKey) bool {
		id, err := rc.registry.IdentifierFromKey(key)
		if err != nil {
			return false
		}

		if _, ok := rc.idBlocks.Get(string(id)); ok {
			return false
		}

		return true
	}
}

func (rc *reportCoordinator) Accept(key types.UpkeepKey) error {
	id, err := rc.registry.IdentifierFromKey(key)
	if err != nil {
		return err
	}

	rc.idBlocks.Set(string(id), true, defaultExpiration)
	rc.activeKeys.Set(string(key), false, defaultExpiration)

	return nil
}

func (rc *reportCoordinator) IsTransmitting(key types.UpkeepKey) bool {
	// key is assumed to be transmitted if it doesn't exist in cache or if it
	// is no longer waiting for a transaction to be transmitted
	inLogs, ok := rc.activeKeys.Get(string(key))
	return ok && inLogs
}

func (rc *reportCoordinator) start() {
	rc.starter.Do(func() {
		go rc.run()
		go rc.cacheCleaner.Run(rc.idBlocks)
		go rc.cacheCleaner.Run(rc.activeKeys)
	})
}

func (rc *reportCoordinator) stop() {
	rc.chStop <- struct{}{}
	rc.cacheCleaner.stop <- struct{}{}
	rc.cacheCleaner.stop <- struct{}{}
}

func (rc *reportCoordinator) run() {
	// TODO: handle panics by restarting this process

	cadence := time.Second
	timer := time.NewTimer(cadence)

	for {

		select {
		case <-timer.C:
			startTime := time.Now()
			logs, _ := rc.logs.PerformLogs(context.Background())

			// log entries indicate that a perform exists on chain in some
			// capacity. the existance of an entry means that the transaction
			// was broadcast by at least one node. reorgs can still happen
			// causing performs to vanish or get moved to a later block.
			//
			// in the case of reorgs causing a perform to be dropped from the
			// chain, the key is kept in cache even after first detection to
			// ensure no other nodes attempt to transmit again.
			for _, log := range logs {
				id, err := rc.registry.IdentifierFromKey(log.Key)
				if err != nil {
					continue
				}

				// if we detect a log, remove it from the observation filters
				// to allow it to be reported on again
				rc.idBlocks.Delete(string(id))

				// set the active key value to true to indicate that
				// we now have a transaction identified and no new
				// attempts at a transaction should be made
				rc.activeKeys.Set(string(log.Key), true, defaultExpiration)
			}

			// attempt to ahere to a cadence of at least every second
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
