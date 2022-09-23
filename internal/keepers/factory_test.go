package keepers

import (
	"testing"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/stretchr/testify/assert"
)

func TestNewReportingPluginFactory(t *testing.T) {
	f := NewReportingPluginFactory(nil, nil, nil, 30*time.Second)
	assert.NotNil(t, f)
}

func TestNewReportingPlugin(t *testing.T) {
	f := &keepersReportingFactory{}

	p, i, err := f.NewReportingPlugin(types.ReportingPluginConfig{})

	assert.NoError(t, err)
	assert.Equal(t, "keepers instance TODO: give instance a unique name", i.Name)
	assert.NotNil(t, p)
}
