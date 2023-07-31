package build

import (
	"fmt"
	"testing"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/util"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	"github.com/stretchr/testify/assert"
)

func TestAddFromRecoveryHook(t *testing.T) {
	mStore := store.NewMetadata(nil)
	cache := util.NewCache[ocr2keepers.CoordinatedProposal](util.DefaultCacheExpiration)

	expectedProps := []ocr2keepers.CoordinatedProposal{
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier("1"),
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 1,
			},
		},
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier("2"),
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 2,
			},
		},
	}

	for _, p := range expectedProps {
		cache.Set(fmt.Sprintf("%v", p), p, util.DefaultCacheExpiration)
	}

	mStore.Set(store.ProposalRecoveryMetadata, cache)

	hook := NewAddFromRecoveryHook(mStore)
	observation := &ocr2keepersv3.AutomationObservation{
		Metadata: map[ocr2keepersv3.ObservationMetadataKey]interface{}{},
	}

	assert.NoError(t, hook.RunHook(observation), "no error from running hook")
	assert.Len(
		t,
		observation.Metadata[ocr2keepersv3.RecoveryProposalObservationKey].([]ocr2keepers.CoordinatedProposal),
		2,
		"observation proposals should match expected length")
}

func TestAddFromRecoveryHook_Error(t *testing.T) {
	mStore := store.NewMetadata(nil)
	hook := NewAddFromRecoveryHook(mStore)
	observation := &ocr2keepersv3.AutomationObservation{
		Metadata: map[ocr2keepersv3.ObservationMetadataKey]interface{}{},
	}

	assert.ErrorIs(t, hook.RunHook(observation), store.ErrMetadataUnavailable, "error from running hook")
}
