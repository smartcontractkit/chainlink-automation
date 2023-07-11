package prebuild

import (
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepers2 "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/instruction"
)

type mockInstructionStore struct {
	SetFn    func(instruction.Instruction)
	HasFn    func(instruction.Instruction) bool
	DeleteFn func(instruction.Instruction)
}

func (s *mockInstructionStore) Set(i instruction.Instruction) {
	s.SetFn(i)
}

func (s *mockInstructionStore) Has(i instruction.Instruction) bool {
	return s.HasFn(i)
}

func (s *mockInstructionStore) Delete(i instruction.Instruction) {
	s.DeleteFn(i)
}

type mockBlockSetter struct {
	SetBlockFn func(key ocr2keepers2.BlockKey)
}

func (s *mockBlockSetter) SetBlock(key ocr2keepers2.BlockKey) {
	s.SetBlockFn(key)
}

func TestNewCoordinateBlockHook(t *testing.T) {
	t.Run("pre build hook adds DoCoordinateBlock to the instruction store", func(t *testing.T) {
		storeState := make(map[instruction.Instruction]bool)
		store := &mockInstructionStore{
			SetFn: func(i instruction.Instruction) {
				storeState[i] = true
			},
			DeleteFn: func(i instruction.Instruction) {
				delete(storeState, i)
			},
			HasFn: func(i instruction.Instruction) bool {
				val, ok := storeState[i]
				if !ok {
					return false
				}
				return val
			},
		}

		blockKey := ocr2keepers2.BlockKey("testBlockKey")

		outcome := ocr2keepers.AutomationOutcome{
			Instructions: []instruction.Instruction{
				ShouldCoordinateBlock,
			},
			Metadata: map[string]interface{}{
				"coordinatedBlock": blockKey,
			},
		}

		store.Set(ShouldCoordinateBlock)

		var setBlock ocr2keepers2.BlockKey
		blockSetter := &mockBlockSetter{
			SetBlockFn: func(key ocr2keepers2.BlockKey) {
				setBlock = key
			},
		}

		hook := NewCoordinateBlockHook(store, blockSetter)

		err := hook.RunHook(outcome)
		assert.NoError(t, err)

		assert.True(t, store.Has(DoCoordinateBlock))
		assert.False(t, store.Has(ShouldCoordinateBlock))

		assert.Equal(t, setBlock, blockKey)
	})

	t.Run("an error is returned when the metadata stores the wrong data type for coordinated block", func(t *testing.T) {
		storeState := make(map[instruction.Instruction]bool)
		store := &mockInstructionStore{
			SetFn: func(i instruction.Instruction) {
				storeState[i] = true
			},
			DeleteFn: func(i instruction.Instruction) {
				delete(storeState, i)
			},
			HasFn: func(i instruction.Instruction) bool {
				val, ok := storeState[i]
				if !ok {
					return false
				}
				return val
			},
		}

		outcome := ocr2keepers.AutomationOutcome{
			Instructions: []instruction.Instruction{
				ShouldCoordinateBlock,
			},
			Metadata: map[string]interface{}{
				"coordinatedBlock": "not a block",
			},
		}

		store.Set(ShouldCoordinateBlock)

		blockSetter := &mockBlockSetter{}

		hook := NewCoordinateBlockHook(store, blockSetter)

		err := hook.RunHook(outcome)
		assert.Error(t, err)

		assert.True(t, store.Has(DoCoordinateBlock))
		assert.False(t, store.Has(ShouldCoordinateBlock))
	})
}
