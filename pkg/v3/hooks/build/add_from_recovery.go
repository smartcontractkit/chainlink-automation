package build

import (
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
)

type addFromRecoveryHook struct {
	metadata *store.Metadata
}

func NewAddFromRecoveryHook(ms *store.Metadata) *addFromRecoveryHook {
	return &addFromRecoveryHook{metadata: ms}
}

func (h *addFromRecoveryHook) RunHook(obs *ocr2keepersv3.AutomationObservation) error {
	// TODO: Need to pass limit here
	proposals := make([]ocr2keepers.Trigger, 0)
	// TODO: Fetch from instruction store and create proposals
	obs.ProposedRecoveryLogs = proposals

	return nil
}
