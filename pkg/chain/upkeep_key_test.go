package chain

import (
	"math/big"
	"testing"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestNewUpkeepKey(t *testing.T) {
	t.Run("creates a new upkeep key", func(t *testing.T) {
		key := NewUpkeepKey(big.NewInt(1), big.NewInt(123))
		assert.Equal(t, key.String(), "1|123")

		blockKey, upkeepID, err := key.BlockKeyAndUpkeepID()
		assert.Nil(t, err)
		assert.Equal(t, blockKey, BlockKey("1"))
		assert.Equal(t, upkeepID, types.UpkeepIdentifier("123"))
	})

	t.Run("creates a new upkeep key from block and ID", func(t *testing.T) {
		key := NewUpkeepKeyFromBlockAndID(BlockKey("1"), types.UpkeepIdentifier("123"))
		assert.Equal(t, key.String(), "1|123")

		blockKey, upkeepID, err := key.BlockKeyAndUpkeepID()
		assert.Nil(t, err)
		assert.Equal(t, blockKey, BlockKey("1"))
		assert.Equal(t, upkeepID, types.UpkeepIdentifier("123"))
	})

	t.Run("fetching the block key and upkeep ID from a malformed upkeep key causes and error", func(t *testing.T) {
		key := UpkeepKey("111")

		_, _, err := key.BlockKeyAndUpkeepID()
		assert.NotNil(t, err)
	})
}
