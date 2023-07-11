package prebuild

import (
	"errors"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/instruction"
)

const (
	ShouldCoordinateBlock instruction.Instruction = "should coordinate block"
	DoCoordinateBlock     instruction.Instruction = "do coordinate block"
)

type coordinatedBlockSetter interface {
	SetBlock(key ocr2keepers.BlockKey)
}

type coordinateBlockHook struct {
	instructionStore instruction.InstructionStore
	blockSetter      coordinatedBlockSetter
}

func NewCoordinateBlockHook(instructionStore instruction.InstructionStore, blockSetter coordinatedBlockSetter) *coordinateBlockHook {
	return &coordinateBlockHook{
		instructionStore: instructionStore,
		blockSetter:      blockSetter,
	}
}

func (h *coordinateBlockHook) RunHook(outcome ocr2keepersv3.AutomationOutcome) error {
	for _, instruction := range outcome.Instructions {
		if instruction == ShouldCoordinateBlock {
			h.instructionStore.Set(DoCoordinateBlock)
			h.instructionStore.Delete(ShouldCoordinateBlock)
			break
		}
	}

loop:
	for k, v := range outcome.Metadata {
		if k == "coordinatedBlock" {
			switch t := v.(type) {
			case ocr2keepers.BlockKey:
				h.blockSetter.SetBlock(t)
				break loop
			default:
				return errors.New("coordinated block is unexpected type")
			}
		}
	}

	return nil
}
