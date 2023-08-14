package postprocessors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/stores"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func TestMetadataAddSamples(t *testing.T) {
	ms := stores.NewMetadataStore(nil, func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
		return ocr2keepers.ConditionTrigger
	})

	values := []ocr2keepers.CheckResult{
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
		{
			Eligible: false,
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{3}),
			WorkID:   "workID3",
		},
	}

	pp := NewAddProposalToMetadataStorePostprocessor(ms)
	err := pp.PostProcess(context.Background(), values, []ocr2keepers.UpkeepPayload{
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
			WorkID:   "workID1",
		},
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{2}),
			WorkID:   "workID2",
		},
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{3}),
			WorkID:   "workID3",
		},
	})

	assert.NoError(t, err, "no error expected from post processor")

	assert.Equal(t, 2, len(ms.ViewProposals(ocr2keepers.ConditionTrigger)))
}
