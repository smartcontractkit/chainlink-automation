package postprocessors

import (
	"context"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type addToMetadataStorePostprocessor struct {
	store      store.MetadataStore
	typeGetter ocr2keepers.UpkeepTypeGetter
}

func NewAddPayloadToMetadataStorePostprocessor(store store.MetadataStore, typeGetter ocr2keepers.UpkeepTypeGetter) *addToMetadataStorePostprocessor {
	return &addToMetadataStorePostprocessor{
		store:      store,
		typeGetter: typeGetter,
	}
}

func (a *addToMetadataStorePostprocessor) PostProcess(_ context.Context, results []ocr2keepers.CheckResult, _ []ocr2keepers.UpkeepPayload) error {
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

type addSamplesToMetadataStorePostprocessor struct {
	store store.MetadataStore
}

func NewAddSamplesToMetadataStorePostprocessor(store store.MetadataStore) *addSamplesToMetadataStorePostprocessor {
	return &addSamplesToMetadataStorePostprocessor{store: store}
}

func (a *addSamplesToMetadataStorePostprocessor) PostProcess(_ context.Context, results []ocr2keepers.CheckResult, _ []ocr2keepers.UpkeepPayload) error {
	// extract ids only
	ids := make([]ocr2keepers.CoordinatedProposal, 0, len(results))
	for _, r := range results {
		if !r.Eligible {
			continue
		}
		ids = append(ids, ocr2keepers.CoordinatedProposal{
			UpkeepID: r.UpkeepID,
			Trigger:  r.Trigger,
			WorkID:   r.WorkID,
		})
	}
	a.store.AddConditionalProposal(ids...)
	return nil
}
