package plugin

import (
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
)

type AddBlockHistoryHook struct {
	metadata store.MetadataStore
}

func NewAddBlockHistoryHook(ms store.MetadataStore) AddBlockHistoryHook {
	return AddBlockHistoryHook{metadata: ms}
}

func (h *AddBlockHistoryHook) RunHook(obs *ocr2keepersv3.AutomationObservation, limit int) error {
	blockHistory := h.metadata.GetBlockHistory()
	if len(blockHistory) > limit {
		blockHistory = blockHistory[:limit]
	}
	obs.BlockHistory = blockHistory
	return nil
}
