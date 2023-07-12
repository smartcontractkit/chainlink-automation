package build

import (
	"fmt"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/instructions"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
)

type instructionSource interface {
	Has(instructions.Instruction) bool
}

type coordinateBlockHook struct {
	instructions instructionSource
	metadata     *store.Metadata
}

// NewCoordinateBlockHook creates a new build hook that adds the latest block
// history to the observation metadata
func NewCoordinateBlockHook(
	inst instructionSource,
	metadata *store.Metadata,
) *coordinateBlockHook {
	return &coordinateBlockHook{
		instructions: inst,
		metadata:     metadata,
	}
}

func (h *coordinateBlockHook) RunHook(obs *ocr2keepers.AutomationObservation) error {
	// node instructions indicate that the node should broadcast that it wants
	// to coordinate on the latest block
	if h.instructions.Has(instructions.ShouldCoordinateBlock) {
		obs.Instructions = append(obs.Instructions, instructions.ShouldCoordinateBlock)
		return nil
	}

	// instructions indicate that the DON has agreed to coordinate on the latest
	// block
	if h.instructions.Has(instructions.DoCoordinateBlock) {
		data, ok := h.metadata.Get(store.BlockHistoryMetadata)
		if !ok {
			return fmt.Errorf("missing block history metadata")
		}

		// add the block history to the observation
		obs.Metadata[ocr2keepers.BlockHistoryObservationKey] = data
	}

	return nil
}
