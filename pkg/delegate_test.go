package ocr2keepers

import (
	"context"
	"testing"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func TestStart(t *testing.T) {
	t.Skip("service start throws nil pointer deference")
	d, err := NewDelegate(DelegateConfig{
		Logger: ktypes.NewMockLogger(t),
		LocalConfig: types.LocalConfig{
			BlockchainTimeout:                  1 * time.Second,
			ContractConfigTrackerPollInterval:  15 * time.Second,
			ContractTransmitterTransmitTimeout: 1 * time.Second,
			DatabaseTimeout:                    100 * time.Millisecond,
			ContractConfigConfirmations:        1,
		},
	})
	assert.NoError(t, err)

	err = d.Start(context.Background())
	assert.Equal(t, err.Error(), "unimplemented")
}

func TestClose(t *testing.T) {
	var err error
	var mockLogger = ktypes.NewMockLogger(t)
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
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}

	assert.NotNil(t, d.keeper, "Delegate keeper should not be nil")
	if d.keeper == nil {
		t.FailNow()
	}

	err = d.Close()
	assert.Equal(t, err.Error(), "can only close a started Oracle: failed to close keeper oracle")
}
