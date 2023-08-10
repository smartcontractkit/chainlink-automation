package plugin

import (
	"testing"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/plugin/mocks"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"

	"github.com/stretchr/testify/assert"
)

func TestAddFromSamplesHook(t *testing.T) {
	mStore := store.NewMetadata(nil)
	coord := new(mocks.MockCoordinator)

	samples := []ocr2keepers.UpkeepIdentifier{
		ocr2keepers.UpkeepIdentifier([32]byte{1}),
		ocr2keepers.UpkeepIdentifier([32]byte{1}),
	}

	mStore.Set(store.ProposalConditionalMetadata, samples)

	hook := NewAddFromSamplesHook(mStore, coord)
	observation := &ocr2keepersv3.AutomationObservation{}

	assert.NoError(t, hook.RunHook(observation, 10, [16]byte{}), "no error from running hook")
}

func TestAddFromSamplesHook_Error(t *testing.T) {
	mStore := store.NewMetadata(nil)
	coord := new(mocks.MockCoordinator)

	hook := NewAddFromSamplesHook(mStore, coord)
	observation := &ocr2keepersv3.AutomationObservation{}

	assert.ErrorIs(t, hook.RunHook(observation, 10, [16]byte{}), store.ErrMetadataUnavailable, "error from running hook")
}