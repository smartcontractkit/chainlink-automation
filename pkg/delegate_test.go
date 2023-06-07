package ocr2keepers

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/ocr2keepers/pkg/mocks"
)

func TestStart(t *testing.T) {
	t.Run("fails to create the delegate with an empty config", func(t *testing.T) {
		logger := mocks.NewMockLogger(t)
		logger.On("Debug", mock.Anything, mock.Anything).Once()

		_, err := NewDelegate(DelegateConfig{
			Logger: logger,
		})

		assert.Equal(t, err.Error(), "bad local config while creating new oracle: blockchain timeout must be between 1s and 20s, but is currently 0s; contract config tracker poll interval must be between 15s and 2m0s, but is currently 0s; contract transmitter transmit timeout must be between 1s and 1m0s, but is currently 0s; database timeout must be between 100ms and 10s, but is currently 0s; contract config block-depth confirmation threshold must be between 1 and 100, but is currently 0: failed to create new OCR oracle")
	})

	t.Run("creates the delegate with the provided config", func(t *testing.T) {
		logger := mocks.NewMockLogger(t)
		logger.On("Debug", mock.Anything, mock.Anything).Maybe()

		_, err := NewDelegate(DelegateConfig{
			Logger: logger,
			LocalConfig: types.LocalConfig{
				BlockchainTimeout:                  1 * time.Second,
				ContractConfigTrackerPollInterval:  15 * time.Second,
				ContractTransmitterTransmitTimeout: 1 * time.Second,
				DatabaseTimeout:                    100 * time.Millisecond,
				ContractConfigConfirmations:        1,
			},
			CacheExpiration:       1 * time.Second,
			CacheEvictionInterval: 1 * time.Second,
			MaxServiceWorkers:     1,
			ServiceQueueLength:    1,
		})

		assert.NoError(t, err)
	})

	t.Run("running the delegate runs the wrapped keeper", func(t *testing.T) {
		var started bool
		delegate := &Delegate{
			keeper: &mockKeeper{
				StartFn: func() error {
					started = true
					return nil
				},
			},
		}
		err := delegate.Start(context.Background())
		assert.NoError(t, err)
		assert.True(t, started)
	})

	t.Run("starting the delegate errors when the keeper errors on start", func(t *testing.T) {
		var started bool
		delegate := &Delegate{
			keeper: &mockKeeper{
				StartFn: func() error {
					started = true
					return errors.New("failed to start")
				},
			},
		}
		err := delegate.Start(context.Background())
		assert.Error(t, err)
		assert.True(t, started)
	})
}

func TestClose(t *testing.T) {
	t.Run("a not yet started oracle fails to close", func(t *testing.T) {
		var mockLogger = mocks.NewMockLogger(t)
		mockLogger.On("Debug", mock.Anything, mock.Anything).Return()

		d, err := NewDelegate(DelegateConfig{
			Logger: mockLogger,
			LocalConfig: types.LocalConfig{
				BlockchainTimeout:                  1 * time.Second,
				ContractConfigTrackerPollInterval:  15 * time.Second,
				ContractTransmitterTransmitTimeout: 1 * time.Second,
				DatabaseTimeout:                    100 * time.Millisecond,
				ContractConfigConfirmations:        1,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, d.keeper, "Delegate keeper should not be nil")

		err = d.Close()
		assert.Equal(t, err.Error(), "can only close a started Oracle: failed to close keeper oracle")
	})

	t.Run("closing the delegate also closes the wrapped keeper", func(t *testing.T) {
		var closed bool
		delegate := &Delegate{
			keeper: &mockKeeper{
				CloseFn: func() error {
					closed = true
					return nil
				},
			},
		}
		err := delegate.Close()
		assert.NoError(t, err)
		assert.True(t, closed)
	})
}

type mockKeeper struct {
	StartFn func() error
	CloseFn func() error
}

func (k *mockKeeper) Start() error {
	return k.StartFn()
}

func (k *mockKeeper) Close() error {
	return k.CloseFn()
}
