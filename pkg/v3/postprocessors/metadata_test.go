package postprocessors

import (
	"context"
	"testing"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/stores"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types/mocks"

	"github.com/stretchr/testify/assert"
)

func TestMetadataAddPayload(t *testing.T) {
	metadataStore := stores.NewMetadataStore(nil, func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
		return ocr2keepers.LogTrigger
	})
	values := []ocr2keepers.UpkeepPayload{
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 4,
				BlockHash:   [32]byte{0},
				LogTriggerExtension: &ocr2keepers.LogTriggerExtension{
					TxHash: [32]byte{1},
					Index:  4,
				},
			},
			WorkID: "workID1",
		},
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{2}),
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 5,
				BlockHash:   [32]byte{0},
				LogTriggerExtension: &ocr2keepers.LogTriggerExtension{
					TxHash: [32]byte{1},
					Index:  5,
				},
			},
			WorkID: "workID2",
		},
	}

	expected := []ocr2keepers.CoordinatedProposal{
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 4,
				BlockHash:   [32]byte{0},
				LogTriggerExtension: &ocr2keepers.LogTriggerExtension{
					TxHash: [32]byte{1},
					Index:  4,
				},
			},
			WorkID: "workID1",
		},
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{2}),
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 5,
				BlockHash:   [32]byte{0},
				LogTriggerExtension: &ocr2keepers.LogTriggerExtension{
					TxHash: [32]byte{1},
					Index:  5,
				},
			},
			WorkID: "workID2",
		},
	}

	postprocessor := NewAddProposalToMetadataStorePostprocessor(metadataStore, func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
		return ocr2keepers.LogTrigger
	})

	err := postprocessor.PostProcess(context.Background(), []ocr2keepers.CheckResult{
		{
			Eligible: true,
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
			WorkID:   "workID1",
		},
		{
			Eligible: true,
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{2}),
			WorkID:   "workID2",
		},
	}, values)

	assert.NoError(t, err, "no error expected from post processor")

	assert.Equal(t, len(metadataStore.ViewLogRecoveryProposal()), len(expected), "values in synced array should match input")
}

func TestMetadataAddSamples(t *testing.T) {
	ms := new(mocks.MockMetadataStore)
	values := []ocr2keepers.CheckResult{
		{
			Eligible: true,
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
		},
		{
			Eligible: true,
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{2}),
		},
		{
			Eligible: false,
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{3}),
		},
	}

	//expected := []ocr2keepers.UpkeepIdentifier{
	//	ocr2keepers.UpkeepIdentifier([32]byte{1}),
	//	ocr2keepers.UpkeepIdentifier([32]byte{2}),
	//}

	//ms.On("Set", store.ProposalConditionalMetadata, expected)

	pp := NewAddProposalToMetadataStorePostprocessor(ms, func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
		return ocr2keepers.ConditionTrigger
	})
	err := pp.PostProcess(context.Background(), values, []ocr2keepers.UpkeepPayload{
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
		},
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{2}),
		},
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{3}),
		},
	})

	assert.NoError(t, err, "no error expected from post processor")

	ms.AssertExpectations(t)
}
