package plugin

import (
	"testing"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types/mocks"

	"github.com/stretchr/testify/assert"
)

func TestAddFromRecoveryHook(t *testing.T) {
	mStore := store.NewMetadataStore(nil)
	coord := new(mocks.MockCoordinator)

	expectedProps := []ocr2keepers.CoordinatedProposal{
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 1,
			},
		},
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{2}),
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 2,
			},
		},
	}

	mStore.AddLogRecoveryProposal(expectedProps...)

	hook := NewAddLogRecoveryProposalsHook(mStore, coord)
	observation := &ocr2keepersv3.AutomationObservation{}

	assert.NoError(t, hook.RunHook(observation, 10, [16]byte{}), "no error from running hook")
}

func TestAddFromRecoveryHook_Error(t *testing.T) {
	mStore := store.NewMetadataStore(nil)
	coord := new(mocks.MockCoordinator)
	hook := NewAddLogRecoveryProposalsHook(mStore, coord)
	observation := &ocr2keepersv3.AutomationObservation{}

	assert.ErrorIs(t, hook.RunHook(observation, 10, [16]byte{}), nil, "error from running hook")
}
