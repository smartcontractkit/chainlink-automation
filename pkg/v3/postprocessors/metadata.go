package postprocessors

import (
	"context"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type addProposalToMetadataStore struct {
	store      store.MetadataStore
	typeGetter ocr2keepers.UpkeepTypeGetter
}

func NewAddProposalToMetadataStorePostprocessor(store store.MetadataStore, typeGetter ocr2keepers.UpkeepTypeGetter) *addProposalToMetadataStore {
	return &addProposalToMetadataStore{store: store, typeGetter: typeGetter}
}

func (a *addProposalToMetadataStore) PostProcess(_ context.Context, results []ocr2keepers.CheckResult, _ []ocr2keepers.UpkeepPayload) error {
	// should only add values and not remove them
	for _, r := range results {
		proposal := ocr2keepers.CoordinatedProposal{
			UpkeepID: r.UpkeepID,
			Trigger:  r.Trigger,
			WorkID:   r.WorkID,
		}
		switch a.typeGetter(r.UpkeepID) {
		case ocr2keepers.LogTrigger:
			a.store.AddLogRecoveryProposal(proposal)
		case ocr2keepers.ConditionTrigger:
			a.store.AddConditionalProposal(proposal)
		default:
		}
	}

	return nil
}
