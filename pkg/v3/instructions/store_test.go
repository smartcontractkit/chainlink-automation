package instructions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstructionStore(t *testing.T) {
	store := NewStore()

	store.Set(ShouldCoordinateBlock)

	assert.Equal(t, true, store.Has(ShouldCoordinateBlock), "store should have key: '%s'", ShouldCoordinateBlock)
	assert.Equal(t, false, store.Has(DoCoordinateBlock), "store should not have key: '%s'", DoCoordinateBlock)

	store.Set(ShouldCoordinateBlock)

	assert.Equal(t, true, store.Has(ShouldCoordinateBlock), "key set should be idempotent")

	store.Delete(DoCoordinateBlock)
	assert.Equal(t, false, store.Has(DoCoordinateBlock), "key delete should be idempotent")

	store.Delete(ShouldCoordinateBlock)
	assert.Equal(t, false, store.Has(ShouldCoordinateBlock), "store should not have key: '%s'", ShouldCoordinateBlock)
}
