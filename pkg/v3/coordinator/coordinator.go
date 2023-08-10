package coordinator

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/util"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/config"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

const (
	cadence           = time.Second
	defaultCacheClean = time.Duration(30) * time.Second
)

type coordinator struct {
	logger               *log.Logger
	eventsProvider       ocr2keepers.TransmitEventProvider
	upkeepTypeGetter     ocr2keepers.UpkeepTypeGetter
	cache                *util.Cache[record]
	cacheCleaner         *util.IntervalCacheCleaner[record]
	minimumConfirmations int
	running              atomic.Bool
	chStop               chan struct{}
}

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
		cache:                util.NewCache[record](time.Hour),
		cacheCleaner:         util.NewIntervalCacheCleaner[record](defaultCacheClean),
		minimumConfirmations: conf.MinConfirmations,
		chStop:               make(chan struct{}, 1),
	}
}

func (c *coordinator) ShouldAccept(reportedUpkeep ocr2keepers.ReportedUpkeep) bool {
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
		if c.ShouldProcess(payload) {
			res = append(res, payload)
		}
	}
	return res, nil
}

func (c *coordinator) ShouldProcess(payload ocr2keepers.UpkeepPayload) bool {
	if v, ok := c.cache.Get(payload.WorkID); ok {
		if v.isTransmissionPending {
			// This workID has a pending transmit, should not process it
			return false
		} else {
			switch c.upkeepTypeGetter(payload.UpkeepID) {
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
					return payload.Trigger.BlockNumber > v.transmitBlockNumber
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

	for _, event := range events {
		if event.Confirmations < int64(c.minimumConfirmations) {
			c.logger.Printf("Skipping event in transaction %s as confirmations (%d) is less than minimum confirmations (%d)", event.TransactionHash, event.Confirmations, c.minimumConfirmations)
			continue
		}

		if v, ok := c.cache.Get(event.WorkID); ok {
			if event.CheckBlock < v.checkBlockNumber {
				c.logger.Printf("Ignoring event in transaction %s from older report (block %v) while waiting for (block %v)", event.TransactionHash, event.CheckBlock, v.checkBlockNumber)
			} else if event.CheckBlock == v.checkBlockNumber {
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

	return nil
}

func (c *coordinator) run() {
	timer := time.NewTimer(cadence)
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
		case <-c.chStop:
			timer.Stop()
			return
		}
	}
}

// Start starts all subprocesses
func (c *coordinator) Start(_ context.Context) error {
	if c.running.Load() {
		return fmt.Errorf("process already running")
	}

	go c.cacheCleaner.Run(c.cache)

	c.running.Store(true)
	c.run()

	return nil
}

// Close terminates all subprocesses
func (c *coordinator) Close() error {
	if !c.running.Load() {
		return fmt.Errorf("process not running")
	}

	c.cacheCleaner.Stop()
	c.chStop <- struct{}{}
	c.running.Store(false)

	return nil
}
