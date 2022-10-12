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

// Add adds the provided upkeep to the filter.
func (rc *reportCoordinator) Add(key types.UpkeepKey) error {
	id, err := rc.registry.IdentifierFromKey(key)
	if err != nil {
		return err
	}

	rc.idBlocks.Set(string(id), true, defaultExpiration)

	return nil
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

func (rc *reportCoordinator) Accept(key types.UpkeepKey) {
	rc.activeKeys.Set(string(key), false, defaultExpiration)
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
			keys := rc.activeKeys.Keys()
			logs, _ := rc.logs.PerformLogs(context.Background())

			// log entries indicate that a perform exists on chain in some
			// capacity. the existance of an entry means that the transaction
			// was broadcast by at least one node. reorgs can still happen
			// causing performs to vanish or get moved to a later block.
			//
			// in the case of reorgs causing a perform to be dropped from the
			// chain, total confirmations are checked to allow other nodes
			// to retry a transaction based on the OCR transmit cadence.
			logLookup := make(map[string]types.PerformLog)
			var minConfirmations int64 = 20
			for _, log := range logs {
				logLookup[string(log.Key)] = log
				if _, ok := rc.activeKeys.Get(string(log.Key)); ok {
					if log.Confirmations < minConfirmations {
						// set the active key value to false to indicate that
						// we now have a transaction identified and no new
						// attempts at a transaction should be made
						rc.activeKeys.Set(string(log.Key), true, defaultExpiration)
					} else {
						// the transaction is complete so the key can be deleted
						rc.activeKeys.Delete(string(log.Key))
					}
				}
			}

			// perform logs can potentially come and go. for all keys, if a
			// perform log is not detected, set the active keys value to true to
			// indicate that we are waiting for a transaction log and any node
			// should attempt to transmit.
			for _, key := range keys {
				if _, ok := logLookup[key]; !ok {
					rc.activeKeys.Set(key, false, defaultExpiration)
				}
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
