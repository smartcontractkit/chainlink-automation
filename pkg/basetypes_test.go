package ocr2keepers

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpkeepPayload_GenerateID(t *testing.T) {
	payload := NewUpkeepPayload(big.NewInt(111), 1, BlockKey("4"), Trigger{
		BlockNumber: 11,
		BlockHash:   "0x11111",
		Extension:   "extension111",
	}, []byte("check-data-111"))
	assert.Equal(t, "5ea7e17f2ddc9745517d2b67f851fed4", payload.ID)

	t.Run("empty payload id", func(t *testing.T) {
		payload = UpkeepPayload{}
		assert.Equal(t, "7bdc3a8f78dcd5f22bc469c4d766e4a9", payload.GenerateID())
	})
}
