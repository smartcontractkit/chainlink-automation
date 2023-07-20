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
