package coordinator

import (
	"context"
	"fmt"
	"log"
	"time"

	internalutil "github.com/smartcontractkit/ocr2keepers/internal/util"
	"github.com/smartcontractkit/ocr2keepers/pkg/util"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/config"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

const (
	cadence           = time.Second
	defaultCacheClean = time.Duration(30) * time.Second
)

type coordinator struct {
	closer               internalutil.Closer
	logger               *log.Logger
	eventsProvider       ocr2keepers.TransmitEventProvider
	upkeepTypeGetter     ocr2keepers.UpkeepTypeGetter
	cache                *util.Cache[record]
	cacheCleaner         *util.IntervalCacheCleaner[record]
	minimumConfirmations int
}

var _ ocr2keepers.Coordinator = (*coordinator)(nil)

type record struct {
	checkBlockNumber      ocr2keepers.BlockNumber
	isTransmissionPending bool // false = transmitted
	transmitType          ocr2keepers.TransmitEventType
	transmitBlockNumber   ocr2keepers.BlockNumber
}

func NewCoordinator(transmitEventProvider ocr2keepers.TransmitEventProvider, upkeepTypeGetter ocr2keepers.UpkeepTypeGetter, conf config.OffchainConfig, logger *log.Logger) *coordinator {
	return &coordinator{
		logger:               logger,
		eventsProvider:       transmitEventProvider,
		upkeepTypeGetter:     upkeepTypeGetter,
		cache:                util.NewCache[record](time.Duration(conf.PerformLockoutWindow) * time.Millisecond),
		cacheCleaner:         util.NewIntervalCacheCleaner[record](defaultCacheClean),
		minimumConfirmations: conf.MinConfirmations,
	}
}

func (c *coordinator) Accept(reportedUpkeep ocr2keepers.ReportedUpkeep) bool {
	if v, ok := c.cache.Get(reportedUpkeep.WorkID); !ok {
		c.cache.Set(reportedUpkeep.WorkID, record{
			checkBlockNumber:      reportedUpkeep.Trigger.BlockNumber,
			isTransmissionPending: true,
		}, util.DefaultCacheExpiration)
		return true
	} else if v.checkBlockNumber < reportedUpkeep.Trigger.BlockNumber {
		c.cache.Set(reportedUpkeep.WorkID, record{
			checkBlockNumber:      reportedUpkeep.Trigger.BlockNumber,
			isTransmissionPending: true,
		}, util.DefaultCacheExpiration)
		return true
	}
	// We are already waiting on a higher checkBlockNumber so no need to accept this report
	return false
}

func (c *coordinator) ShouldTransmit(reportedUpkeep ocr2keepers.ReportedUpkeep) bool {
	if v, ok := c.cache.Get(reportedUpkeep.WorkID); !ok {
		// We never saw this report, so don't try to transmit
		// Can happen in edge cases when plugin restarts after shouldAccept was called
		return false
	} else if reportedUpkeep.Trigger.BlockNumber < v.checkBlockNumber {
		// We already accepted a report for a higher checkBlockNumber, so don't try to transmit
		return false
	} else if reportedUpkeep.Trigger.BlockNumber == v.checkBlockNumber {
		return true
	} else {
		// We never saw this report for such a high block number, so don't try to transmit
		// Can happen in edge cases when plugin restarts after shouldAccept was called
		return false
	}
}

func (c *coordinator) PreProcess(_ context.Context, payloads []ocr2keepers.UpkeepPayload) ([]ocr2keepers.UpkeepPayload, error) {
	res := make([]ocr2keepers.UpkeepPayload, 0)
	for _, payload := range payloads {
		if c.ShouldProcess(payload.WorkID, payload.UpkeepID, payload.Trigger) {
			res = append(res, payload)
		}
	}
	return res, nil
}

func (c *coordinator) FilterResults(results []ocr2keepers.CheckResult) ([]ocr2keepers.CheckResult, error) {
	res := make([]ocr2keepers.CheckResult, 0)
	for _, result := range results {
		if c.ShouldProcess(result.WorkID, result.UpkeepID, result.Trigger) {
			res = append(res, result)
		}
	}
	return res, nil
}

func (c *coordinator) FilterProposals(proposals []ocr2keepers.CoordinatedBlockProposal) ([]ocr2keepers.CoordinatedBlockProposal, error) {
	res := make([]ocr2keepers.CoordinatedBlockProposal, 0)
	for _, proposal := range proposals {
		if v, ok := c.cache.Get(proposal.WorkID); ok {
			if v.isTransmissionPending {
				// This workID has a pending transmit, should not process it
				continue
			} else if c.upkeepTypeGetter(proposal.UpkeepID) == ocr2keepers.LogTrigger && v.transmitType == ocr2keepers.PerformEvent {
				// For log triggers if workID was performed then skip
				// However for conditional triggers, allow proposals to be made for newer check block numbers
				continue
			}
		}
		res = append(res, proposal)
	}
	return res, nil
}

func (c *coordinator) ShouldProcess(workID string, upkeepID ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) bool {
	if v, ok := c.cache.Get(workID); ok {
		if v.isTransmissionPending {
			// This workID has a pending transmit, should not process it
			return false
		} else {
			switch c.upkeepTypeGetter(upkeepID) {
			case ocr2keepers.LogTrigger:
				switch v.transmitType {
				case ocr2keepers.PerformEvent:
					// For log triggers, a particular workID should only ever be performed once
					return false
				default:
					// There was an attempt to perform this workID, but it failed, so should be processed again
					return true
				}
			case ocr2keepers.ConditionTrigger:
				switch v.transmitType {
				case ocr2keepers.PerformEvent:
					// For conditionals, a particular workID should only be checked after its last perform
					return trigger.BlockNumber > v.transmitBlockNumber
				default:
					// There was an attempt to check this workID, but it failed, so should be processed again
					return true
				}
			}
		}
	}
	// If we have never seen this workID before, then we should process it
	return true
}

func (c *coordinator) checkEvents(ctx context.Context) error {
	events, err := c.eventsProvider.GetLatestEvents(ctx)
	if err != nil {
		return err
	}

	skipped := 0
	for _, event := range events {
		if event.Confirmations < int64(c.minimumConfirmations) {
			skipped++
			continue
		}

		if v, ok := c.cache.Get(event.WorkID); ok {
			if event.CheckBlock < v.checkBlockNumber {
				c.logger.Printf("Ignoring event in transaction %s from older report (block %v) while waiting for (block %v)", event.TransactionHash, event.CheckBlock, v.checkBlockNumber)
			} else if event.CheckBlock == v.checkBlockNumber {
				c.logger.Printf("Got event in transaction %s of type %d for workID %s and check block %v", event.TransactionHash, event.Type, event.WorkID, event.CheckBlock)
				c.cache.Set(event.WorkID, record{
					checkBlockNumber:      v.checkBlockNumber,
					isTransmissionPending: false,
					transmitType:          event.Type,
					transmitBlockNumber:   event.TransmitBlock,
				}, util.DefaultCacheExpiration)
			} else {
				c.logger.Printf("Got event in transaction %s from newer report (block %v) while waiting for (block %v)", event.TransactionHash, event.CheckBlock, v.checkBlockNumber)
				c.cache.Set(event.WorkID, record{
					checkBlockNumber:      event.CheckBlock,
					isTransmissionPending: false,
					transmitType:          event.Type,
					transmitBlockNumber:   event.TransmitBlock,
				}, util.DefaultCacheExpiration)
			}
		}
	}
	c.logger.Printf("Skipped %d events as confirmations are less than minimum confirmations (%d)", skipped, c.minimumConfirmations)

	return nil
}

func (c *coordinator) run(ctx context.Context) {
	timer := time.NewTimer(cadence)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			startTime := time.Now()

			if err := c.checkEvents(context.Background()); err != nil {
				c.logger.Printf("failed to check for transmit events: %s", err)
			}

			// attempt to adhere to a cadence of at least every second
			// a slow DB will cause the cadence to increase. these cases are logged
			diff := time.Since(startTime)
			if diff > cadence {
				c.logger.Printf("check transmit events took %dms to complete; expected cadence is %dms; check database indexes and other performance improvements", diff/time.Millisecond, cadence/time.Millisecond)
				// start again immediately
				timer.Reset(time.Microsecond)
			} else {
				// wait the difference between the cadence and the time taken
				timer.Reset(cadence - diff)
			}
		case <-ctx.Done():
			return
		}
	}
}

// Start starts all subprocesses
func (c *coordinator) Start(pctx context.Context) error {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	if !c.closer.Store(cancel) {
		return fmt.Errorf("process already running")
	}

	go c.cacheCleaner.Run(c.cache)

	c.run(ctx)

	return nil
}

// Close terminates all subprocesses
func (c *coordinator) Close() error {
	if !c.closer.Close() {
		return fmt.Errorf("process not running")
	}

	c.cacheCleaner.Stop()

	return nil
}
