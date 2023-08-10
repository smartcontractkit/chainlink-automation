package plugin

import (
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type AddLogRecoveryProposalsHook struct {
	metadata *store.Metadata
	coord    Coordinator
}

func NewAddLogRecoveryProposalsHook(ms *store.Metadata, coord Coordinator) AddLogRecoveryProposalsHook {
	return AddLogRecoveryProposalsHook{metadata: ms, coord: coord}
}

func (h *AddLogRecoveryProposalsHook) RunHook(obs *ocr2keepersv3.AutomationObservation, limit int, rSrc [16]byte) error {
	// TODO: filter using coordinator, add limit and random seed here
	cache, err := store.RecoveryProposalCacheFromMetadata(h.metadata)
	if err != nil {
		return err
	}

	proposals := make([]ocr2keepers.CoordinatedProposal, 0)
	for _, key := range cache.Keys() {
		v, ok := cache.Get(key)
		if !ok {
			cache.Delete(key)

			continue
		}

		proposals = append(proposals, v)
	}

	// TODO: Append obs.CoordinatedProposals

	return nil
}
