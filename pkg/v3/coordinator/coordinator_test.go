package coordinator

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/ocr2keepers/pkg/util"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/config"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func TestNewCoordinator(t *testing.T) {
	t.Run("the coordinator starts and stops without erroring", func(t *testing.T) {
		// these vars help us identify when it's safe to close the coordinator
		callCount := 0
		fullRunComplete := make(chan struct{}, 1)

		// constructor dependencies
		upkeepTypeGetter := func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
			return ocr2keepers.ConditionTrigger
		}

		eventProvider := &mockEventProvider{
			GetLatestEventsFn: func(ctx context.Context) ([]ocr2keepers.TransmitEvent, error) {
				callCount++
				if callCount > 1 {
					fullRunComplete <- struct{}{}
				}
				return []ocr2keepers.TransmitEvent{}, nil
			},
		}

		logger := log.New(io.Discard, "coordinator_test", 0)

		c := NewCoordinator(eventProvider, upkeepTypeGetter, config.OffchainConfig{PerformLockoutWindow: 3600 * 1000, MinConfirmations: 2}, logger)

		go func() {
			err := c.Start(context.Background())
			assert.NoError(t, err)
		}()

		// wait for one full run of the coordinator before closing
		<-fullRunComplete

		err := c.Close()
		assert.NoError(t, err)
	})

	t.Run("if an error is encountered when checking events, a message is logged", func(t *testing.T) {
		// these vars help us identify when it's safe to close the coordinator
		callCount := 0
		fullRunComplete := make(chan struct{}, 1)

		// constructor dependencies
		upkeepTypeGetter := func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
			return ocr2keepers.ConditionTrigger
		}

		eventProvider := &mockEventProvider{
			GetLatestEventsFn: func(ctx context.Context) ([]ocr2keepers.TransmitEvent, error) {
				callCount++
				if callCount > 1 {
					fullRunComplete <- struct{}{}
				}
				// returning an error here will cause checkEvents to log an error
				return nil, errors.New("get latest events boom")
			},
		}

		logger := log.New(io.Discard, "coordinator_test", 0)

		var memLog bytes.Buffer
		logger.SetOutput(&memLog)

		c := NewCoordinator(eventProvider, upkeepTypeGetter, config.OffchainConfig{PerformLockoutWindow: 3600 * 1000, MinConfirmations: 2}, logger)

		go func() {
			err2 := c.Start(context.Background())
			assert.NoError(t, err2)
		}()

		// wait for one full run of the coordinator before closing
		<-fullRunComplete

		err := c.Close()
		assert.NoError(t, err)
	})

	t.Run("if checking events takes longer than the loop cadence, a message is logged", func(t *testing.T) {
		// these vars help us identify when it's safe to close the coordinator
		callCount := 0
		fullRunComplete := make(chan struct{}, 1)

		// constructor dependencies
		upkeepTypeGetter := func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
			return ocr2keepers.ConditionTrigger
		}

		eventProvider := &mockEventProvider{
			GetLatestEventsFn: func(ctx context.Context) ([]ocr2keepers.TransmitEvent, error) {
				callCount++
				if callCount > 1 {
					fullRunComplete <- struct{}{}
				}

				time.Sleep(cadence * 2)

				return []ocr2keepers.TransmitEvent{}, nil
			},
		}

		logger := log.New(io.Discard, "coordinator_test", 0)

		var memLog bytes.Buffer
		logger.SetOutput(&memLog)

		c := NewCoordinator(eventProvider, upkeepTypeGetter, config.OffchainConfig{PerformLockoutWindow: 3600 * 1000, MinConfirmations: 2}, logger)

		go func() {
			err := c.Start(context.Background())
			assert.NoError(t, err)
		}()

		// wait for one full run of the coordinator before closing
		<-fullRunComplete

		err := c.Close()
		assert.NoError(t, err)

		assert.True(t, strings.Contains(memLog.String(), "check database indexes and other performance improvements"))
	})

	t.Run("starting an already started coordinator returns an error", func(t *testing.T) {
		c := NewCoordinator(nil, nil, config.OffchainConfig{PerformLockoutWindow: 3600 * 1000, MinConfirmations: 2}, nil)
		c.running.Store(true)
		err := c.Start(context.Background())
		assert.Error(t, err)
	})

	t.Run("closing an already closed coordinator returns an error", func(t *testing.T) {
		c := NewCoordinator(nil, nil, config.OffchainConfig{PerformLockoutWindow: 3600 * 1000, MinConfirmations: 2}, nil)
		c.running.Store(false)
		err := c.Close()
		assert.Error(t, err)
	})
}

func TestNewCoordinator_checkEvents(t *testing.T) {
	for _, tc := range []struct {
		name             string
		upkeepTypeGetter ocr2keepers.UpkeepTypeGetter
		eventProvider    ocr2keepers.TransmitEventProvider
		cacheInit        map[string]record
		wantCache        map[string]record
		expectsErr       bool
		wantErr          error
		expectsMessage   bool
		wantMessage      string
	}{
		{
			name: "if GetLatestEvents errors, the error is returned",
			eventProvider: &mockEventProvider{
				GetLatestEventsFn: func(ctx context.Context) ([]ocr2keepers.TransmitEvent, error) {
					return nil, errors.New("get latest events boom")
				},
			},
			expectsErr: true,
			wantErr:    errors.New("get latest events boom"),
		},
		{
			name: "if a transmit event has fewer than the required minimum confirmations, a message is logged",
			eventProvider: &mockEventProvider{
				GetLatestEventsFn: func(ctx context.Context) ([]ocr2keepers.TransmitEvent, error) {
					return []ocr2keepers.TransmitEvent{
						{
							Confirmations: 1,
						},
					}, nil
				},
			},
			expectsMessage: true,
			wantMessage:    "Skipping event in transaction",
		},
		{
			name: "if a transmit event has a lower check block number than the corresponding record in the cache, a message is logged",
			eventProvider: &mockEventProvider{
				GetLatestEventsFn: func(ctx context.Context) ([]ocr2keepers.TransmitEvent, error) {
					return []ocr2keepers.TransmitEvent{
						{
							Confirmations: 2,
							CheckBlock:    ocr2keepers.BlockNumber(99),
							WorkID:        "workID1",
						},
					}, nil
				},
			},
			cacheInit: map[string]record{
				"workID1": {
					checkBlockNumber: 100,
					transmitType:     ocr2keepers.PerformEvent,
				},
			},
			expectsMessage: true,
			wantMessage:    "Ignoring event in transaction",
		},
		{
			name: "if a transmit event has a matching block number with the corresponding record in the cache, the record is updated",
			eventProvider: &mockEventProvider{
				GetLatestEventsFn: func(ctx context.Context) ([]ocr2keepers.TransmitEvent, error) {
					return []ocr2keepers.TransmitEvent{
						{
							Confirmations: 2,
							Type:          ocr2keepers.PerformEvent,
							CheckBlock:    ocr2keepers.BlockNumber(100),
							WorkID:        "workID1",
							TransmitBlock: ocr2keepers.BlockNumber(99),
						},
					}, nil
				},
			},
			cacheInit: map[string]record{
				"workID1": {
					checkBlockNumber:      ocr2keepers.BlockNumber(100),
					transmitType:          ocr2keepers.PerformEvent,
					isTransmissionPending: false,
					transmitBlockNumber:   ocr2keepers.BlockNumber(99),
				},
			},
			wantCache: map[string]record{
				"workID1": {
					checkBlockNumber:      ocr2keepers.BlockNumber(100),
					transmitType:          ocr2keepers.PerformEvent,
					isTransmissionPending: false,
					transmitBlockNumber:   ocr2keepers.BlockNumber(99),
				},
			},
		},
		{
			name: "if a transmit event has a higher block number than the corresponding record in the cache, the record is completely reset with the transmit event data",
			eventProvider: &mockEventProvider{
				GetLatestEventsFn: func(ctx context.Context) ([]ocr2keepers.TransmitEvent, error) {
					return []ocr2keepers.TransmitEvent{
						{
							Confirmations: 2,
							Type:          ocr2keepers.PerformEvent,
							CheckBlock:    ocr2keepers.BlockNumber(200),
							WorkID:        "workID1",
							TransmitBlock: ocr2keepers.BlockNumber(99),
						},
					}, nil
				},
			},
			cacheInit: map[string]record{
				"workID1": {
					checkBlockNumber:      ocr2keepers.BlockNumber(100),
					transmitType:          ocr2keepers.PerformEvent,
					isTransmissionPending: false,
					transmitBlockNumber:   ocr2keepers.BlockNumber(99),
				},
			},
			wantCache: map[string]record{
				"workID1": {
					checkBlockNumber:      ocr2keepers.BlockNumber(200),
					transmitType:          ocr2keepers.PerformEvent,
					isTransmissionPending: false,
					transmitBlockNumber:   ocr2keepers.BlockNumber(99),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger := log.New(io.Discard, "coordinator_test", 0)
			var memLog bytes.Buffer
			logger.SetOutput(&memLog)

			c := NewCoordinator(tc.eventProvider, tc.upkeepTypeGetter, config.OffchainConfig{PerformLockoutWindow: 3600 * 1000, MinConfirmations: 2}, logger)
			// initialise the cache if needed
			for k, v := range tc.cacheInit {
				c.cache.Set(k, v, util.DefaultCacheExpiration)
			}

			err := c.checkEvents(context.Background())
			if tc.expectsErr {
				assert.Error(t, err)
				assert.Equal(t, err.Error(), tc.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if tc.expectsMessage {
				assert.True(t, strings.Contains(memLog.String(), tc.wantMessage))
			}

			if len(tc.wantCache) > 0 {
				assert.Equal(t, len(tc.wantCache), len(c.cache.Keys()))
				for k, v := range tc.wantCache {
					cachedValue, ok := c.cache.Get(k)
					assert.True(t, ok)
					assert.Equal(t, v, cachedValue)
				}
			}
		})
	}
}

func TestCoordinator_ShouldAccept(t *testing.T) {
	for _, tc := range []struct {
		name           string
		cacheInit      map[string]record
		reportedUpkeep ocr2keepers.ReportedUpkeep
		shouldAccept   bool
		wantCache      map[string]record
	}{
		{
			name: "if the given work ID does not exist in the cache, we should accept and update the cache",
			reportedUpkeep: ocr2keepers.ReportedUpkeep{
				WorkID: "workID1",
				Trigger: ocr2keepers.Trigger{
					BlockNumber: 200,
				},
			},
			shouldAccept: true,
			wantCache: map[string]record{
				"workID1": {
					checkBlockNumber:      200,
					isTransmissionPending: true,
				},
			},
		},
		{
			name: "if the given work ID does exist in the cache, we should accept and update the cached check block number when the reported upkeep's check block number is higher than the cached check block",
			cacheInit: map[string]record{
				"workID1": {
					checkBlockNumber:      100,
					isTransmissionPending: true,
					transmitType:          ocr2keepers.PerformEvent,
					transmitBlockNumber:   99,
				},
			},
			reportedUpkeep: ocr2keepers.ReportedUpkeep{
				WorkID: "workID1",
				Trigger: ocr2keepers.Trigger{
					BlockNumber: 200,
				},
			},
			shouldAccept: true,
			wantCache: map[string]record{
				"workID1": {
					checkBlockNumber:      200,
					isTransmissionPending: true,
				},
			},
		},
		{
			name: "if the given work ID does exist in the cache, and the reported upkeep's check block number is lower than the cached check block, we should not accept",
			cacheInit: map[string]record{
				"workID1": {
					checkBlockNumber:      100,
					isTransmissionPending: true,
					transmitType:          ocr2keepers.PerformEvent,
					transmitBlockNumber:   99,
				},
			},
			reportedUpkeep: ocr2keepers.ReportedUpkeep{
				WorkID: "workID1",
				Trigger: ocr2keepers.Trigger{
					BlockNumber: 99,
				},
			},
			shouldAccept: false,
			wantCache: map[string]record{
				"workID1": {
					checkBlockNumber:      100,
					isTransmissionPending: true,
					transmitType:          ocr2keepers.PerformEvent,
					transmitBlockNumber:   99,
				},
			},
		},
		{
			name: "if the given work ID does exist in the cache, and the reported upkeep's check block number is equal to the cached check block, we should not accept",
			cacheInit: map[string]record{
				"workID1": {
					checkBlockNumber:      100,
					isTransmissionPending: true,
					transmitType:          ocr2keepers.PerformEvent,
					transmitBlockNumber:   99,
				},
			},
			reportedUpkeep: ocr2keepers.ReportedUpkeep{
				WorkID: "workID1",
				Trigger: ocr2keepers.Trigger{
					BlockNumber: 100,
				},
			},
			shouldAccept: false,
			wantCache: map[string]record{
				"workID1": {
					checkBlockNumber:      100,
					isTransmissionPending: true,
					transmitType:          ocr2keepers.PerformEvent,
					transmitBlockNumber:   99,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCoordinator(nil, nil, config.OffchainConfig{}, nil)
			// initialise the cache
			for k, v := range tc.cacheInit {
				c.cache.Set(k, v, util.DefaultCacheExpiration)
			}

			shouldAccept := c.Accept(tc.reportedUpkeep)

			assert.Equal(t, tc.shouldAccept, shouldAccept)

			if len(tc.wantCache) > 0 {
				assert.Equal(t, len(tc.wantCache), len(c.cache.Keys()))
				for k, v := range tc.wantCache {
					cachedValue, ok := c.cache.Get(k)
					assert.True(t, ok)
					assert.Equal(t, v, cachedValue)
				}
			}
		})
	}
}
func TestCoordinator_ShouldTransmit(t *testing.T) {
	for _, tc := range []struct {
		name           string
		cacheInit      map[string]record
		reportedUpkeep ocr2keepers.ReportedUpkeep
		expectsMessage bool
		wantMessage    string
		shouldTransmit bool
	}{
		{
			name: "if the given work ID does not exist in the cache, we should not transmit",
			reportedUpkeep: ocr2keepers.ReportedUpkeep{
				WorkID: "workID1",
			},
			shouldTransmit: false,
		},
		{
			name: "if the given work ID does exist in the cache, and the reported upkeep's check block is lower than the cached check block, we should not transmit",
			cacheInit: map[string]record{
				"workID1": {
					checkBlockNumber: 200,
				},
			},
			reportedUpkeep: ocr2keepers.ReportedUpkeep{
				WorkID: "workID1",
				Trigger: ocr2keepers.Trigger{
					BlockNumber: 100,
				},
			},
			shouldTransmit: false,
		},
		{
			name: "if the given work ID does exist in the cache, and the reported upkeep's check block is equal to the cached check block, we should transmit",
			cacheInit: map[string]record{
				"workID1": {
					checkBlockNumber: 200,
				},
			},
			reportedUpkeep: ocr2keepers.ReportedUpkeep{
				WorkID: "workID1",
				Trigger: ocr2keepers.Trigger{
					BlockNumber: 200,
				},
			},
			shouldTransmit: true,
		},
		{
			name: "if the given work ID does exist in the cache, and the reported upkeep's check block is greater than the cached check block, we should not transmit",
			cacheInit: map[string]record{
				"workID1": {
					checkBlockNumber: 100,
				},
			},
			reportedUpkeep: ocr2keepers.ReportedUpkeep{
				WorkID: "workID1",
				Trigger: ocr2keepers.Trigger{
					BlockNumber: 200,
				},
			},
			shouldTransmit: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger := log.New(io.Discard, "coordinator_test", 0)
			var memLog bytes.Buffer
			logger.SetOutput(&memLog)

			c := NewCoordinator(nil, nil, config.OffchainConfig{}, logger)
			// initialise the cache
			for k, v := range tc.cacheInit {
				c.cache.Set(k, v, util.DefaultCacheExpiration)
			}
			shouldTransmit := c.ShouldTransmit(tc.reportedUpkeep)
			assert.Equal(t, tc.shouldTransmit, shouldTransmit)
			if tc.expectsMessage {
				assert.True(t, strings.Contains(memLog.String(), tc.wantMessage))
			}
		})
	}
}

func TestCoordinator_ShouldProcess(t *testing.T) {
	for _, tc := range []struct {
		name             string
		upkeepTypeGetter ocr2keepers.UpkeepTypeGetter
		cacheInit        map[string]record
		payload          ocr2keepers.UpkeepPayload
		shouldProcess    bool
	}{
		{
			name: "if the given work ID does not exist in the cache, we should process",
			payload: ocr2keepers.UpkeepPayload{
				WorkID: "workID1",
			},
			shouldProcess: true,
		},
		{
			name: "if the given work ID does exist in the cache, and is pending transmission, we should not process",
			payload: ocr2keepers.UpkeepPayload{
				WorkID: "workID1",
			},
			cacheInit: map[string]record{
				"workID1": {
					isTransmissionPending: true,
				},
			},
			shouldProcess: false,
		},
		{
			name: "work ID exists, is not pending transmission, transmit type is perform, and upkeep is log trigger, we should not process",
			payload: ocr2keepers.UpkeepPayload{
				WorkID: "workID1",
			},
			upkeepTypeGetter: func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.LogTrigger
			},
			cacheInit: map[string]record{
				"workID1": {
					isTransmissionPending: false,
					transmitType:          ocr2keepers.PerformEvent,
				},
			},
			shouldProcess: false,
		},
		{
			name: "work ID exists, is not pending transmission, transmit type is stale, and upkeep is log trigger, we should process",
			payload: ocr2keepers.UpkeepPayload{
				WorkID: "workID1",
			},
			upkeepTypeGetter: func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.LogTrigger
			},
			cacheInit: map[string]record{
				"workID1": {
					isTransmissionPending: false,
					transmitType:          ocr2keepers.StaleReportEvent,
				},
			},
			shouldProcess: true,
		},
		{
			name: "work ID exists, is not pending transmission, transmit type is perform, payload check block is greater than the cache transmit block, and upkeep is conditional, we should process",
			payload: ocr2keepers.UpkeepPayload{
				WorkID: "workID1",
				Trigger: ocr2keepers.Trigger{
					BlockNumber: 200,
				},
			},
			upkeepTypeGetter: func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.ConditionTrigger
			},
			cacheInit: map[string]record{
				"workID1": {
					isTransmissionPending: false,
					transmitType:          ocr2keepers.PerformEvent,
					transmitBlockNumber:   100,
				},
			},
			shouldProcess: true,
		},
		{
			name: "work ID exists, is not pending transmission, transmit type is perform, payload check block is less than the cache transmit block, and upkeep is conditional, we should not process",
			payload: ocr2keepers.UpkeepPayload{
				WorkID: "workID1",
				Trigger: ocr2keepers.Trigger{
					BlockNumber: 100,
				},
			},
			upkeepTypeGetter: func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.ConditionTrigger
			},
			cacheInit: map[string]record{
				"workID1": {
					isTransmissionPending: false,
					transmitType:          ocr2keepers.PerformEvent,
					transmitBlockNumber:   200,
				},
			},
			shouldProcess: false,
		},
		{
			name: "work ID exists, is not pending transmission, transmit type is stale, and upkeep is conditional, we should process",
			payload: ocr2keepers.UpkeepPayload{
				WorkID: "workID1",
			},
			upkeepTypeGetter: func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.ConditionTrigger
			},
			cacheInit: map[string]record{
				"workID1": {
					isTransmissionPending: false,
					transmitType:          ocr2keepers.StaleReportEvent,
				},
			},
			shouldProcess: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCoordinator(nil, tc.upkeepTypeGetter, config.OffchainConfig{}, nil)
			// initialise the cache
			for k, v := range tc.cacheInit {
				c.cache.Set(k, v, util.DefaultCacheExpiration)
			}
			shouldProcess := c.ShouldProcess(tc.payload.WorkID, tc.payload.UpkeepID, tc.payload.Trigger)
			assert.Equal(t, tc.shouldProcess, shouldProcess)
		})
	}
}

func TestNewCoordinator_Preprocess(t *testing.T) {
	for _, tc := range []struct {
		name             string
		upkeepTypeGetter ocr2keepers.UpkeepTypeGetter
		cacheInit        map[string]record
		payloads         []ocr2keepers.UpkeepPayload
		wantPayloads     []ocr2keepers.UpkeepPayload
	}{
		{
			name: "if the given work ID does not exist in the cache, we should process",
			payloads: []ocr2keepers.UpkeepPayload{
				{
					WorkID: "workID1",
				},
			},
			wantPayloads: []ocr2keepers.UpkeepPayload{
				{
					WorkID: "workID1",
				},
			},
		},
		{
			name: "if the given work ID does exist in the cache, and is pending transmission, we should not process",
			payloads: []ocr2keepers.UpkeepPayload{
				{
					WorkID: "workID1",
				},
			},
			wantPayloads: []ocr2keepers.UpkeepPayload{},
			cacheInit: map[string]record{
				"workID1": {
					isTransmissionPending: true,
				},
			},
		},
		{
			name: "work ID exists, is not pending transmission, transmit type is perform, and upkeep is log trigger, we should not process",
			payloads: []ocr2keepers.UpkeepPayload{
				{WorkID: "workID1"},
			},
			upkeepTypeGetter: func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.LogTrigger
			},
			cacheInit: map[string]record{
				"workID1": {
					isTransmissionPending: false,
					transmitType:          ocr2keepers.PerformEvent,
				},
			},
			wantPayloads: []ocr2keepers.UpkeepPayload{},
		},
		{
			name: "work ID exists, is not pending transmission, transmit type is stale, and upkeep is log trigger, we should process",
			payloads: []ocr2keepers.UpkeepPayload{
				{WorkID: "workID1"},
			},
			upkeepTypeGetter: func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.LogTrigger
			},
			cacheInit: map[string]record{
				"workID1": {
					isTransmissionPending: false,
					transmitType:          ocr2keepers.StaleReportEvent,
				},
			},
			wantPayloads: []ocr2keepers.UpkeepPayload{
				{WorkID: "workID1"},
			},
		},
		{
			name: "work ID exists, is not pending transmission, transmit type is perform, payload check block is greater than the cache transmit block, and upkeep is conditional, we should process",
			payloads: []ocr2keepers.UpkeepPayload{
				{
					WorkID: "workID1",
					Trigger: ocr2keepers.Trigger{
						BlockNumber: 200,
					},
				},
			},
			upkeepTypeGetter: func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.ConditionTrigger
			},
			cacheInit: map[string]record{
				"workID1": {
					isTransmissionPending: false,
					transmitType:          ocr2keepers.PerformEvent,
					transmitBlockNumber:   100,
				},
			},
			wantPayloads: []ocr2keepers.UpkeepPayload{
				{
					WorkID: "workID1",
					Trigger: ocr2keepers.Trigger{
						BlockNumber: 200,
					},
				},
			},
		},
		{
			name: "work ID exists, is not pending transmission, transmit type is perform, payload check block is less than the cache transmit block, and upkeep is conditional, we should not process",
			payloads: []ocr2keepers.UpkeepPayload{
				{
					WorkID: "workID1",
					Trigger: ocr2keepers.Trigger{
						BlockNumber: 100,
					},
				},
			},

			upkeepTypeGetter: func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.ConditionTrigger
			},
			cacheInit: map[string]record{
				"workID1": {
					isTransmissionPending: false,
					transmitType:          ocr2keepers.PerformEvent,
					transmitBlockNumber:   200,
				},
			},
			wantPayloads: []ocr2keepers.UpkeepPayload{},
		},
		{
			name: "work ID exists, is not pending transmission, transmit type is stale, and upkeep is conditional, we should process",
			payloads: []ocr2keepers.UpkeepPayload{
				{WorkID: "workID1"},
			},
			upkeepTypeGetter: func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.ConditionTrigger
			},
			cacheInit: map[string]record{
				"workID1": {
					isTransmissionPending: false,
					transmitType:          ocr2keepers.StaleReportEvent,
				},
			},
			wantPayloads: []ocr2keepers.UpkeepPayload{
				{WorkID: "workID1"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCoordinator(nil, tc.upkeepTypeGetter, config.OffchainConfig{}, nil)
			// initialise the cache
			for k, v := range tc.cacheInit {
				c.cache.Set(k, v, util.DefaultCacheExpiration)
			}
			payloads, err := c.PreProcess(context.Background(), tc.payloads)
			assert.NoError(t, err)

			assert.True(t, reflect.DeepEqual(payloads, tc.wantPayloads))
		})
	}
}

func TestCoordinator_FilterResults(t *testing.T) {
	for _, tc := range []struct {
		name             string
		upkeepTypeGetter ocr2keepers.UpkeepTypeGetter
		cacheInit        map[string]record
		results          []ocr2keepers.CheckResult
		wantResults      []ocr2keepers.CheckResult
		shouldProcess    bool
	}{
		{
			name: "if the given work ID does not exist in the cache, results are included",
			results: []ocr2keepers.CheckResult{
				{
					WorkID: "workID1",
				},
			},
			wantResults: []ocr2keepers.CheckResult{
				{
					WorkID: "workID1",
				},
			},
		},
		{
			name: "if the given work ID does exist in the cache, and is pending transmission, results are not included",
			results: []ocr2keepers.CheckResult{
				{
					WorkID: "workID1",
				},
			},
			cacheInit: map[string]record{
				"workID1": {
					isTransmissionPending: true,
				},
			},
			wantResults: []ocr2keepers.CheckResult{},
		},
		{
			name: "work ID exists, is not pending transmission, transmit type is perform, and upkeep is log trigger, results are not included",
			results: []ocr2keepers.CheckResult{
				{WorkID: "workID1"},
			},
			upkeepTypeGetter: func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.LogTrigger
			},
			cacheInit: map[string]record{
				"workID1": {
					isTransmissionPending: false,
					transmitType:          ocr2keepers.PerformEvent,
				},
			},
			wantResults: []ocr2keepers.CheckResult{},
		},
		{
			name: "work ID exists, is not pending transmission, transmit type is stale, and upkeep is log trigger, results are included",
			results: []ocr2keepers.CheckResult{
				{
					WorkID: "workID1",
				},
			},
			upkeepTypeGetter: func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.LogTrigger
			},
			cacheInit: map[string]record{
				"workID1": {
					isTransmissionPending: false,
					transmitType:          ocr2keepers.StaleReportEvent,
				},
			},
			wantResults: []ocr2keepers.CheckResult{
				{
					WorkID: "workID1",
				},
			},
		},
		{
			name: "work ID exists, is not pending transmission, transmit type is perform, payload check block is greater than the cache transmit block, and upkeep is conditional, results are included",
			results: []ocr2keepers.CheckResult{
				{
					WorkID: "workID1",
					Trigger: ocr2keepers.Trigger{
						BlockNumber: 200,
					},
				},
			},
			upkeepTypeGetter: func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.ConditionTrigger
			},
			cacheInit: map[string]record{
				"workID1": {
					isTransmissionPending: false,
					transmitType:          ocr2keepers.PerformEvent,
					transmitBlockNumber:   100,
				},
			},
			wantResults: []ocr2keepers.CheckResult{
				{
					WorkID: "workID1",
					Trigger: ocr2keepers.Trigger{
						BlockNumber: 200,
					},
				},
			},
		},
		{
			name: "work ID exists, is not pending transmission, transmit type is perform, payload check block is less than the cache transmit block, and upkeep is conditional, results are not included",
			results: []ocr2keepers.CheckResult{
				{
					WorkID: "workID1",
					Trigger: ocr2keepers.Trigger{
						BlockNumber: 100,
					},
				},
			},
			upkeepTypeGetter: func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.ConditionTrigger
			},
			cacheInit: map[string]record{
				"workID1": {
					isTransmissionPending: false,
					transmitType:          ocr2keepers.PerformEvent,
					transmitBlockNumber:   200,
				},
			},
			wantResults: []ocr2keepers.CheckResult{},
		},
		{
			name: "work ID exists, is not pending transmission, transmit type is stale, and upkeep is conditional, results are included",
			results: []ocr2keepers.CheckResult{
				{
					WorkID: "workID1",
				},
			},
			upkeepTypeGetter: func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.ConditionTrigger
			},
			cacheInit: map[string]record{
				"workID1": {
					isTransmissionPending: false,
					transmitType:          ocr2keepers.StaleReportEvent,
				},
			},
			wantResults: []ocr2keepers.CheckResult{
				{
					WorkID: "workID1",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCoordinator(nil, tc.upkeepTypeGetter, config.OffchainConfig{}, nil)
			// initialise the cache
			for k, v := range tc.cacheInit {
				c.cache.Set(k, v, util.DefaultCacheExpiration)
			}
			results, err := c.FilterResults(tc.results)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantResults, results)
		})
	}
}

func TestCoordinator_FilterProposals(t *testing.T) {
	for _, tc := range []struct {
		name             string
		upkeepTypeGetter ocr2keepers.UpkeepTypeGetter
		cacheInit        map[string]record
		results          []ocr2keepers.CoordinatedBlockProposal
		wantResults      []ocr2keepers.CoordinatedBlockProposal
		shouldProcess    bool
	}{
		{
			name: "all proposals are included",
			results: []ocr2keepers.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
			},
			wantResults: []ocr2keepers.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
			},
		},
		{
			name: "proposals with pending transmission are excluded",
			results: []ocr2keepers.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
				{
					WorkID: "workID2",
				},
			},
			cacheInit: map[string]record{
				"workID2": {
					isTransmissionPending: true,
				},
			},
			wantResults: []ocr2keepers.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
			},
		},
		{
			name: "log proposals with a non pending transmission with a perform transmit type are excluded",
			results: []ocr2keepers.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
				{
					WorkID: "workID2",
				},
			},
			upkeepTypeGetter: func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.LogTrigger
			},
			cacheInit: map[string]record{
				"workID2": {
					isTransmissionPending: false,
					transmitType:          ocr2keepers.PerformEvent,
				},
			},
			wantResults: []ocr2keepers.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
			},
		},
		{
			name: "condition trigger proposals with a non pending transmission with a perform transmit type are included",
			results: []ocr2keepers.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
				{
					WorkID: "workID2",
				},
			},
			upkeepTypeGetter: func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.ConditionTrigger
			},
			cacheInit: map[string]record{
				"workID2": {
					isTransmissionPending: false,
					transmitType:          ocr2keepers.PerformEvent,
				},
			},
			wantResults: []ocr2keepers.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
				{
					WorkID: "workID2",
				},
			},
		},
		{
			name: "log proposals with a non pending transmission with a stale report transmit type are included",
			results: []ocr2keepers.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
				{
					WorkID: "workID2",
				},
			},
			upkeepTypeGetter: func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.LogTrigger
			},
			cacheInit: map[string]record{
				"workID2": {
					isTransmissionPending: false,
					transmitType:          ocr2keepers.StaleReportEvent,
				},
			},
			wantResults: []ocr2keepers.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
				{
					WorkID: "workID2",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCoordinator(nil, tc.upkeepTypeGetter, config.OffchainConfig{}, nil)
			// initialise the cache
			for k, v := range tc.cacheInit {
				c.cache.Set(k, v, util.DefaultCacheExpiration)
			}
			results, err := c.FilterProposals(tc.results)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantResults, results)
		})
	}
}

type mockEventProvider struct {
	GetLatestEventsFn func(context.Context) ([]ocr2keepers.TransmitEvent, error)
}

func (t *mockEventProvider) GetLatestEvents(ctx context.Context) ([]ocr2keepers.TransmitEvent, error) {
	return t.GetLatestEventsFn(ctx)
}
