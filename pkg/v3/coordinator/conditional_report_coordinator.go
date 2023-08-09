/*
A coordinator provides the ability to filter upkeeps based on some type of
in-flight status.

The report coordinator provides 3 main functions:
IsPending
Accept
IsTransmissionConfirmed

This has 2 purposes:
When an id is accepted using the Accept function, the upkeep id should be
indicated as pending in the IsPending function. This allows an upkeep id to be
filtered out of a list of upkeep keys.

When an upkeep key is accepted using the Accept function, the upkeep key will
return false on IsTransmissionConfirmed until a perform log is identified with
the same key. This allows a coordinated effort on transmit fallbacks.

The report coordinator relies on two log types:
PerformLog - this log type indicates that an upkeep was completed
StaleReportLog - this log type indicates that an upkeep failed and can be
attempted again at a later block height
*/
package coordinator

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync/atomic"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/util"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

var (
	DefaultLockoutWindow  = time.Duration(20) * time.Minute
	ErrKeyAlreadyAccepted = fmt.Errorf("key already accepted")
	IndefiniteBlockingKey = ocr2keepers.BlockNumber(math.MaxUint64) // Higher than possible block numbers (uint64), used to block keys indefintely
	cadence               = time.Second
)

//go:generate mockery --name Encoder --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v2/coordinator" --case underscore --filename encoder.generated.go
type Encoder interface {
	// After a is after b
	After(ocr2keepers.BlockNumber, ocr2keepers.BlockNumber) (bool, error)
	// Increment
	Increment(ocr2keepers.BlockNumber) (ocr2keepers.BlockNumber, error)
}

type conditionalReportCoordinator struct {
	// injected dependencies
	logger *log.Logger
	events ocr2keepers.TransmitEventProvider

	encoder Encoder

	// initialised by the constructor
	idBlocks       *util.Cache[idBlocker] // should clear out when the next perform with this id occurs
	activeKeys     *util.Cache[bool]
	cacheCleaner   *util.IntervalCacheCleaner[bool]
	idCacheCleaner *util.IntervalCacheCleaner[idBlocker]

	// configurations
	minConfs int

	// run state data
	running atomic.Bool
	chStop  chan struct{}
	chDone  chan struct{}
}

// NewConditionalReportCoordinator provides a new conditional report coordinator. The coordinator
// should be started before use.
func NewConditionalReportCoordinator(
	events ocr2keepers.TransmitEventProvider,
	minConfs int,
	logger *log.Logger,
	encoder Encoder,
) *conditionalReportCoordinator {
	c := &conditionalReportCoordinator{
		logger:         logger,
		events:         events,
		minConfs:       minConfs,
		idBlocks:       util.NewCache[idBlocker](DefaultLockoutWindow),
		activeKeys:     util.NewCache[bool](time.Hour), // 1 hour allows the cleanup routine to clear stale data
		idCacheCleaner: util.NewIntervalCacheCleaner[idBlocker](DefaultCacheClean),
		cacheCleaner:   util.NewIntervalCacheCleaner[bool](DefaultCacheClean),
		chStop:         make(chan struct{}, 1),
		encoder:        encoder,
		chDone:         make(chan struct{}, 1),
	}

	return c
}

// isPending returns true if a key should be filtered out.
func (rc *conditionalReportCoordinator) isPending(key ocr2keepers.UpkeepPayload) bool {
	blockKey := ocr2keepers.BlockKey{
		Number: ocr2keepers.BlockNumber(key.Trigger.BlockNumber),
	}

	// only apply filter if key id is registered in the cache
	if bl, ok := rc.idBlocks.Get(key.UpkeepID.String()); ok {
		isAfter, err := rc.encoder.After(blockKey.Number, bl.TransmitBlockNumber)
		if err != nil {
			return true
		}

		// do not filter the key out if key block is after block in cache
		return !isAfter
	}

	return false
}

// Accept sets the pending status for a key
func (rc *conditionalReportCoordinator) Accept(key ocr2keepers.ReportedUpkeep) error {
	blockKey := ocr2keepers.BlockKey{
		Number: ocr2keepers.BlockNumber(key.Trigger.BlockNumber),
	}
	// If a key is already active then don't update filters, but also not throw errors as
	// there might be other keys in the same report which can get accepted
	if _, ok := rc.activeKeys.Get(key.UpkeepID.String()); !ok {
		// Set the key as accepted within activeKeys
		rc.activeKeys.Set(key.UpkeepID.String(), false, util.DefaultCacheExpiration)

		// Set idBlocks with the key as checkBlockNumber and IndefiniteBlockingKey as TransmitBlockNumber
		rc.updateIdBlock(key.UpkeepID.String(), idBlocker{
			CheckBlockNumber:    blockKey.Number,
			TransmitBlockNumber: IndefiniteBlockingKey,
		})
	}

	return nil
}

// IsTransmissionConfirmed returns whether the upkeep was successfully
// completed or not
func (rc *conditionalReportCoordinator) IsTransmissionConfirmed(key ocr2keepers.UpkeepPayload) bool {
	// key is confirmed if it both exists and has been confirmed by the log
	// poller
	confirmed, ok := rc.activeKeys.Get(key.UpkeepID.String())
	return !ok || (ok && confirmed)
}

func (rc *conditionalReportCoordinator) PreProcess(_ context.Context, payloads []ocr2keepers.UpkeepPayload) ([]ocr2keepers.UpkeepPayload, error) {
	var filteredPayloads []ocr2keepers.UpkeepPayload

	for _, payload := range payloads {
		if !rc.isPending(payload) {
			// If the payload is not pending, add it to the filteredPayloads slice
			filteredPayloads = append(filteredPayloads, payload)
		}
	}

	return filteredPayloads, nil
}

func (rc *conditionalReportCoordinator) checkEvents(ctx context.Context) error {
	var (
		events []ocr2keepers.TransmitEvent
		err    error
	)

	events, err = rc.events.GetLatestEvents(ctx)
	if err != nil {
		return err
	}
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
	for _, evt := range events {
		if evt.Confirmations < int64(rc.minConfs) {
			rc.logger.Printf("Skipping transmit event in transaction %s as confirmations (%d) is less than min confirmations (%d)", evt.TransactionHash, evt.Confirmations, rc.minConfs)
			continue
		}

		nextKey := evt.TransmitBlock

		if evt.Type == ocr2keepers.StaleReportEvent {
			nextKey, err = rc.encoder.Increment(evt.CheckBlock)
			if err != nil {
				continue
			}
		}

		if confirmed, ok := rc.activeKeys.Get(evt.UpkeepID.String()); ok {
			if confirmed {
				// This can happen if we get a stale log for the same key again on a newer block or in case
				// the key was unblocked due to a performLog which later got reorged into a stale log
				idBlock, ok := rc.idBlocks.Get(evt.UpkeepID.String())
				if ok && idBlock.CheckBlockNumber == evt.CheckBlock &&
					idBlock.TransmitBlockNumber != nextKey {

					rc.logger.Printf("Got a stale event for previously accepted key %s in transaction %s at block %d, with confirmations %d", evt.WorkID, evt.TransactionHash, evt.TransmitBlock, evt.Confirmations)

					rc.updateIdBlock(evt.UpkeepID.String(), idBlocker{
						CheckBlockNumber:    evt.CheckBlock,
						TransmitBlockNumber: nextKey,
					})
				}
			} else {
				// Process log if the key hasn't been confirmed yet
				rc.logger.Printf("Stale event found for key %s in transaction %s at block %d, with confirmations %d", evt.WorkID, evt.TransactionHash, evt.TransmitBlock, evt.Confirmations)
				// set state of key to indicate that the report was transmitted
				rc.activeKeys.Set(evt.UpkeepID.String(), true, util.DefaultCacheExpiration)

				rc.updateIdBlock(evt.UpkeepID.String(), idBlocker{
					CheckBlockNumber:    evt.CheckBlock,
					TransmitBlockNumber: nextKey,
				})
			}
		}
	}

	return err
}

type idBlocker struct {
	CheckBlockNumber    ocr2keepers.BlockNumber
	TransmitBlockNumber ocr2keepers.BlockNumber
}

// idBlock should only be updated if checkBlockNumber is set higher
// or checkBlockNumber is the same and transmitBlockNumber is higher
// (with a special case for IndefiniteBlockingKey being considered lowest).
//
// For a sequence of updates, updateIdBlock can be called in any order
// on different nodes, but by maintaining this invariant it results in
// an eventually consistent value across nodes.
func (b idBlocker) shouldUpdate(val idBlocker, e Encoder) (bool, error) {
	isAfter, err := e.After(val.CheckBlockNumber, b.CheckBlockNumber)
	if err != nil {
		return false, err
	}

	if isAfter {
		// val has higher checkBlockNumber
		return true, nil
	}

	isAfter, err = e.After(b.CheckBlockNumber, val.CheckBlockNumber)
	if err != nil {
		return false, err
	}

	if isAfter {
		// b has higher checkBlockNumber
		return false, nil
	}

	// Now the checkBlockNumber should be same

	// If b has an IndefiniteBlockingKey, then update
	if b.TransmitBlockNumber == IndefiniteBlockingKey {
		return true, nil
	}

	// If val has an IndefiniteBlockingKey, then don't update
	if val.TransmitBlockNumber == IndefiniteBlockingKey {
		return false, nil
	}

	// return true if val.TransmitBlockNumber is higher
	return e.After(val.TransmitBlockNumber, b.TransmitBlockNumber)
}

func (rc *conditionalReportCoordinator) updateIdBlock(key string, val idBlocker) {
	idBlock, ok := rc.idBlocks.Get(key)
	if ok {
		shouldUpdate, err := idBlock.shouldUpdate(val, rc.encoder)
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

// Start starts all subprocesses
func (rc *conditionalReportCoordinator) Start() {
	if !rc.running.Load() {
		go rc.run()
		go rc.idCacheCleaner.Run(rc.idBlocks)
		go rc.cacheCleaner.Run(rc.activeKeys)

		rc.running.Swap(true)
	}
}

// Close terminates all subprocesses
func (rc *conditionalReportCoordinator) Close() error {
	if rc.running.Load() {
		rc.chStop <- struct{}{}
		rc.idCacheCleaner.Stop()
		rc.cacheCleaner.Stop()
		rc.running.Swap(false)
		<-rc.chDone
	}

	return nil
}

func (rc *conditionalReportCoordinator) run() {
	timer := time.NewTimer(cadence)

	for {
		select {
		case <-timer.C:
			startTime := time.Now()

			if err := rc.checkEvents(context.Background()); err != nil {
				rc.logger.Printf("failed to check events: %s", err)
			}

			// attempt to adhere to a cadence of at least every second
			// a slow DB will cause the cadence to increase. these cases are logged
			diff := time.Since(startTime)
			if diff > cadence {
				rc.logger.Printf("checkEvents took %dms to complete; expected cadence is %dms; check database indexes and other performance improvements", diff/time.Millisecond, cadence/time.Millisecond)
				// start again immediately
				timer.Reset(time.Microsecond)
			} else {
				// wait the difference between the cadence and the time taken
				timer.Reset(cadence - diff)
			}
		case <-rc.chStop:
			timer.Stop()
			rc.chDone <- struct{}{}
			return
		}
	}
}
