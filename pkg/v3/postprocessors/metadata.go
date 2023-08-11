package postprocessors

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/ocr2keepers/pkg/util"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type addToMetadataStorePostprocessor struct {
	store store.MetadataStore
}

func NewAddPayloadToMetadataStorePostprocessor(store store.MetadataStore) *addToMetadataStorePostprocessor {
	return &addToMetadataStorePostprocessor{store: store}
}

func (a *addToMetadataStorePostprocessor) PostProcess(_ context.Context, results []ocr2keepers.CheckResult, _ []ocr2keepers.UpkeepPayload) error {
	// should only add values and not remove them
	for _, r := range results {
		proposal := ocr2keepers.CoordinatedProposal{
			UpkeepID: r.UpkeepID,
			Trigger:  r.Trigger,
		}

		a.store.SetProposalLogRecovery(fmt.Sprintf("%v", proposal), proposal, util.DefaultCacheExpiration)
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
	ids := make([]ocr2keepers.UpkeepIdentifier, 0, len(results))
	for _, r := range results {
		if !r.Eligible {
			continue
		}

		ids = append(ids, r.UpkeepID)
	}

	// should always reset values every time sampling runs
	a.store.AppendProposalConditional(ids...)
	return nil
}
