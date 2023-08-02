package build

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
	// TODO: need to pass limit here
	ids, err := store.SampleProposalsFromMetadata(h.metadata)
	if err != nil {
		return err
	}
	obs.ProposedConditionalSamples = ids
	return nil
}
