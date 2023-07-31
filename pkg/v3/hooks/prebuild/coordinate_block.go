package prebuild

import (
	"errors"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/instructions"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
)

type instructionSource interface {
	Set(instructions.Instruction)
	Delete(instructions.Instruction)
}

type coordinateBlockHook struct {
	instructionStore instructionSource
	metadata         *store.Metadata
}

func NewCoordinateBlockHook(
	instructionStore instructionSource,
	metadata *store.Metadata,
) *coordinateBlockHook {
	return &coordinateBlockHook{
		instructionStore: instructionStore,
		metadata:         metadata,
	}
}

func (h *coordinateBlockHook) RunHook(outcome ocr2keepersv3.AutomationOutcome) error {
	for _, in := range outcome.Instructions {
		if in == instructions.ShouldCoordinateBlock {
			h.instructionStore.Set(instructions.DoCoordinateBlock)
			h.instructionStore.Delete(instructions.ShouldCoordinateBlock)

			break
		}
	}

loop:
	for k, v := range outcome.Metadata {
		if k == ocr2keepersv3.CoordinatedBlockOutcomeKey {
			switch t := v.(type) {
			case ocr2keepers.BlockKey:
				// since the block key exists, reset the instructions and save
				// the latest coordinated block
				h.instructionStore.Delete(instructions.DoCoordinateBlock)
				h.instructionStore.Set(instructions.ShouldCoordinateBlock)
				h.metadata.Set(store.CoordinatedBlockMetadata, t)

				break loop
			default:
				return errors.New("coordinated block is unexpected type")
			}
		}
	}

	return nil
}
