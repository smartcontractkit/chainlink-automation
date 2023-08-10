package plugin

import (
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
)

type AddFromSamplesHook struct {
	metadata *store.Metadata
	coord    Coordinator
}

func NewAddFromSamplesHook(ms *store.Metadata, coord Coordinator) AddFromSamplesHook {
	return AddFromSamplesHook{metadata: ms, coord: coord}
}

func (h *AddFromSamplesHook) RunHook(obs *ocr2keepersv3.AutomationObservation, limit int, rSrc [16]byte) error {
	// TODO: filter using coordinator, add limit and random seed here
	_, err := store.SampleProposalsFromMetadata(h.metadata)
	if err != nil {
		return err
	}

	// TODO: Append obs.CoordinatedProposals

	return nil
}
