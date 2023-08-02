package build

import (
	"fmt"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
)

type coordinateBlockHook struct {
	metadata *store.Metadata
}

// NewBlockHistoryHook creates a new build hook that adds the latest block
// history to the observation metadata
func NewBlockHistoryHook(
	metadata *store.Metadata,
) *coordinateBlockHook {
	return &coordinateBlockHook{
		metadata: metadata,
	}
}

func (h *coordinateBlockHook) RunHook(obs *ocr2keepers.AutomationObservation) error {
	data, ok := h.metadata.Get(store.BlockHistoryMetadata)
	if !ok {
		return fmt.Errorf("missing block history metadata")
	}

	// add the block history to the observation
	obs.BlockHistory = data.(ocr2keepers.BlockHistory)

	return nil
}
