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
	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

var (
	ErrKeyAlreadyAccepted = fmt.Errorf("key alredy accepted")
)

type reportCoordinator struct {
	logger                *log.Logger
	registry              types.Registry
	logs                  types.PerformLogProvider
	minConfs              int
	idBlocks              *util.Cache[idBlocker] // should clear out when the next perform with this id occurs
	activeKeys            *util.Cache[bool]
	reorgLogs             *util.Cache[bool] // Stores the processed reorg log hashes
	cacheCleaner          *util.IntervalCacheCleaner[bool]
	idCacheCleaner        *util.IntervalCacheCleaner[idBlocker]
	reorgLogsCacheCleaner *util.IntervalCacheCleaner[bool]
	starter               sync.Once
	chStop                chan struct{}
}

func newReportCoordinator(r types.Registry, s time.Duration, cacheClean time.Duration, logs types.PerformLogProvider, minConfs int, logger *log.Logger) *reportCoordinator {
	c := &reportCoordinator{
		logger:                logger,
		registry:              r,
		logs:                  logs,
		minConfs:              minConfs,
		idBlocks:              util.NewCache[idBlocker](s),
		activeKeys:            util.NewCache[bool](time.Hour), // 1 hour allows the cleanup routine to clear stale data
		reorgLogs:             util.NewCache[bool](time.Hour),
		idCacheCleaner:        util.NewIntervalCacheCleaner[idBlocker](cacheClean),
		cacheCleaner:          util.NewIntervalCacheCleaner[bool](cacheClean),
		reorgLogsCacheCleaner: util.NewIntervalCacheCleaner[bool](cacheClean),
		chStop:                make(chan struct{}),
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

	// Set idBlocks with the key as checkBlockNumber and empty as TransmitBlockNumber
	// Empty TransmitBlockNumber filters the upkeep indefinitely (until it is updated by performLog or after performLockoutWindow)
	rc.updateIdBlock(string(id), idBlocker{
		CheckBlockNumber: blockKey,
	})

	return nil
}

func (rc *reportCoordinator) IsTransmissionConfirmed(key types.UpkeepKey) bool {
	// key is confirmed if it both exists and has been confirmed by the log
	// poller
	confirmed, ok := rc.activeKeys.Get(key.String())
	return !ok || (ok && confirmed)
}

func (rc *reportCoordinator) checkLogs() {
	performLogs, _ := rc.logs.PerformLogs(context.Background())
	// Perform log entries indicate that a perform exists on chain in some
	// capacity. the existance of an entry means that the transaction
	// was broadcast by at least one node. reorgs can still happen
	// causing performs to vanish or get moved to a later block. Higher minConfirmations
	// setting reduces the chances of this happening.
	//
	// We do two things upon receiving a perform log
	// - Mark the upkeep key responsible for the perform as 'transmitted', so that this node does not
	//   waste gas trying to transmit the same report again
	// - Unblock the upkeep from idBlocks so that it can be observed and reported on again.
	for _, l := range performLogs {
		if l.Confirmations < int64(rc.minConfs) {
			rc.logger.Printf("Skipping perform log in transaction %s as confirmations (%d) is less than min confirmations (%d)", l.TransactionHash, l.Confirmations, rc.minConfs)
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

			logCheckBlockKey, _, _ := l.Key.BlockKeyAndUpkeepID()
			rc.updateIdBlock(string(id), idBlocker{
				CheckBlockNumber:    logCheckBlockKey,
				TransmitBlockNumber: l.TransmitBlock, // Removes the id from filters from higher blocks
			})
		}
	}

	reorgLogs, _ := rc.logs.ReorgLogs(context.Background())
	// It can happen that in between the time the report is generated and it gets
	// confirmed on chain there's a reorg. In such cases the upkeep is not performed
	// as it was checked on a different chain, and the contract emits a ReorgedUpkeepReport log
	// instead of UpkeepPerformed log.
	//
	// For a ReorgedUpkeep log we do not have the exact key which generated this log. Hence we
	// are not able to mark the key responsible as transmitted which will result in some wasted
	// gas if this node tries to transmit it again, however we prioritise the upkeep performance
	// and clear the idBlocks for this upkeep.
	for _, l := range reorgLogs {
		if l.Confirmations < int64(rc.minConfs) {
			rc.logger.Printf("Skipping reorg log in transaction %s as confirmations (%d) is less than min confirmations (%d)", l.TransactionHash, l.Confirmations, rc.minConfs)
			continue
		}

		// If a reorg log was processed already, do not reprocess it. TransmitBlock + UpkeepId is used to identify the log
		logKey := chain.NewUpkeepKeyFromBlockAndID(l.TransmitBlock, l.UpkeepId).String()
		_, ok := rc.reorgLogs.Get(logKey)
		if !ok {
			rc.logger.Printf("Reorg log found for upkeep %s in transaction %s at block %s, with confirmations %d", l.UpkeepId, l.TransactionHash, l.TransmitBlock, l.Confirmations)
			rc.reorgLogs.Set(logKey, true, util.DefaultCacheExpiration)

			// As we do not have the actual checkBlockNumber which generated this reorg log, use transmitBlockNumber
			rc.updateIdBlock(string(l.UpkeepId), idBlocker{
				CheckBlockNumber:    l.TransmitBlock,
				TransmitBlockNumber: l.TransmitBlock, // Removes the id from filters from higher blocks
			})
		}
	}
}

// This function tries to update idBlock for a given key to val. If no idBlock existed for this key
// then it's just updated with val, however if it existed before then it is only updated if checkBlockNumber
// is set higher, or checkBlockNumber is the same and transmitBlockNumber is higher.
//
// For a sequence of updates, updateIdBlock can be called in any order on different nodes, but by
// maintaining this invariant it results in an eventually consistent value across nodes.
func (rc *reportCoordinator) updateIdBlock(key string, val idBlocker) {
	idBlock, ok := rc.idBlocks.Get(key)
	if !ok {
		// No value before, simply set it and return
		rc.logger.Printf("updateIdBlock for key %s: value updated to %+v", key, val)
		rc.idBlocks.Set(key, val, util.DefaultCacheExpiration)
		return
	}

	// If idBlock.checkBlockNumber is strictly after val.checkBlockNumber, nothing to update, simply return
	isAfter, err := idBlock.CheckBlockNumber.After(val.CheckBlockNumber)
	if err != nil {
		// No updates in case of error
		return
	}
	if isAfter {
		rc.logger.Printf("updateIdBlock for key %s: Higher check block already exists in idBlocks (%+v), not setting new val (%+v)", key, idBlock, val)
		return
	}

	// If val.checkBlockNumber is strictly after idBlock.checkBlockNumber, simply update it
	isAfter, err = val.CheckBlockNumber.After(idBlock.CheckBlockNumber)
	if err != nil {
		// No updates in case of error
		return
	}
	if isAfter {
		rc.logger.Printf("updateIdBlock for key %s: value updated to %+v", key, val)
		rc.idBlocks.Set(key, val, util.DefaultCacheExpiration)
		return
	}

	// Now val.checkBlockNumber == idBlock.checkBlockNumber, update if transmitBlockNumber has increased
	// Note: val.TransmitBlockNumber can be nil or empty, in which case it is considered lower and not updated
	// We do this separately so that after is not called on the key
	if idBlock.TransmitBlockNumber == nil || idBlock.TransmitBlockNumber.String() == "" {
		rc.logger.Printf("updateIdBlock for key %s: Higher transmit block already exists in idBlocks (%+v), not setting new val (%+v)", key, idBlock, val)
		return
	}
	isAfter, err = val.TransmitBlockNumber.After(idBlock.TransmitBlockNumber)
	if err != nil {
		// No updates in case of error
		return
	}
	if isAfter {
		rc.logger.Printf("updateIdBlock for key %s: value updated to %+v", key, val)
		rc.idBlocks.Set(key, val, util.DefaultCacheExpiration)
		return
	}

	rc.logger.Printf("updateIdBlock for key %s: Higher transmit block already exists in idBlocks (%+v), not setting new val (%+v)", key, idBlock, val)
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
		go rc.reorgLogsCacheCleaner.Run(rc.reorgLogs)
	})
}

func (rc *reportCoordinator) stop() {
	rc.chStop <- struct{}{}
	rc.idCacheCleaner.Stop()
	rc.cacheCleaner.Stop()
	rc.reorgLogsCacheCleaner.Stop()
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
