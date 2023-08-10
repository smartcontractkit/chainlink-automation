package plugin

import (
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
)

type AddBlockHistoryHook struct {
	metadata *store.Metadata
}

func NewAddBlockHistoryHook(ms *store.Metadata) AddBlockHistoryHook {
	return AddBlockHistoryHook{metadata: ms}
}

func (h *AddBlockHistoryHook) RunHook(obs *ocr2keepersv3.AutomationObservation, limit int) error {
	// TODO: get block history from metadata store
	// limit it (First limit blocks if they are in descending order)
	// add it to obs.BlockHistory
	return nil
}
