package plugin

import (
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
)

type addFromSamplesHook struct {
	metadata *store.Metadata
}

func NewAddFromSamplesHook(ms *store.Metadata) *addFromSamplesHook {
	return &addFromSamplesHook{metadata: ms}
}

func (h *addFromSamplesHook) RunHook(obs *ocr2keepersv3.AutomationObservation) error {
	// TODO: add limit and random seed here
	_, err := store.SampleProposalsFromMetadata(h.metadata)
	if err != nil {
		return err
	}

	// TODO: Append obs.CoordinatedProposals

	return nil
}
