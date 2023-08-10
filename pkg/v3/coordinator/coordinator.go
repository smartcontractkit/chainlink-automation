package coordinator

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/config"
	"github.com/smartcontractkit/ocr2keepers/pkg/util"
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
			checkBlockNumber: reportedUpkeep.Trigger.BlockNumber,
		}, util.DefaultCacheExpiration)
		return true
	} else if v.checkBlockNumber < reportedUpkeep.Trigger.BlockNumber {
		c.cache.Set(reportedUpkeep.WorkID, record{
			checkBlockNumber:      reportedUpkeep.Trigger.BlockNumber,
			isTransmissionPending: v.isTransmissionPending,
			transmitType:          v.transmitType,
			transmitBlockNumber:   v.transmitBlockNumber,
		}, util.DefaultCacheExpiration)
		return true
	} else if v.checkBlockNumber > reportedUpkeep.Trigger.BlockNumber {
		return false
	}
	// TODO should we not accept if the reported payload check block is equal to the cached check block?
	return false
}

func (c *coordinator) ShouldTransmit(reportedUpkeep ocr2keepers.ReportedUpkeep) bool {
	if v, ok := c.cache.Get(reportedUpkeep.WorkID); !ok {
		return false
	} else if reportedUpkeep.Trigger.BlockNumber < v.checkBlockNumber {
		return false
	} else if reportedUpkeep.Trigger.BlockNumber == v.checkBlockNumber {
		return true
	} else {
		c.logger.Printf("libocr should call shouldAccept before shouldTransmit")
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
			return false
		} else {
			switch c.upkeepTypeGetter(payload.UpkeepID) {
			case ocr2keepers.LogTrigger:
				switch v.transmitType {
				case ocr2keepers.PerformEvent:
					return false
				default:
					return true
				}
			case ocr2keepers.ConditionTrigger:
				switch v.transmitType {
				case ocr2keepers.PerformEvent:
					return payload.Trigger.BlockNumber > v.transmitBlockNumber
				default:
					return true
				}
			}
		}
	}
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
				c.logger.Printf("Ignoring event in transaction %s from older report (block %v)", event.TransactionHash, event.CheckBlock)
			} else if event.CheckBlock == v.checkBlockNumber {
				c.cache.Set(event.WorkID, record{
					checkBlockNumber:      v.checkBlockNumber,
					isTransmissionPending: true,
					transmitType:          event.Type,
					transmitBlockNumber:   event.TransmitBlock,
				}, util.DefaultCacheExpiration)
			} else {
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
				c.logger.Printf("failed to check events: %s", err)
			}

			// attempt to adhere to a cadence of at least every second
			// a slow DB will cause the cadence to increase. these cases are logged
			diff := time.Since(startTime)
			if diff > cadence {
				c.logger.Printf("log poll took %dms to complete; expected cadence is %dms; check database indexes and other performance improvements", diff/time.Millisecond, cadence/time.Millisecond)
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
