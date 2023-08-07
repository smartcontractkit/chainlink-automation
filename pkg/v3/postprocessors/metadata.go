package postprocessors

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/ocr2keepers/pkg/util"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type MetadataStore interface {
	Set(store.MetadataKey, interface{})
	Get(store.MetadataKey) (interface{}, bool)
}

type addPayloadToMetadataStorePostprocessor struct {
	store MetadataStore
}

func NewAddPayloadToMetadataStorePostprocessor(store MetadataStore) *addPayloadToMetadataStorePostprocessor {
	return &addPayloadToMetadataStorePostprocessor{store: store}
}

func (a *addPayloadToMetadataStorePostprocessor) PostProcess(_ context.Context, results []ocr2keepers.UpkeepPayload) error {
	rawArray, ok := a.store.Get(store.ProposalRecoveryMetadata)
	if !ok {
		return fmt.Errorf("proposal recovery metadata unavailable")
	}

	values, ok := rawArray.(*util.Cache[ocr2keepers.CoordinatedProposal])
	if !ok {
		return fmt.Errorf("invalid store value type")
	}

	// should only add values and not remove them
	for _, r := range results {
		proposal := ocr2keepers.CoordinatedProposal{
			UpkeepID: r.Upkeep.ID,
			Trigger:  r.Trigger,
		}

		values.Set(fmt.Sprintf("%v", proposal), proposal, util.DefaultCacheExpiration)
	}

	return nil
}

type addSamplesToMetadataStorePostprocessor struct {
	store MetadataStore
}

func NewAddSamplesToMetadataStorePostprocessor(store MetadataStore) *addSamplesToMetadataStorePostprocessor {
	return &addSamplesToMetadataStorePostprocessor{store: store}
}

func (a *addSamplesToMetadataStorePostprocessor) PostProcess(_ context.Context, results []ocr2keepers.CheckResult) error {
	// extract ids only
	ids := make([]ocr2keepers.UpkeepIdentifier, 0, len(results))
	for _, r := range results {
		if !r.Eligible {
			continue
		}

		ids = append(ids, r.Payload.Upkeep.ID)
	}

	// should always reset values every time sampling runs
	a.store.Set(store.ProposalSampleMetadata, ids)

	return nil
}
