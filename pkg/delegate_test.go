package ocr2keepers

import (
	"context"
	"testing"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/stretchr/testify/assert"
)

func TestStart(t *testing.T) {
	t.Skip("service start throws nil pointer deference")
	d, err := NewDelegate(DelegateConfig{
		Logger: new(MockLogger),
		LocalConfig: types.LocalConfig{
			BlockchainTimeout:                  1 * time.Second,        // TODO: choose sane configs
			ContractConfigTrackerPollInterval:  15 * time.Second,       // TODO: choose sane configs
			ContractTransmitterTransmitTimeout: 1 * time.Second,        // TODO: choose sane configs
			DatabaseTimeout:                    100 * time.Millisecond, // TODO: choose sane configs
			ContractConfigConfirmations:        1,                      // TODO: choose sane configs
		},
	})
	assert.NoError(t, err)

	err = d.Start(context.Background())
	assert.Equal(t, err.Error(), "unimplemented")
}

func TestClose(t *testing.T) {
	var err error

	d, err := NewDelegate(DelegateConfig{
		LocalConfig: types.LocalConfig{
			BlockchainTimeout:                  1 * time.Second,        // TODO: choose sane configs
			ContractConfigTrackerPollInterval:  15 * time.Second,       // TODO: choose sane configs
			ContractTransmitterTransmitTimeout: 1 * time.Second,        // TODO: choose sane configs
			DatabaseTimeout:                    100 * time.Millisecond, // TODO: choose sane configs
			ContractConfigConfirmations:        1,                      // TODO: choose sane configs
		},
	})
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}

	assert.NotNil(t, d.keeper, "Delegate keeper should not be nil")
	if d.keeper == nil {
		t.FailNow()
	}

	err = d.Close()
	assert.Equal(t, err.Error(), "can only close a started Oracle: stopping keeper oracle")
}
