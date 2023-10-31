package modify_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	ocr2keeperstypes "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	"github.com/smartcontractkit/ocr2keepers/tools/testprotocol/modify"
)

func TestModifyBytes(t *testing.T) {
	originalName := "test modifier"
	modifier := modify.ModifyBytes(
		originalName,
		modify.WithModifyKeyValue(
			"BlockNumber",
			func(path string, values interface{}) interface{} {
				return -1
			}))

	observation := ocr2keepers.AutomationObservation{
		Performable: []ocr2keeperstypes.CheckResult{
			{
				Trigger: ocr2keeperstypes.NewLogTrigger(
					ocr2keeperstypes.BlockNumber(10),
					[32]byte{},
					&ocr2keeperstypes.LogTriggerExtension{
						TxHash:      [32]byte{},
						Index:       1,
						BlockHash:   [32]byte{},
						BlockNumber: ocr2keeperstypes.BlockNumber(10),
					},
				),
			},
		},
		UpkeepProposals: []ocr2keeperstypes.CoordinatedBlockProposal{},
		BlockHistory:    []ocr2keeperstypes.BlockKey{},
	}

	original, err := json.Marshal(observation)
	name, modified, err := modifier(context.Background(), original, err)

	assert.NoError(t, err)
	assert.NotEqual(t, original, modified)
	assert.Equal(t, originalName, name)
}
