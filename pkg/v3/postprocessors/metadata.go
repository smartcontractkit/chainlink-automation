package postprocessors

import (
	"context"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type addProposalToMetadataStore struct {
	metadataStore ocr2keepers.MetadataStore
	typeGetter    ocr2keepers.UpkeepTypeGetter
}

func NewAddProposalToMetadataStorePostprocessor(store ocr2keepers.MetadataStore, typeGetter ocr2keepers.UpkeepTypeGetter) *addProposalToMetadataStore {
	return &addProposalToMetadataStore{metadataStore: store, typeGetter: typeGetter}
}

func (a *addProposalToMetadataStore) PostProcess(_ context.Context, results []ocr2keepers.CheckResult, _ []ocr2keepers.UpkeepPayload) error {
	// should only add values and not remove them
	for _, r := range results {
		proposal := ocr2keepers.CoordinatedBlockProposal{
			UpkeepID: r.UpkeepID,
			Trigger:  r.Trigger,
			WorkID:   r.WorkID,
		}
		switch a.typeGetter(r.UpkeepID) {
		case ocr2keepers.LogTrigger:
			a.metadataStore.AddLogRecoveryProposal(proposal)
		case ocr2keepers.ConditionTrigger:
			a.metadataStore.AddConditionalProposal(proposal)
		default:
		}
	}

	return nil
}
