package keepers

import (
	"fmt"
	"testing"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/stretchr/testify/assert"
)

func TestNewReportingPluginFactory(t *testing.T) {
	f := NewReportingPluginFactory(nil, nil, nil, ReportingFactoryConfig{})
	assert.NotNil(t, f)
}

func TestNewReportingPlugin(t *testing.T) {
	f := &keepersReportingFactory{
		config: ReportingFactoryConfig{
			CacheExpiration:       30 * time.Second,
			CacheEvictionInterval: 5 * time.Second,
			MaxServiceWorkers:     1,
			ServiceQueueLength:    10,
		},
	}

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
