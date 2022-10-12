package keepers

import (
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewReportingPluginFactory(t *testing.T) {
	f := NewReportingPluginFactory(nil, nil, nil, nil, ReportingFactoryConfig{})
	assert.NotNil(t, f)
}

func TestNewReportingPlugin(t *testing.T) {
	mp := new(MockedPerformLogProvider)

	f := &keepersReportingFactory{
		registry: new(MockedRegistry),
		encoder:  new(MockedReportEncoder),
		perfLogs: mp,
		logger:   log.New(io.Discard, "test", 0),
		config: ReportingFactoryConfig{
			CacheExpiration:       30 * time.Second,
			CacheEvictionInterval: 5 * time.Second,
			MaxServiceWorkers:     1,
			ServiceQueueLength:    10,
		},
	}

	mp.Mock.On("PerformLogs", mock.Anything).Return([]ktypes.PerformLog{}, nil).Maybe()

	digest := [32]byte{}
	digestStr := fmt.Sprintf("%32s", "test")
	copy(digest[:], []byte(digestStr)[:32])

	p, i, err := f.NewReportingPlugin(types.ReportingPluginConfig{
		ConfigDigest:   digest,
		OracleID:       1,
		N:              5,
		F:              2,
		OffchainConfig: []byte{},
	})

	assert.NoError(t, err)
	assert.Equal(t, "Oracle 1: Keepers Plugin Instance w/ Digest '2020202020202020202020202020202020202020202020202020202074657374'", i.Name)
	assert.NotNil(t, p)
}
