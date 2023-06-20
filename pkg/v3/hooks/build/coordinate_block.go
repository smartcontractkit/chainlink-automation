package build

import (
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/flows"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/instruction"
)

const (
	ShouldCoordinateBlock instruction.Instruction = "should coordinate block"
	DoCoordinateBlock     instruction.Instruction = "do coordinate block"
)

type BuildHook func(*ocr2keepers.AutomationObservation, instruction.InstructionStore, ocr2keepers.MetadataStore, flows.ResultStore) error

type coordinateBlockHook struct{}

func NewCoordinateBlockHook() *coordinateBlockHook {
	return &coordinateBlockHook{}
}

func (h *coordinateBlockHook) RunHook(obs *ocr2keepers.AutomationObservation, instructionStore instruction.InstructionStore, metadataStore ocr2keepers.MetadataStore, resultStore flows.ResultStore) error {
	if instructionStore.Has(ShouldCoordinateBlock) {
		obs.Instructions = append(obs.Instructions, ShouldCoordinateBlock)
	} else if instructionStore.Has(DoCoordinateBlock) {
		obs.Metadata["blockHistory"] = metadataStore.GetBlockHistory()
	}

	return nil
}
