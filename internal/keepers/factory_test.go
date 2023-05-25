package keepers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/smartcontractkit/ocr2keepers/pkg/types/mocks"
)

func TestNewReportingPluginFactory(t *testing.T) {
	t.Run("a new reporting plugin factory is created without dependencies", func(t *testing.T) {
		f := NewReportingPluginFactory(
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			ReportingFactoryConfig{},
		)
		assert.NotNil(t, f)
	})
}

func TestNewReportingPlugin(t *testing.T) {
	t.Run("fails to decode the offchain config", func(t *testing.T) {
		mp := mocks.NewPerformLogProvider(t)
		hs := mocks.NewHeadSubscriber(t)

		f := &keepersReportingFactory{
			registry:       mocks.NewRegistry(t),
			encoder:        mocks.NewReportEncoder(t),
			headSubscriber: hs,
			perfLogs:       mp,
			logger:         log.New(io.Discard, "test", 0),
			config: ReportingFactoryConfig{
				CacheExpiration:       30 * time.Second,
				CacheEvictionInterval: 5 * time.Second,
				MaxServiceWorkers:     1,
				ServiceQueueLength:    10,
			},
		}

		mp.Mock.On("PerformLogs", mock.Anything).
			Return([]ktypes.PerformLog{}, nil).
			Maybe()
		mp.Mock.On("StaleReportLogs", mock.Anything).
			Return([]ktypes.StaleReportLog{}, nil).
			Maybe()

		digest := [32]byte{}
		digestStr := fmt.Sprintf("%32s", "test")
		copy(digest[:], []byte(digestStr)[:32])

		_, _, err := f.NewReportingPlugin(types.ReportingPluginConfig{
			ConfigDigest:   digest,
			OracleID:       1,
			N:              5,
			F:              2,
			OffchainConfig: []byte("invalid json"),
		})

		assert.NotNil(t, err)
	})

	t.Run("fails to create a new reporting plugin due to malformed TargetProbability", func(t *testing.T) {
		mp := mocks.NewPerformLogProvider(t)
		hs := mocks.NewHeadSubscriber(t)

		f := &keepersReportingFactory{
			registry:       mocks.NewRegistry(t),
			encoder:        mocks.NewReportEncoder(t),
			headSubscriber: hs,
			perfLogs:       mp,
			logger:         log.New(io.Discard, "test", 0),
			config: ReportingFactoryConfig{
				CacheExpiration:       30 * time.Second,
				CacheEvictionInterval: 5 * time.Second,
				MaxServiceWorkers:     1,
				ServiceQueueLength:    10,
			},
		}

		mp.Mock.On("PerformLogs", mock.Anything).
			Return([]ktypes.PerformLog{}, nil).
			Maybe()
		mp.Mock.On("StaleReportLogs", mock.Anything).
			Return([]ktypes.StaleReportLog{}, nil).
			Maybe()

		digest := [32]byte{}
		digestStr := fmt.Sprintf("%32s", "test")
		copy(digest[:], []byte(digestStr)[:32])

		offchainConfig, err := json.Marshal(ktypes.OffchainConfig{
			GasLimitPerReport:    500000,
			GasOverheadPerUpkeep: 300000,
			TargetProbability:    "invalid",
		})
		require.NoError(t, err)

		_, _, err = f.NewReportingPlugin(types.ReportingPluginConfig{
			ConfigDigest:   digest,
			OracleID:       1,
			N:              5,
			F:              2,
			OffchainConfig: offchainConfig,
		})

		assert.NotNil(t, err)
	})

	t.Run("fails to create a new reporting plugin due to invalid TargetProbability", func(t *testing.T) {
		mp := mocks.NewPerformLogProvider(t)
		hs := mocks.NewHeadSubscriber(t)

		f := &keepersReportingFactory{
			registry:       mocks.NewRegistry(t),
			encoder:        mocks.NewReportEncoder(t),
			headSubscriber: hs,
			perfLogs:       mp,
			logger:         log.New(io.Discard, "test", 0),
			config: ReportingFactoryConfig{
				CacheExpiration:       30 * time.Second,
				CacheEvictionInterval: 5 * time.Second,
				MaxServiceWorkers:     1,
				ServiceQueueLength:    10,
			},
		}

		mp.Mock.On("PerformLogs", mock.Anything).
			Return([]ktypes.PerformLog{}, nil).
			Maybe()
		mp.Mock.On("StaleReportLogs", mock.Anything).
			Return([]ktypes.StaleReportLog{}, nil).
			Maybe()

		digest := [32]byte{}
		digestStr := fmt.Sprintf("%32s", "test")
		copy(digest[:], []byte(digestStr)[:32])

		offchainConfig, err := json.Marshal(ktypes.OffchainConfig{
			GasLimitPerReport:    500000,
			GasOverheadPerUpkeep: 300000,
			TargetProbability:    "2.0",
		})
		require.NoError(t, err)

		_, _, err = f.NewReportingPlugin(types.ReportingPluginConfig{
			ConfigDigest:   digest,
			OracleID:       1,
			N:              5,
			F:              2,
			OffchainConfig: offchainConfig,
		})

		assert.NotNil(t, err)
	})

	t.Run("creates a new reporting plugin", func(t *testing.T) {
		mp := mocks.NewPerformLogProvider(t)
		hs := mocks.NewHeadSubscriber(t)

		f := &keepersReportingFactory{
			registry:       mocks.NewRegistry(t),
			encoder:        mocks.NewReportEncoder(t),
			headSubscriber: hs,
			perfLogs:       mp,
			logger:         log.New(io.Discard, "test", 0),
			config: ReportingFactoryConfig{
				CacheExpiration:       30 * time.Second,
				CacheEvictionInterval: 5 * time.Second,
				MaxServiceWorkers:     1,
				ServiceQueueLength:    10,
			},
		}

		mp.Mock.On("PerformLogs", mock.Anything).
			Return([]ktypes.PerformLog{}, nil).
			Maybe()
		mp.Mock.On("StaleReportLogs", mock.Anything).
			Return([]ktypes.StaleReportLog{}, nil).
			Maybe()
		hs.Mock.On("HeadTicker").Return(make(chan ktypes.BlockKey)).Maybe()

		digest := [32]byte{}
		digestStr := fmt.Sprintf("%32s", "test")
		copy(digest[:], []byte(digestStr)[:32])

		offchainConfig, err := json.Marshal(ktypes.OffchainConfig{
			GasLimitPerReport:    500000,
			GasOverheadPerUpkeep: 300000,
		})
		require.NoError(t, err)

		p, i, err := f.NewReportingPlugin(types.ReportingPluginConfig{
			ConfigDigest:   digest,
			OracleID:       1,
			N:              5,
			F:              2,
			OffchainConfig: offchainConfig,
		})

		// provide enough time for all start functions to be called
		<-time.After(100 * time.Millisecond)

		assert.NoError(t, err)
		assert.Equal(t, "Oracle 1: Keepers Plugin Instance w/ Digest '2020202020202020202020202020202020202020202020202020202074657374'", i.Name)
		assert.NotNil(t, p)

		hs.AssertExpectations(t)
		mp.AssertExpectations(t)
	})
}
