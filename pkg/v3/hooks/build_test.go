package hooks

import (
	"io"
	"log"
	"testing"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/resultstore"
	"github.com/stretchr/testify/assert"
)

func TestBuildHookAddFromStaging(t *testing.T) {
	hook := NewBuildHookAddFromStaging()
	observation := &ocr2keepersv3.AutomationObservation{}
	rs := resultstore.New(log.New(io.Discard, "", 0))
	expected := []ocr2keepers.CheckResult{
		{Payload: ocr2keepers.UpkeepPayload{ID: "test1"}},
		{Payload: ocr2keepers.UpkeepPayload{ID: "test2"}},
	}

	rs.Add(expected...)

	err := hook.RunHook(observation, nil, nil, rs)

	assert.NoError(t, err, "no error from run hook")
	assert.Len(t, observation.Performable, len(expected), "all check results should be in observation")
}
