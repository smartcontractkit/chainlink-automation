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
	IndefiniteBlockingKey = chain.BlockKey("18446744073709551616") // Higher than possible block numbers (uint64), used to block keys indefintely
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

	// If a key is already active then don't update filters, but also not throw errors as
	// there might be other keys in the same report which can get accepted
	_, ok := rc.activeKeys.Get(key.String())
	if !ok {
		// Set the key as accepted within activeKeys
		rc.activeKeys.Set(key.String(), false, util.DefaultCacheExpiration)

		// Set idBlocks with the key as checkBlockNumber and IndefiniteBlockingKey as TransmitBlockNumber
		rc.updateIdBlock(string(id), idBlocker{
			CheckBlockNumber:    blockKey,
			TransmitBlockNumber: IndefiniteBlockingKey,
		})
	}

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
	// causing performs to get moved to a later block or change to reorg logs.
	// Higher minConfirmations setting reduces the chances of this happening.
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

		logCheckBlockKey, _, _ := l.Key.BlockKeyAndUpkeepID()
		// Process log if the key hasn't been confirmed yet
		confirmed, ok := rc.activeKeys.Get(l.Key.String())
		if ok && !confirmed {
			rc.logger.Printf("Perform log found for key %s in transaction %s at block %s, with confirmations %d", l.Key, l.TransactionHash, l.TransmitBlock, l.Confirmations)

			// set state of key to indicate that the report was transmitted
			rc.activeKeys.Set(l.Key.String(), true, util.DefaultCacheExpiration)

			rc.updateIdBlock(string(id), idBlocker{
				CheckBlockNumber:    logCheckBlockKey,
				TransmitBlockNumber: l.TransmitBlock, // Removes the id from filters from higher blocks
			})
		}

		if ok && confirmed {
			// This can happen if we get a perform log for the same key again on a newer block in case of reorgs
			// In this case, no change to activeKeys is needed, but idBlocks is updated to the newer BlockNumber
			idBlock, ok := rc.idBlocks.Get(string(id))
			if ok && idBlock.CheckBlockNumber.String() == logCheckBlockKey.String() &&
				idBlock.TransmitBlockNumber.String() != l.TransmitBlock.String() {

				rc.logger.Printf("Got a re-orged perform log for key %s in transaction %s at block %s, with confirmations %d", l.Key, l.TransactionHash, l.TransmitBlock, l.Confirmations)

				rc.updateIdBlock(string(id), idBlocker{
					CheckBlockNumber:    logCheckBlockKey,
					TransmitBlockNumber: l.TransmitBlock,
				})
			}
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

			rc.updateIdBlock(string(l.UpkeepId), idBlocker{
				CheckBlockNumber: l.TransmitBlock, // As we do not have the actual checkBlockNumber which generated this
				//reorg log, use transmitBlockNumber to override all previous checkBlockNumbers
				TransmitBlockNumber: l.TransmitBlock, // Removes the id from filters from higher blocks
			})
		}
	}
}

type idBlocker struct {
	CheckBlockNumber    types.BlockKey
	TransmitBlockNumber types.BlockKey
}

// idBlock should only be updated if checkBlockNumber is set higher
// or checkBlockNumber is the same and transmitBlockNumber is higher
// (with a special case for IndefiniteBlockingKey).
//
// For a sequence of updates, updateIdBlock can be called in any order
// on different nodes, but by maintaining this invariant it results in
// an eventually consistent value across nodes.
func (b idBlocker) ShouldUpdate(val idBlocker) (bool, error) {
	isAfter, err := val.CheckBlockNumber.After(b.CheckBlockNumber)
	if err != nil {
		return false, err
	}
	if isAfter {
		// val has higher checkBlockNumber
		return true, nil
	}

	isAfter, err = b.CheckBlockNumber.After(val.CheckBlockNumber)
	if err != nil {
		return false, err
	}
	if isAfter {
		// b has higher checkBlockNumber
		return false, nil
	}

	// Now the checkBlockNumber should be same
	// If idBlock has an IndefiniteBlockingKey, then update
	if b.TransmitBlockNumber.String() == IndefiniteBlockingKey.String() {
		return true, nil
	}

	// return true if val.TransmitBlockNumber is higher
	return val.TransmitBlockNumber.After(b.TransmitBlockNumber)
}

func (rc *reportCoordinator) updateIdBlock(key string, val idBlocker) {
	idBlock, ok := rc.idBlocks.Get(key)
	if ok {
		shouldUpdate, err := idBlock.ShouldUpdate(val)
		if err != nil {
			// Don't update on errors
			return
		}
		if !shouldUpdate {
			rc.logger.Printf("updateIdBlock for key %s: Not updating idBlocks (%+v) to new val (%+v)", key, idBlock, val)
			return
		}
	}

	rc.logger.Printf("updateIdBlock for key %s: value updated to %+v", key, val)
	rc.idBlocks.Set(key, val, util.DefaultCacheExpiration)
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
