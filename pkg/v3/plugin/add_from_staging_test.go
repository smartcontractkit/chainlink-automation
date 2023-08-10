package plugin

import (
	"fmt"
	"io"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/plugin/mocks"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func TestAddFromStaging(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		ms := new(mocks.MockResultViewer)
		coord := new(mocks.MockCoordinator)
		hook := NewAddFromStaging(ms, log.New(io.Discard, "", 0), coord)
		observation := &ocr2keepersv3.AutomationObservation{}
		expected := []ocr2keepers.CheckResult{
			{UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1})},
			{UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{2})},
		}

		ms.On("View").Return(expected, nil)

		err := hook.RunHook(observation, 10, [16]byte{})

		assert.NoError(t, err, "no error from run hook")
		assert.Len(t, observation.Performable, len(expected), "all check results should be in observation")
	})

	t.Run("result store error", func(t *testing.T) {
		ms := new(mocks.MockResultViewer)
		coord := new(mocks.MockCoordinator)
		hook := NewAddFromStaging(ms, log.New(io.Discard, "", 0), coord)
		observation := &ocr2keepersv3.AutomationObservation{}

		ms.On("View").Return(nil, fmt.Errorf("test error"))

		err := hook.RunHook(observation, 10, [16]byte{})

		assert.NotNil(t, err, "error expected from run hook")
	})
}
