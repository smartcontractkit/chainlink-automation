package upkeep_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/simulate/upkeep"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func TestUtil_EncodeDecode(t *testing.T) {
	utilities := upkeep.Util{}

	encoded, err := utilities.Encode(ocr2keepers.CheckResult{})

	require.NoError(t, err)

	reported, err := utilities.Extract(encoded)

	require.NoError(t, err)
	assert.Len(t, reported, 1)
}
