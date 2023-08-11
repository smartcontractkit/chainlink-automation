package postprocessors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/ocr2keepers/pkg/util"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store/mocks"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"

	"github.com/stretchr/testify/assert"
)

func TestMetadataAddPayload(t *testing.T) {
	ms := new(mocks.MockMetadataStore)
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
		},
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{2}),
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 4,
				BlockHash:   [32]byte{0},
				LogTriggerExtension: &ocr2keepers.LogTriggerExtension{
					TxHash: [32]byte{1},
					Index:  4,
				},
			},
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
		},
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{2}),
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 4,
				BlockHash:   [32]byte{0},
				LogTriggerExtension: &ocr2keepers.LogTriggerExtension{
					TxHash: [32]byte{1},
					Index:  4,
				},
			},
		},
	}

	ar := util.NewCache[ocr2keepers.CoordinatedProposal](util.DefaultCacheExpiration)

	ms.On("SetProposalLogRecovery", "{452312848583266388373324160190187140051835877600158453279131187530910662656 {0 [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0] <nil>} }", mock.Anything, mock.Anything).Once()
	ms.On("SetProposalLogRecovery", "{904625697166532776746648320380374280103671755200316906558262375061821325312 {0 [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0] <nil>} }", mock.Anything, mock.Anything).Once()

	pp := NewAddPayloadToMetadataStorePostprocessor(ms, func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
		return ocr2keepers.LogTrigger
	})
	err := pp.PostProcess(context.Background(), []ocr2keepers.CheckResult{
		{
			Eligible: true,
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
		},
		{
			Eligible: true,
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{2}),
		},
	}, values)

	assert.NoError(t, err, "no error expected from post processor")

	ms.AssertExpectations(t)

	result := make([]ocr2keepers.CoordinatedProposal, 0)
	for _, key := range ar.Keys() {
		v, _ := ar.Get(key)
		result = append(result, v)
	}

	assert.Len(t, result, len(expected), "values in synced array should match input")
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

	pp := NewAddSamplesToMetadataStorePostprocessor(ms)
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
