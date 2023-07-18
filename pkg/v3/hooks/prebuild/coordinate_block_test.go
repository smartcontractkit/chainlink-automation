package prebuild

import (
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepers2 "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/instructions"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
)

type mockInstructionStore struct {
	SetFn    func(instructions.Instruction)
	HasFn    func(instructions.Instruction) bool
	DeleteFn func(instructions.Instruction)
}

func (s *mockInstructionStore) Set(i instructions.Instruction) {
	s.SetFn(i)
}

func (s *mockInstructionStore) Has(i instructions.Instruction) bool {
	return s.HasFn(i)
}

func (s *mockInstructionStore) Delete(i instructions.Instruction) {
	s.DeleteFn(i)
}

func TestNewCoordinateBlockHook(t *testing.T) {
	t.Run("pre build hook adds DoCoordinateBlock to the instruction store", func(t *testing.T) {
		storeState := make(map[instructions.Instruction]bool)
		iStore := &mockInstructionStore{
			SetFn: func(i instructions.Instruction) {
				storeState[i] = true
			},
			DeleteFn: func(i instructions.Instruction) {
				delete(storeState, i)
			},
			HasFn: func(i instructions.Instruction) bool {
				val, ok := storeState[i]
				if !ok {
					return false
				}
				return val
			},
		}

		outcome := ocr2keepers.AutomationOutcome{
			Instructions: []instructions.Instruction{
				instructions.ShouldCoordinateBlock,
			},
			Metadata: map[ocr2keepers.OutcomeMetadataKey]interface{}{},
		}

		iStore.Set(instructions.ShouldCoordinateBlock)

		mStore := store.NewMetadata(nil)

		hook := NewCoordinateBlockHook(iStore, mStore)

		err := hook.RunHook(outcome)
		assert.NoError(t, err)

		assert.True(t, iStore.Has(instructions.DoCoordinateBlock))
		assert.False(t, iStore.Has(instructions.ShouldCoordinateBlock))

		_, ok := mStore.Get(store.CoordinatedBlockMetadata)
		assert.False(t, ok)
	})

	t.Run("pre build hook adds DoCoordinateBlock to the instruction store", func(t *testing.T) {
		storeState := make(map[instructions.Instruction]bool)
		iStore := &mockInstructionStore{
			SetFn: func(i instructions.Instruction) {
				storeState[i] = true
			},
			DeleteFn: func(i instructions.Instruction) {
				delete(storeState, i)
			},
			HasFn: func(i instructions.Instruction) bool {
				val, ok := storeState[i]
				if !ok {
					return false
				}
				return val
			},
		}

		blockKey := ocr2keepers2.BlockKey("testBlockKey")

		outcome := ocr2keepers.AutomationOutcome{
			Instructions: []instructions.Instruction{
				instructions.DoCoordinateBlock,
			},
			Metadata: map[ocr2keepers.OutcomeMetadataKey]interface{}{
				ocr2keepers.CoordinatedBlockOutcomeKey: blockKey,
			},
		}

		iStore.Set(instructions.DoCoordinateBlock)

		mStore := store.NewMetadata(nil)

		hook := NewCoordinateBlockHook(iStore, mStore)

		assert.NoError(t, hook.RunHook(outcome), "no error from running hook")

		assert.False(t, iStore.Has(instructions.DoCoordinateBlock), "no instructions should exist")
		assert.False(t, iStore.Has(instructions.ShouldCoordinateBlock), "no instructions should exist")

		v, ok := mStore.Get(store.CoordinatedBlockMetadata)
		assert.True(t, ok, "coordinated block should be in metadata store")
		assert.Equal(t, v, blockKey, "value for coordinated block should be from outcome")
	})

	t.Run("an error is returned when the metadata stores the wrong data type for coordinated block", func(t *testing.T) {
		storeState := make(map[instructions.Instruction]bool)
		iStore := &mockInstructionStore{
			SetFn: func(i instructions.Instruction) {
				storeState[i] = true
			},
			DeleteFn: func(i instructions.Instruction) {
				delete(storeState, i)
			},
			HasFn: func(i instructions.Instruction) bool {
				val, ok := storeState[i]
				if !ok {
					return false
				}
				return val
			},
		}

		outcome := ocr2keepers.AutomationOutcome{
			Instructions: []instructions.Instruction{
				instructions.ShouldCoordinateBlock,
			},
			Metadata: map[ocr2keepers.OutcomeMetadataKey]interface{}{
				ocr2keepers.CoordinatedBlockOutcomeKey: "not a block",
			},
		}

		iStore.Set(instructions.ShouldCoordinateBlock)

		hook := NewCoordinateBlockHook(iStore, store.NewMetadata(nil))

		err := hook.RunHook(outcome)
		assert.Error(t, err)

		assert.True(t, iStore.Has(instructions.DoCoordinateBlock))
		assert.False(t, iStore.Has(instructions.ShouldCoordinateBlock))
	})
}
