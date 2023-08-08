package coordinator

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/ocr2keepers/pkg/config"
	"github.com/smartcontractkit/ocr2keepers/pkg/util"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

const (
	DefaultCacheClean = time.Duration(30) * time.Second
)

//go:generate mockery --name EventProvider --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/coordinator" --case underscore --filename event_provider.generated.go
type EventProvider interface {
	Events(context.Context) ([]ocr2keepers.TransmitEvent, error)
}

type reportCoordinator struct {
	// injected dependencies
	logger           *log.Logger
	events           EventProvider
	upkeepTypeGetter ocr2keepers.UpkeepTypeGetter

	// initialised by the constructor
	activeKeys        *util.Cache[bool]
	activeKeysCleaner *util.IntervalCacheCleaner[bool]
	minConfs          int

	// run state data
	running atomic.Bool
	chStop  chan struct{}
}

func NewReportCoordinator(logs EventProvider, utg ocr2keepers.UpkeepTypeGetter, conf config.OffchainConfig, logger *log.Logger) *reportCoordinator {
	return &reportCoordinator{
		logger:            logger,
		events:            logs,
		upkeepTypeGetter:  utg,
		activeKeys:        util.NewCache[bool](time.Hour), // 1 hour allows the cleanup routine to clear stale data
		activeKeysCleaner: util.NewIntervalCacheCleaner[bool](DefaultCacheClean),
		minConfs:          conf.MinConfirmations,
		chStop:            make(chan struct{}, 1),
	}
}

// UpkeepWorkID returns the identifier using the given upkeepID and trigger extension(tx hash and log index).
func UpkeepWorkID(id *big.Int, trigger ocr2keepers.Trigger) (string, error) {
	extensionBytes, err := json.Marshal(trigger.LogTriggerExtension)
	if err != nil {
		return "", err
	}

	// TODO (auto-4314): Ensure it works with conditionals and add unit tests
	combined := fmt.Sprintf("%s%s", id, extensionBytes)
	hash := crypto.Keccak256([]byte(combined))
	return hex.EncodeToString(hash[:]), nil
}

func (rc *reportCoordinator) Accept(upkeep ocr2keepers.ReportedUpkeep) error {
	if rc.upkeepTypeGetter(upkeep.UpkeepID) != ocr2keepers.LogTrigger {
		return fmt.Errorf("Upkeep is not log event based, skipping: %s", upkeep.UpkeepID.String())
	}

	workID, err := UpkeepWorkID(upkeep.UpkeepID.BigInt(), upkeep.Trigger)
	if err != nil {
		return fmt.Errorf("Unable to build work ID: %w", err)
	}

	if _, ok := rc.activeKeys.Get(workID); !ok {
		rc.activeKeys.Set(workID, false, util.DefaultCacheExpiration)
	}

	return nil
}

func (rc *reportCoordinator) IsTransmissionConfirmed(upkeep ocr2keepers.ReportedUpkeep) bool {
	workID, err := UpkeepWorkID(upkeep.UpkeepID.BigInt(), upkeep.Trigger)
	if err != nil {
		return false
	}
	// if non-exist in cache, return true
	// if exist in cache and confirmed by log poller, return true
	confirmed, ok := rc.activeKeys.Get(workID)
	return !ok || (ok && confirmed)
}

func (rc *reportCoordinator) checkEvents(ctx context.Context) error {
	var (
		events []ocr2keepers.TransmitEvent
		err    error
	)

	events, err = rc.events.Events(ctx)
	if err != nil {
		return err
	}

	for _, evt := range events {
		if evt.Confirmations < int64(rc.minConfs) {
			rc.logger.Printf("Skipping perform log in transaction %s as confirmations (%d) is less than min confirmations (%d)", evt.TransactionHash, evt.Confirmations, rc.minConfs)
			continue
		}
		switch evt.Type {
		case ocr2keepers.PerformEvent, ocr2keepers.StaleReportEvent:
			rc.performEvent(evt)
		case ocr2keepers.ReorgReportEvent, ocr2keepers.InsufficientFundsReportEvent:
			rc.activeKeys.Delete(evt.WorkID)
			// TODO: push to recovery flow
		}
	}

	return nil
}

func (rc *reportCoordinator) performEvent(evt ocr2keepers.TransmitEvent) {
	rc.activeKeys.Set(evt.WorkID, true, util.DefaultCacheExpiration)
}

// isPending returns true if a key should be filtered out.
func (rc *reportCoordinator) isPending(payload ocr2keepers.UpkeepPayload) bool {
	if _, ok := rc.activeKeys.Get(payload.ID); ok {
		// If the payload already exists, return true
		return true
	}
	return false
}

func (rc *reportCoordinator) PreProcess(_ context.Context, payloads []ocr2keepers.UpkeepPayload) ([]ocr2keepers.UpkeepPayload, error) {
	var filteredPayloads []ocr2keepers.UpkeepPayload

	for _, payload := range payloads {
		if !rc.isPending(payload) {
			// If the payload is not pending, add it to the filteredPayloads slice
			filteredPayloads = append(filteredPayloads, payload)
		}
	}

	return filteredPayloads, nil
}

func (rc *reportCoordinator) run() {
	cadence := time.Second
	timer := time.NewTimer(cadence)

	for {
		select {
		case <-timer.C:
			startTime := time.Now()

			if err := rc.checkEvents(context.Background()); err != nil {
				rc.logger.Printf("failed to check perform and stale report logs: %s", err)
			}

			// attempt to adhere to a cadence of at least every second
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

// Start starts all subprocesses
func (rc *reportCoordinator) Start(_ context.Context) error {
	if rc.running.Load() {
		return fmt.Errorf("process already running")
	}

	go rc.activeKeysCleaner.Run(rc.activeKeys)

	rc.running.Store(true)
	rc.run()

	return nil
}

// Close terminates all subprocesses
func (rc *reportCoordinator) Close() error {
	if !rc.running.Load() {
		return fmt.Errorf("process not running")
	}

	rc.activeKeysCleaner.Stop()
	rc.chStop <- struct{}{}
	rc.running.Store(false)

	return nil
}
