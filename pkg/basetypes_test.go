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
	assert.Equal(t, "0a73a5fd0fc265416da897fa9d08509c336c847f80236389426ef0b95506912b", payload.ID)

	t.Run("empty payload id", func(t *testing.T) {
		payload = UpkeepPayload{}
		assert.Equal(t, "20c9c9e789a8e576ba9d58b1324869aefcd92545f80a5ee3834ac29b2531a8aa", payload.GenerateID())
	})
}
