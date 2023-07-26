package postprocessors

import (
	"context"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
)

type MetadataStore interface {
	Set(store.MetadataKey, interface{})
}

type addPayloadToMetadataStorePostprocessor struct {
	store MetadataStore
}

func NewAddPayloadToMetadataStorePostprocessor(store MetadataStore) *addPayloadToMetadataStorePostprocessor {
	return &addPayloadToMetadataStorePostprocessor{store: store}
}

func (a *addPayloadToMetadataStorePostprocessor) PostProcess(_ context.Context, results []ocr2keepers.UpkeepPayload) error {
	// TODO: should only add values and not remove them
	a.store.Set(store.ProposalRecoveryMetadata, results)
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
