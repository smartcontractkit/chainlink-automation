package build_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepers2 "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/hooks/build"
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
	t.Run("when the instruction store has the should coordinate block instruction, the observation gets updated with should coordinate block instruction", func(t *testing.T) {
		obs := &ocr2keepers.AutomationObservation{
			Instructions: []instructions.Instruction{},
			Metadata:     map[ocr2keepers.ObservationMetadataKey]interface{}{},
			Performable:  []ocr2keepers2.CheckResult{},
		}

		instructionStoreMap := map[instructions.Instruction]bool{}

		instructionStore := &mockInstructionStore{
			SetFn: func(i instructions.Instruction) {
				instructionStoreMap[i] = true
			},
			HasFn: func(i instructions.Instruction) bool {
				return instructionStoreMap[i]
			},
			DeleteFn: func(i instructions.Instruction) {
				delete(instructionStoreMap, i)
			},
		}

		mStore := store.NewMetadata(nil)
		hook := build.NewCoordinateBlockHook(instructionStore, mStore)

		// before the hook is run
		instructionStore.Set(instructions.ShouldCoordinateBlock)
		assert.Equal(t, 0, len(obs.Instructions))

		// run the hook and test results
		assert.NoError(t, hook.RunHook(obs))
		assert.Equal(t, 1, len(obs.Instructions))
		assert.Equal(t, obs.Instructions[0], instructions.ShouldCoordinateBlock)
	})

	t.Run("when the instruction store has the do coordinate block instruction, the observation gets updated with the block history", func(t *testing.T) {
		obs := &ocr2keepers.AutomationObservation{
			Instructions: []instructions.Instruction{},
			Metadata:     map[ocr2keepers.ObservationMetadataKey]interface{}{},
			Performable:  []ocr2keepers2.CheckResult{},
		}

		instructionStoreMap := map[instructions.Instruction]bool{}

		instructionStore := &mockInstructionStore{
			SetFn: func(i instructions.Instruction) {
				instructionStoreMap[i] = true
			},
			HasFn: func(i instructions.Instruction) bool {
				return instructionStoreMap[i]
			},
			DeleteFn: func(i instructions.Instruction) {
				delete(instructionStoreMap, i)
			},
		}

		mStore := store.NewMetadata(nil)
		hook := build.NewCoordinateBlockHook(instructionStore, mStore)
		blockHistory := ocr2keepers2.BlockHistory{
			ocr2keepers2.BlockKey("3"),
			ocr2keepers2.BlockKey("2"),
			ocr2keepers2.BlockKey("1"),
		}

		mStore.Set(store.BlockHistoryMetadata, blockHistory)
		instructionStore.Set(instructions.DoCoordinateBlock)

		// run the hook and test the results
		assert.NoError(t, hook.RunHook(obs))
		assert.Equal(t, obs.Metadata[ocr2keepers.BlockHistoryObservationKey], blockHistory)
	})
}
