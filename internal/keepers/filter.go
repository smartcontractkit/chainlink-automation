/*
The report coordinator provides 3 main functions:
Filter
Accept
IsTransmissionConfirmed

This has 2 purposes:
When an id is accepted using the Accept function, the upkeep id should be
included in the Filter function. This allows an upkeep id to be filtered out
of a list of upkeep keys.

When an upkeep key is accepted using the Accept function, the upkeep key will
return false on IsTransmissionConfirmed until a perform log is identified with
the same key. This allows a coordinated effort on transmit fallbacks.
*/
package keepers

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

var (
	ErrKeyAlreadySet = fmt.Errorf("key alredy set")
)

type reportCoordinator struct {
	logger         *log.Logger
	registry       types.Registry
	logs           types.PerformLogProvider
	minConfs       int
	idBlocks       *cache[types.BlockKey]
	activeKeys     *cache[bool]
	idCacheCleaner *intervalCacheCleaner[types.BlockKey]
	cacheCleaner   *intervalCacheCleaner[bool]
	starter        sync.Once
	chStop         chan struct{}
}

func newReportCoordinator(r types.Registry, s time.Duration, cacheClean time.Duration, logs types.PerformLogProvider, minConfs int, logger *log.Logger) *reportCoordinator {
	c := &reportCoordinator{
		logger:     logger,
		registry:   r,
		logs:       logs,
		minConfs:   minConfs,
		idBlocks:   newCache[types.BlockKey](s),
		activeKeys: newCache[bool](time.Hour), // 1 hour allows the cleanup routine to clear stale data
		chStop:     make(chan struct{}),
		idCacheCleaner: &intervalCacheCleaner[types.BlockKey]{
			Interval: cacheClean,
			stop:     make(chan struct{}),
		},
		cacheCleaner: &intervalCacheCleaner[bool]{
			Interval: cacheClean,
			stop:     make(chan struct{}),
		},
	}

	runtime.SetFinalizer(c, func(srv *reportCoordinator) { srv.stop() })

	c.start()

	return c
}

// Filter returns a filter function that removes upkeep keys that apply to this
// filter. Returns false if a key should be filtered out.
func (rc *reportCoordinator) Filter() func(types.UpkeepKey) bool {
	return func(key types.UpkeepKey) bool {
		id, err := rc.registry.IdentifierFromKey(key)
		if err != nil {
			// filter on error
			return false
		}

		// only apply filter if key id is registered in the cache
		if bl, ok := rc.idBlocks.Get(string(id)); ok {
			// only apply filter if key block is after block in cache
			if len(bl) > 0 && string(key) >= string(bl) {
				return true
			}

			return false
		}

		return true
	}
}

func (rc *reportCoordinator) Accept(key types.UpkeepKey) error {
	_, ok := rc.activeKeys.Get(string(key))
	if ok {
		return fmt.Errorf("%w: %s", ErrKeyAlreadySet, key)
	}

	id, err := rc.registry.IdentifierFromKey(key)
	if err != nil {
		return err
	}

	rc.idBlocks.Set(string(id), types.BlockKey([]byte{}), defaultExpiration)
	rc.activeKeys.Set(string(key), false, defaultExpiration)

	return nil
}

func (rc *reportCoordinator) IsTransmissionConfirmed(key types.UpkeepKey) bool {
	// key is confirmed if it both exists and has been confirmed by the log
	// poller
	confirmed, ok := rc.activeKeys.Get(string(key))
	return !ok || (ok && confirmed)
}

func (rc *reportCoordinator) checkLogs() {
	logs, _ := rc.logs.PerformLogs(context.Background())

	// log entries indicate that a perform exists on chain in some
	// capacity. the existance of an entry means that the transaction
	// was broadcast by at least one node. reorgs can still happen
	// causing performs to vanish or get moved to a later block.
	//
	// in the case of reorgs causing a perform to be dropped from the
	// chain, the key is kept in cache even after first detection to
	// ensure no other nodes attempt to transmit again.
	for _, l := range logs {
		if l.Confirmations < int64(rc.minConfs) {
			continue
		}

		id, err := rc.registry.IdentifierFromKey(l.Key)
		if err != nil {
			continue
		}

		// Process log if the key hasn't been confirmed yet
		confirmed, ok := rc.activeKeys.Get(string(l.Key))
		if ok && !confirmed {
			// if we detect a log, remove it from the observation filters
			// to allow it to be reported on again at or after the block in
			// which it was transmitted
			rc.idBlocks.Set(string(id), l.TransmitBlock, defaultExpiration)

			// set state of key to indicate that the report was transmitted
			// setting a key in this way also blocks it in Accept even if
			// Accept was never called for on a single node for this key
			rc.activeKeys.Set(string(l.Key), true, defaultExpiration)
		}
	}
}

func (rc *reportCoordinator) start() {
	rc.starter.Do(func() {
		go rc.run()
		go rc.idCacheCleaner.Run(rc.idBlocks)
		go rc.cacheCleaner.Run(rc.activeKeys)
	})
}

func (rc *reportCoordinator) stop() {
	rc.chStop <- struct{}{}
	rc.idCacheCleaner.stop <- struct{}{}
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
			rc.checkLogs()

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
