package build_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	ocr2keepers2 "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/hooks/build"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/instruction"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
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

type mockSubscriber struct {
	SubscribeFn   func() (int, chan ocr2keepers2.BlockHistory, error)
	UnsubscribeFn func(id int) error
}

func (s *mockSubscriber) Subscribe() (int, chan ocr2keepers2.BlockHistory, error) {
	return s.SubscribeFn()
}

func (s *mockSubscriber) Unsubscribe(id int) error {
	return s.UnsubscribeFn(id)
}

func TestNewCoordinateBlockHook(t *testing.T) {
	t.Run("when the instruction store has the should coordinate block instruction, the observation gets updated with should coordinate block instruction", func(t *testing.T) {
		hook := build.NewCoordinateBlockHook()
		obs := &ocr2keepers.AutomationObservation{
			Instructions: []instruction.Instruction{},
			Metadata:     map[string]interface{}{},
			Performable:  []ocr2keepers2.CheckResult{},
		}

		instructionStoreMap := map[instruction.Instruction]bool{}

		instructionStore := &mockInstructionStore{
			SetFn: func(i instruction.Instruction) {
				instructionStoreMap[i] = true
			},
			HasFn: func(i instruction.Instruction) bool {
				return instructionStoreMap[i]
			},
			DeleteFn: func(i instruction.Instruction) {
				delete(instructionStoreMap, i)
			},
		}

		ch := make(chan ocr2keepers2.BlockHistory)

		subscriber := &mockSubscriber{
			SubscribeFn: func() (int, chan ocr2keepers2.BlockHistory, error) {
				return 0, ch, nil
			},
			UnsubscribeFn: func(id int) error {
				return nil
			},
		}

		ticker, err := tickers.NewBlockTicker(subscriber)
		assert.NoError(t, err)

		metadataStore := ocr2keepers.NewMetadataStore(ticker)

		instructionStore.Set(build.ShouldCoordinateBlock)
		assert.Equal(t, 0, len(obs.Instructions))
		err = hook.RunHook(obs, instructionStore, metadataStore, nil)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(obs.Instructions))
		assert.Equal(t, obs.Instructions[0], build.ShouldCoordinateBlock)
	})

	t.Run("when the instruction store has the do coordinate block instruction, the observation gets updated with the block history", func(t *testing.T) {
		hook := build.NewCoordinateBlockHook()
		obs := &ocr2keepers.AutomationObservation{
			Instructions: []instruction.Instruction{},
			Metadata:     map[string]interface{}{},
			Performable:  []ocr2keepers2.CheckResult{},
		}

		instructionStoreMap := map[instruction.Instruction]bool{}

		instructionStore := &mockInstructionStore{
			SetFn: func(i instruction.Instruction) {
				instructionStoreMap[i] = true
			},
			HasFn: func(i instruction.Instruction) bool {
				return instructionStoreMap[i]
			},
			DeleteFn: func(i instruction.Instruction) {
				delete(instructionStoreMap, i)
			},
		}

		ch := make(chan ocr2keepers2.BlockHistory)

		subscriber := &mockSubscriber{
			SubscribeFn: func() (int, chan ocr2keepers2.BlockHistory, error) {
				return 0, ch, nil
			},
			UnsubscribeFn: func(id int) error {
				return nil
			},
		}

		ticker, err := tickers.NewBlockTicker(subscriber)
		assert.NoError(t, err)

		metadataStore := ocr2keepers.NewMetadataStore(ticker)

		blockHistory := ocr2keepers2.BlockHistory{
			ocr2keepers2.BlockKey{Block: 1},
			ocr2keepers2.BlockKey{Block: 2},
			ocr2keepers2.BlockKey{Block: 3},
		}

		go func() {
			err = metadataStore.Start()
			assert.NoError(t, err)
		}()
		go func() {
			err = ticker.Start(context.Background())
			assert.NoError(t, err)
		}()

		ch <- blockHistory

		time.Sleep(1 * time.Second)

		instructionStore.Set(build.DoCoordinateBlock)
		err = hook.RunHook(obs, instructionStore, metadataStore, nil)
		assert.NoError(t, err)
		assert.Equal(t, obs.Metadata["blockHistory"], blockHistory)
	})
}
