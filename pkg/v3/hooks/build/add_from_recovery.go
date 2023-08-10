package build

import (
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type addFromRecoveryHook struct {
	metadata *store.Metadata
}

func NewAddFromRecoveryHook(ms *store.Metadata) *addFromRecoveryHook {
	return &addFromRecoveryHook{metadata: ms}
}

func (h *addFromRecoveryHook) RunHook(obs *ocr2keepersv3.AutomationObservation) error {
	// TODO: add limit and random seed here
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
