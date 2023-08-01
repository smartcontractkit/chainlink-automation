package coordinator

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/config"
)

type mockEncoder struct {
	AfterFn     func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error)
	IncrementFn func(ocr2keepers.BlockKey) (ocr2keepers.BlockKey, error)
}

func (e *mockEncoder) After(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
	return e.AfterFn(a, b)
}

func (e *mockEncoder) Increment(k ocr2keepers.BlockKey) (ocr2keepers.BlockKey, error) {
	return e.IncrementFn(k)
}

type mockEvents struct {
	EventsFn func(ctx context.Context) ([]ocr2keepers.TransmitEvent, error)
}

func (e *mockEvents) Events(ctx context.Context) ([]ocr2keepers.TransmitEvent, error) {
	return e.EventsFn(ctx)
}

func TestNewConditionalReportCoordinator(t *testing.T) {
	t.Run("a new report coordinator is created successfully", func(t *testing.T) {
		events := &mockEvents{
			EventsFn: func(ctx context.Context) ([]ocr2keepers.TransmitEvent, error) {
				return []ocr2keepers.TransmitEvent{}, nil
			},
		}
		encoder := &mockEncoder{}
		coordinator := NewConditionalReportCoordinator(events, 1, log.New(io.Discard, "", 0), encoder)
		assert.NotNil(t, coordinator)

		t.Run("coordinator starts successfully", func(t *testing.T) {
			coordinator.Start()
			assert.True(t, coordinator.running.Load())

			t.Run("coordinator stops successfully", func(t *testing.T) {
				coordinator.Close()
				assert.False(t, coordinator.running.Load())
			})
		})
	})
}

func TestConditionalReportCoordinator_run(t *testing.T) {
	t.Run("coordinator runs with a cadence lower than checkEvents execution time", func(t *testing.T) {
		oldCadence := cadence
		cadence = 100 * time.Millisecond
		defer func() {
			cadence = oldCadence
		}()

		events := &mockEvents{
			EventsFn: func(ctx context.Context) ([]ocr2keepers.TransmitEvent, error) {
				time.Sleep(500 * time.Millisecond)
				return []ocr2keepers.TransmitEvent{}, nil
			},
		}
		encoder := &mockEncoder{}

		var buf bytes.Buffer
		coordinator := NewConditionalReportCoordinator(events, 1, log.New(&buf, "", 0), encoder)
		coordinator.Start()

		time.Sleep(time.Second)

		coordinator.Close()
		assert.True(t, strings.Contains(buf.String(), "check database indexes and other performance improvements"))
	})

	t.Run("coordinator runs with a cadence higher than checkEvents execution time", func(t *testing.T) {
		oldCadence := cadence
		cadence = 500 * time.Millisecond
		defer func() {
			cadence = oldCadence
		}()

		events := &mockEvents{
			EventsFn: func(ctx context.Context) ([]ocr2keepers.TransmitEvent, error) {
				time.Sleep(100 * time.Millisecond)
				return []ocr2keepers.TransmitEvent{}, nil
			},
		}
		encoder := &mockEncoder{}

		var buf bytes.Buffer
		coordinator := NewConditionalReportCoordinator(events, 1, log.New(&buf, "", 0), encoder)
		coordinator.Start()

		time.Sleep(time.Second)

		coordinator.Close()
		assert.Equal(t, buf.String(), "")
	})

	t.Run("check events errors and a message is logged", func(t *testing.T) {
		oldCadence := cadence
		cadence = 100 * time.Millisecond
		defer func() {
			cadence = oldCadence
		}()

		events := &mockEvents{
			EventsFn: func(ctx context.Context) ([]ocr2keepers.TransmitEvent, error) {
				time.Sleep(500 * time.Millisecond)
				return []ocr2keepers.TransmitEvent{}, errors.New("events error")
			},
		}
		encoder := &mockEncoder{}

		var buf bytes.Buffer
		coordinator := NewConditionalReportCoordinator(events, 1, log.New(&buf, "", 0), encoder)
		coordinator.Start()

		time.Sleep(time.Second)

		coordinator.Close()
		assert.True(t, strings.Contains(buf.String(), "failed to check"))
	})
}

func TestConditionalReportCoordinator_isPending(t *testing.T) {
	t.Run("unregistered key is not pending", func(t *testing.T) {
		events := &mockEvents{}
		encoder := &mockEncoder{}

		coordinator := NewConditionalReportCoordinator(events, 1, log.New(io.Discard, "", 0), encoder)
		assert.NotNil(t, coordinator)

		pending := coordinator.isPending(ocr2keepers.UpkeepPayload{
			WorkID: "123",
			Upkeep: ocr2keepers.ConfiguredUpkeep{
				ID: []byte("123"),
			},
		})

		assert.False(t, pending)
	})

	t.Run("registered key is pending when block key is not after the transmit block number", func(t *testing.T) {
		events := &mockEvents{}
		encoder := &mockEncoder{
			AfterFn: func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
				return false, nil
			},
		}

		coordinator := NewConditionalReportCoordinator(events, 1, log.New(io.Discard, "", 0), encoder)
		assert.NotNil(t, coordinator)

		coordinator.idBlocks.Set("123", idBlocker{
			TransmitBlockNumber: ocr2keepers.BlockKey("100"),
		}, config.DefaultCacheExpiration)

		pending := coordinator.isPending(ocr2keepers.UpkeepPayload{
			WorkID: "123",
			Upkeep: ocr2keepers.ConfiguredUpkeep{
				ID: []byte("123"),
			},
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 99,
			},
		})

		assert.True(t, pending)
	})

	t.Run("registered key is not pending when block key is after the transmit block number", func(t *testing.T) {
		events := &mockEvents{}
		encoder := &mockEncoder{
			AfterFn: func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
				return true, nil
			},
		}

		coordinator := NewConditionalReportCoordinator(events, 1, log.New(io.Discard, "", 0), encoder)
		assert.NotNil(t, coordinator)

		coordinator.idBlocks.Set("123", idBlocker{
			TransmitBlockNumber: ocr2keepers.BlockKey("100"),
		}, config.DefaultCacheExpiration)

		pending := coordinator.isPending(ocr2keepers.UpkeepPayload{
			Upkeep: ocr2keepers.ConfiguredUpkeep{
				ID: []byte("123"),
			},
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 101,
			},
		})

		assert.False(t, pending)
	})

	t.Run("registered key is pending when the encoder errors", func(t *testing.T) {
		events := &mockEvents{}
		encoder := &mockEncoder{
			AfterFn: func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
				return false, errors.New("encoder error")
			},
		}

		coordinator := NewConditionalReportCoordinator(events, 1, log.New(io.Discard, "", 0), encoder)
		assert.NotNil(t, coordinator)

		coordinator.idBlocks.Set("123", idBlocker{
			TransmitBlockNumber: ocr2keepers.BlockKey("100"),
		}, config.DefaultCacheExpiration)

		pending := coordinator.isPending(ocr2keepers.UpkeepPayload{
			WorkID: "123",
			Upkeep: ocr2keepers.ConfiguredUpkeep{
				ID: []byte("123"),
			},
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 99,
			},
		})

		assert.True(t, pending)
	})
}

func TestConditionalReportCoordinator_updateIdBlock(t *testing.T) {
	t.Run("updateIdBlock for a non existent key sets the idBlocker on the cache", func(t *testing.T) {
		events := &mockEvents{}
		encoder := &mockEncoder{
			AfterFn: func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
				return false, nil
			},
		}

		var buf bytes.Buffer
		coordinator := NewConditionalReportCoordinator(events, 1, log.New(&buf, "", 0), encoder)
		assert.NotNil(t, coordinator)

		coordinator.updateIdBlock("abc", idBlocker{
			CheckBlockNumber: "123",
		})

		assert.True(t, strings.Contains(buf.String(), "updateIdBlock for key abc: value updated"))

		block, ok := coordinator.idBlocks.Get("abc")
		assert.NotNil(t, block)
		assert.True(t, ok)
	})

	t.Run("updateIdBlock for an existing key checks if the cache should be updated", func(t *testing.T) {
		events := &mockEvents{}
		encoder := &mockEncoder{
			AfterFn: func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
				return false, nil
			},
		}

		var buf bytes.Buffer
		coordinator := NewConditionalReportCoordinator(events, 1, log.New(&buf, "", 0), encoder)
		assert.NotNil(t, coordinator)

		blocker := idBlocker{
			CheckBlockNumber: "123",
		}
		coordinator.idBlocks.Set("abc", blocker, config.DefaultCacheExpiration)

		coordinator.updateIdBlock("abc", blocker)

		assert.True(t, strings.Contains(buf.String(), "updateIdBlock for key abc: Not updating"))

		block, ok := coordinator.idBlocks.Get("abc")
		assert.NotNil(t, block)
		assert.True(t, ok)
	})

	t.Run("updateIdBlock for an existing key checks if the cache should be updated, but is a no op when the encoder errors", func(t *testing.T) {
		events := &mockEvents{}
		encoder := &mockEncoder{
			AfterFn: func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
				return false, errors.New("after error")
			},
		}

		var buf bytes.Buffer
		coordinator := NewConditionalReportCoordinator(events, 1, log.New(&buf, "", 0), encoder)
		assert.NotNil(t, coordinator)

		blocker := idBlocker{
			CheckBlockNumber: "123",
		}
		coordinator.idBlocks.Set("abc", blocker, config.DefaultCacheExpiration)

		coordinator.updateIdBlock("abc", blocker)

		block, ok := coordinator.idBlocks.Get("abc")
		assert.NotNil(t, block)
		assert.True(t, ok)
	})
}

func TestIDBlocker_shouldUpdate(t *testing.T) {
	for _, tc := range []struct {
		name      string
		idBlocker idBlocker
		val       idBlocker
		encoder   Encoder

		wantRes bool
		wantErr error
	}{
		{
			name: "erroring encoder returns false",
			idBlocker: idBlocker{
				CheckBlockNumber: "123",
			},
			val: idBlocker{
				CheckBlockNumber: "456",
			},
			encoder: &mockEncoder{
				AfterFn: func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
					return false, errors.New("after error")
				},
			},
			wantRes: false,
			wantErr: errors.New("after error"),
		},
		{
			name: "true when the val checkBlockNumber is after this checkBlockNumber",
			idBlocker: idBlocker{
				CheckBlockNumber: "123",
			},
			val: idBlocker{
				CheckBlockNumber: "456",
			},
			encoder: &mockEncoder{
				AfterFn: func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
					if string(a) == "456" {
						return true, nil
					}
					return false, nil
				},
			},
			wantRes: true,
			wantErr: nil,
		},
		{
			name: "erroring encoder returns false",
			idBlocker: idBlocker{
				CheckBlockNumber: "999",
			},
			val: idBlocker{
				CheckBlockNumber: "111",
			},
			encoder: &mockEncoder{
				AfterFn: func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
					if string(a) == "999" {
						return false, errors.New("after error")
					}
					return false, nil
				},
			},
			wantRes: false,
			wantErr: errors.New("after error"),
		},
		{
			name: "false when this check block number is higher than val checkBlockNumber",
			idBlocker: idBlocker{
				CheckBlockNumber: "999",
			},
			val: idBlocker{
				CheckBlockNumber: "111",
			},
			encoder: &mockEncoder{
				AfterFn: func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
					if string(a) == "999" {
						return true, nil
					}
					return false, nil
				},
			},
			wantRes: false,
			wantErr: nil,
		},
		{
			name: "true when this transmitBlockNumber is the IndefiniteBlockingKey",
			idBlocker: idBlocker{
				CheckBlockNumber:    "999",
				TransmitBlockNumber: IndefiniteBlockingKey,
			},
			val: idBlocker{
				CheckBlockNumber: "111",
			},
			encoder: &mockEncoder{
				AfterFn: func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
					if string(a) == "999" {
						return false, nil
					}
					return false, nil
				},
			},
			wantRes: true,
			wantErr: nil,
		},
		{
			name: "false when the val transmitBlockNumber is the IndefiniteBlockingKey",
			idBlocker: idBlocker{
				CheckBlockNumber: "999",
			},
			val: idBlocker{
				CheckBlockNumber:    "111",
				TransmitBlockNumber: IndefiniteBlockingKey,
			},
			encoder: &mockEncoder{
				AfterFn: func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
					if string(a) == "999" {
						return false, nil
					}
					return false, nil
				},
			},
			wantRes: false,
			wantErr: nil,
		},
		{
			name: "true when the val transmitBlockNumber is higher",
			idBlocker: idBlocker{
				CheckBlockNumber:    "999",
				TransmitBlockNumber: "199",
			},
			val: idBlocker{
				CheckBlockNumber:    "111",
				TransmitBlockNumber: "200",
			},
			encoder: &mockEncoder{
				AfterFn: func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
					if string(a) == "200" {
						return true, nil
					}
					return false, nil
				},
			},
			wantRes: true,
			wantErr: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, err := tc.idBlocker.shouldUpdate(tc.val, tc.encoder)
			assert.Equal(t, tc.wantRes, res)
			if tc.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.wantErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConditionalReportCoordinator_Accept(t *testing.T) {
	t.Run("a non existent key is accepted", func(t *testing.T) {
		events := &mockEvents{}
		encoder := &mockEncoder{
			AfterFn: func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
				return false, nil
			},
		}

		var buf bytes.Buffer
		coordinator := NewConditionalReportCoordinator(events, 1, log.New(&buf, "", 0), encoder)
		assert.NotNil(t, coordinator)

		err := coordinator.Accept(ocr2keepers.ReportedUpkeep{
			ID:       "123",
			WorkID:   "10",
			UpkeepID: ocr2keepers.UpkeepIdentifier("10"),
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 567,
			},
		})
		assert.NoError(t, err)

		val, ok := coordinator.activeKeys.Get("10")
		assert.True(t, ok)
		assert.Equal(t, false, val)

		block, ok := coordinator.idBlocks.Get("10")
		assert.True(t, ok)
		assert.Equal(t, idBlocker{
			CheckBlockNumber:    "567",
			TransmitBlockNumber: IndefiniteBlockingKey,
		}, block)
	})
}

func TestConditionalReportCoordinator_IsTransmissionConfirmed(t *testing.T) {
	t.Run("a non existent key is confirmed", func(t *testing.T) {
		events := &mockEvents{}
		encoder := &mockEncoder{
			AfterFn: func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
				return false, nil
			},
		}

		var buf bytes.Buffer
		coordinator := NewConditionalReportCoordinator(events, 1, log.New(&buf, "", 0), encoder)
		assert.NotNil(t, coordinator)

		confirmed := coordinator.IsTransmissionConfirmed(ocr2keepers.UpkeepPayload{
			ID:     "123",
			WorkID: "4",
			Upkeep: ocr2keepers.ConfiguredUpkeep{
				ID:   ocr2keepers.UpkeepIdentifier("4"),
				Type: 1,
			},
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 501,
			},
		})

		assert.True(t, confirmed)
	})

	t.Run("a key set to true is confirmed", func(t *testing.T) {
		events := &mockEvents{}
		encoder := &mockEncoder{
			AfterFn: func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
				return false, nil
			},
		}

		var buf bytes.Buffer
		coordinator := NewConditionalReportCoordinator(events, 1, log.New(&buf, "", 0), encoder)
		assert.NotNil(t, coordinator)

		coordinator.activeKeys.Set("4", true, config.DefaultCacheExpiration)

		confirmed := coordinator.IsTransmissionConfirmed(ocr2keepers.UpkeepPayload{
			ID:     "123",
			WorkID: "4",
			Upkeep: ocr2keepers.ConfiguredUpkeep{
				ID:   ocr2keepers.UpkeepIdentifier("4"),
				Type: 1,
			},
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 501,
			},
		})

		assert.True(t, confirmed)
	})
}

func TestConditionalReportCoordinator_PreProcess(t *testing.T) {
	t.Run("filters all non pending payloads", func(t *testing.T) {
		events := &mockEvents{}
		encoder := &mockEncoder{
			AfterFn: func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
				if string(a) == "99" {
					return false, errors.New("after boom")
				}
				return false, nil
			},
		}

		var buf bytes.Buffer
		coordinator := NewConditionalReportCoordinator(events, 1, log.New(&buf, "", 0), encoder)
		assert.NotNil(t, coordinator)

		coordinator.idBlocks.Set("123", idBlocker{
			CheckBlockNumber:    "99",
			TransmitBlockNumber: "100",
		}, config.DefaultCacheExpiration)

		filtered, err := coordinator.PreProcess(context.Background(), []ocr2keepers.UpkeepPayload{
			{
				WorkID: "123",
				Upkeep: ocr2keepers.ConfiguredUpkeep{
					ID: []byte("123"),
				},
				Trigger: ocr2keepers.Trigger{
					BlockNumber: 99,
				},
			},
			{
				WorkID: "456",
				Upkeep: ocr2keepers.ConfiguredUpkeep{
					ID: []byte("456"),
				},
				Trigger: ocr2keepers.Trigger{
					BlockNumber: 100,
				},
			},
			{
				WorkID: "789",
				Upkeep: ocr2keepers.ConfiguredUpkeep{
					ID: []byte("789"),
				},
				Trigger: ocr2keepers.Trigger{
					BlockNumber: 101,
				},
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, 2, len(filtered))
	})
}

func TestConditionalReportCoordinator_checkEvents(t *testing.T) {
	events := &mockEvents{
		EventsFn: func(ctx context.Context) ([]ocr2keepers.TransmitEvent, error) {
			return []ocr2keepers.TransmitEvent{
				{
					Confirmations: 1,
				},
				{
					Confirmations: 3,
					TransmitBlock: ocr2keepers.BlockKey("123"),
					Type:          ocr2keepers.PerformEvent,
					UpkeepID:      ocr2keepers.UpkeepIdentifier("10"),
					WorkID:        "10",
					CheckBlock:    ocr2keepers.BlockKey("123"),
				},
				{
					Confirmations: 3,
					TransmitBlock: ocr2keepers.BlockKey("124"),
					Type:          ocr2keepers.StaleReportEvent,
					UpkeepID:      ocr2keepers.UpkeepIdentifier("20"),
					WorkID:        "20",
					CheckBlock:    ocr2keepers.BlockKey("124"),
				},
				{
					Confirmations: 3,
					TransmitBlock: ocr2keepers.BlockKey("200"),
					Type:          ocr2keepers.StaleReportEvent,
					UpkeepID:      ocr2keepers.UpkeepIdentifier("30"),
					WorkID:        "30",
					CheckBlock:    ocr2keepers.BlockKey("200"),
				},
			}, nil
		},
	}
	encoder := &mockEncoder{
		AfterFn: func(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
			return false, nil
		},
		IncrementFn: func(key ocr2keepers.BlockKey) (ocr2keepers.BlockKey, error) {
			if string(key) == "200" {
				return key, errors.New("increment error")
			}
			return ocr2keepers.BlockKey("125"), nil
		},
	}

	var buf bytes.Buffer
	coordinator := NewConditionalReportCoordinator(events, 2, log.New(&buf, "", 0), encoder)
	assert.NotNil(t, coordinator)

	coordinator.activeKeys.Set("10", false, config.DefaultCacheExpiration)
	coordinator.activeKeys.Set("20", true, config.DefaultCacheExpiration)

	coordinator.idBlocks.Set("20", idBlocker{
		CheckBlockNumber:    "124",
		TransmitBlockNumber: "124",
	}, config.DefaultCacheExpiration)

	err := coordinator.checkEvents(context.Background())
	assert.Error(t, err)
	assert.Equal(t, "increment error", err.Error())
	assert.True(t, strings.Contains(buf.String(), "Skipping transmit event in transaction  as confirmations (1) is less than min confirmations (2)"))
	assert.True(t, strings.Contains(buf.String(), "Got a stale event for previously accepted key  in transaction  at block 124, with confirmations 3"))
	assert.True(t, strings.Contains(buf.String(), "Stale event found for key  in transaction  at block 123, with confirmations 3"))
}
