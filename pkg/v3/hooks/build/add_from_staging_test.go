package build

import (
	"io"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/resultstore"
)

func TestAddFromStaging(t *testing.T) {
	rs := resultstore.New(log.New(io.Discard, "", 0))
	hook := NewAddFromStaging(rs, log.New(io.Discard, "", 0))
	observation := &ocr2keepersv3.AutomationObservation{}
	expected := []ocr2keepers.CheckResult{
		{Payload: ocr2keepers.UpkeepPayload{ID: "test1"}},
		{Payload: ocr2keepers.UpkeepPayload{ID: "test2"}},
	}

	rs.Add(expected...)

	err := hook.RunHook(observation)

	assert.NoError(t, err, "no error from run hook")
	assert.Len(t, observation.Performable, len(expected), "all check results should be in observation")
}
