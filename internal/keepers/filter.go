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

	"github.com/smartcontractkit/ocr2keepers/internal/util"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

var (
	ErrKeyAlreadyAccepted = fmt.Errorf("key alredy accepted")
)

type reportCoordinator struct {
	logger         *log.Logger
	registry       types.Registry
	logs           types.PerformLogProvider
	minConfs       int
	idBlocks       *util.Cache[idBlocker] // should clear out when the next perform with this id occurs
	activeKeys     *util.Cache[bool]
	cacheCleaner   *util.IntervalCacheCleaner[bool]
	idCacheCleaner *util.IntervalCacheCleaner[idBlocker]
	starter        sync.Once
	chStop         chan struct{}
}

func newReportCoordinator(r types.Registry, s time.Duration, cacheClean time.Duration, logs types.PerformLogProvider, minConfs int, logger *log.Logger) *reportCoordinator {
	c := &reportCoordinator{
		logger:         logger,
		registry:       r,
		logs:           logs,
		minConfs:       minConfs,
		idBlocks:       util.NewCache[idBlocker](s),
		activeKeys:     util.NewCache[bool](time.Hour), // 1 hour allows the cleanup routine to clear stale data
		idCacheCleaner: util.NewIntervalCacheCleaner[idBlocker](cacheClean),
		cacheCleaner:   util.NewIntervalCacheCleaner[bool](cacheClean),
		chStop:         make(chan struct{}),
	}

	runtime.SetFinalizer(c, func(srv *reportCoordinator) { srv.stop() })

	c.start()

	return c
}

// Filter returns a filter function that removes upkeep keys that apply to this
// filter. Returns false if a key should be filtered out.
func (rc *reportCoordinator) Filter() func(types.UpkeepKey) bool {
	return func(key types.UpkeepKey) bool {
		blockKey, id, err := key.BlockKeyAndUpkeepID()
		if err != nil {
			return false
		}

		// only apply filter if key id is registered in the cache
		if bl, ok := rc.idBlocks.Get(string(id)); ok {
			// Return false if empty
			if bl.TransmitBlockNumber == nil || bl.TransmitBlockNumber.String() == "" {
				return false
			}

			isAfter, err := blockKey.After(bl.TransmitBlockNumber)
			if err != nil {
				return false
			}

			// do not filter the key out if key block is after block in cache
			return isAfter
		}

		return true
	}
}

func (rc *reportCoordinator) Accept(key types.UpkeepKey) error {
	blockKey, id, err := key.BlockKeyAndUpkeepID()
	if err != nil {
		return err
	}

	// Set the key as accepted within activeKeys
	rc.activeKeys.Set(key.String(), false, util.DefaultCacheExpiration)

	bl, ok := rc.idBlocks.Get(string(id))
	if ok {
		// TODO: What if the block number is same? (and the key was cleared out before)
		isAfter, err := bl.CheckBlockNumber.After(blockKey)
		if err != nil {
			return err
		}

		if isAfter {
			rc.logger.Printf("Higher check block already exists in idBlocks, not changing idBlocks while accepting key %s", key)
			return nil
		}
	}

	rc.idBlocks.Set(string(id), idBlocker{
		CheckBlockNumber: blockKey,
	}, util.DefaultCacheExpiration)

	return nil
}

func (rc *reportCoordinator) IsTransmissionConfirmed(key types.UpkeepKey) bool {
	// key is confirmed if it both exists and has been confirmed by the log
	// poller
	confirmed, ok := rc.activeKeys.Get(key.String())
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
			rc.logger.Printf("Skipping log in transaction %s as confirmations (%d) is less than min confirmations (%d)", l.TransactionHash, l.Confirmations, rc.minConfs)
			continue
		}

		_, id, err := l.Key.BlockKeyAndUpkeepID()
		if err != nil {
			continue
		}

		// Process log if the key hasn't been confirmed yet
		confirmed, ok := rc.activeKeys.Get(l.Key.String())
		if ok && !confirmed {
			rc.logger.Printf("Perform log found for key %s in transaction %s at block %s, with confirmations %d", l.Key, l.TransactionHash, l.TransmitBlock, l.Confirmations)

			// set state of key to indicate that the report was transmitted
			rc.activeKeys.Set(l.Key.String(), true, util.DefaultCacheExpiration)

			// if an idBlock already exists for a higher check block number, don't update it
			logCheckBlockKey, _, _ := l.Key.BlockKeyAndUpkeepID()
			bl, ok := rc.idBlocks.Get(string(id))

			if ok {
				isAfter, err := bl.CheckBlockNumber.After(logCheckBlockKey)
				if err != nil {
					continue
				}
				if isAfter {
					rc.logger.Printf("Higher check block already exists in idBlocks, not clearing idBlocks while processing perform log for key %s", l.Key)
					continue
				}
			}

			// if we detect a log, remove it from the observation filters
			// to allow it to be reported on after the block in
			// which it was transmitted
			bl.TransmitBlockNumber = l.TransmitBlock
			rc.idBlocks.Set(string(id), bl, util.DefaultCacheExpiration)

		}
	}
}

type idBlocker struct {
	CheckBlockNumber    types.BlockKey
	TransmitBlockNumber types.BlockKey
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
	rc.idCacheCleaner.Stop()
	rc.cacheCleaner.Stop()
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
