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
	cache, err := store.RecoveryProposalCacheFromMetadata(h.metadata)
	if err != nil {
		return err
	}

	proposals := make([]ocr2keepers.CoordinatedProposal, 0)
	for _, key := range cache.Keys() {
		v, ok := cache.Get(key)
		if !ok {
			cache.Delete(key)
		}

		proposals = append(proposals, v)
	}

	obs.Metadata[ocr2keepersv3.RecoveryProposalObservationKey] = proposals

	return nil
}
