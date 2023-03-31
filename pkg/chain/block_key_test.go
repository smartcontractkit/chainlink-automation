package chain

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlockKey(t *testing.T) {
	t.Run("creates a new block key", func(t *testing.T) {
		key := BlockKey("123")
		assert.Equal(t, key.String(), "123")
		keyInt, ok := key.BigInt()
		assert.True(t, ok)
		assert.Equal(t, keyInt, big.NewInt(123))
		nextKey, err := key.Next()
		assert.Nil(t, err)
		assert.Equal(t, nextKey, BlockKey("124"))
	})

	t.Run("calling next on a malformed block key causes an error", func(t *testing.T) {
		key := BlockKey("abc")
		nextKey, err := key.Next()
		assert.NotNil(t, err)
		assert.Equal(t, nextKey, BlockKey(""))
	})

	t.Run("after errors if the first block key is not parsable", func(t *testing.T) {
		keyA := BlockKey("abc")
		keyB := BlockKey("123")
		after, err := keyA.After(keyB)
		assert.False(t, after)
		assert.Equal(t, err, ErrBlockKeyNotParsable)
	})

	t.Run("after errors if the second block key is not parsable", func(t *testing.T) {
		keyA := BlockKey("123")
		keyB := BlockKey("abc")
		after, err := keyA.After(keyB)
		assert.False(t, after)
		assert.Equal(t, err, ErrBlockKeyNotParsable)
	})

	t.Run("calculates that an earlier block is not after a later block", func(t *testing.T) {
		keyA := BlockKey("123")
		keyB := BlockKey("124")
		after, err := keyA.After(keyB)
		assert.False(t, after)
		assert.Nil(t, err)
	})

	t.Run("calculates that a later block is after an earlier block", func(t *testing.T) {
		keyA := BlockKey("124")
		keyB := BlockKey("123")
		after, err := keyA.After(keyB)
		assert.True(t, after)
		assert.Nil(t, err)
	})
}
