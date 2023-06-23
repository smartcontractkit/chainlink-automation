package hooks

import (
	"io"
	"log"
	"testing"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/stretchr/testify/assert"
)

func TestPrebuildHookRemoveFromStaging(t *testing.T) {
	tests := []struct {
		Name  string
		Input []ocr2keepers.CheckResult
	}{
		{
			Name:  "No Results",
			Input: []ocr2keepers.CheckResult{},
		},
		{
			Name: "One Result",
			Input: []ocr2keepers.CheckResult{
				{Payload: ocr2keepers.UpkeepPayload{ID: "test"}},
			},
		},
		{
			Name: "Five Results",
			Input: []ocr2keepers.CheckResult{
				{Payload: ocr2keepers.UpkeepPayload{ID: "test"}},
				{Payload: ocr2keepers.UpkeepPayload{ID: "test1"}},
				{Payload: ocr2keepers.UpkeepPayload{ID: "test2"}},
				{Payload: ocr2keepers.UpkeepPayload{ID: "test3"}},
				{Payload: ocr2keepers.UpkeepPayload{ID: "test4"}},
				{Payload: ocr2keepers.UpkeepPayload{ID: "test5"}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			ob := ocr2keepersv3.AutomationOutcome{
				Performable: test.Input,
			}

			mr := new(mockRemover)

			r := NewPrebuildHookRemoveFromStaging(mr, log.New(io.Discard, "", 0))

			assert.NoError(t, r.RunHook(ob))
			assert.Equal(t, len(ob.Performable), len(mr.removed))
		})
	}
}

type mockRemover struct {
	removed []string
}

func (_m *mockRemover) Remove(toRemove ...string) {
	_m.removed = append(_m.removed, toRemove...)
}
