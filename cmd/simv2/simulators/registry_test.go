package simulators

import (
	"context"
	"math/big"
	"testing"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestCheckUpkeep(t *testing.T) {
	contract := &SimulatedContract{
		avgLatency: 2,
		upkeeps: map[string]SimulatedUpkeep{
			"201": {
				ID: big.NewInt(201),
				EligibleAt: []*big.Int{
					big.NewInt(5),
					big.NewInt(10),
					big.NewInt(15),
					big.NewInt(20),
				},
				Performs: map[string]types.PerformLog{
					"7": {
						Key: types.UpkeepKey([]byte("4|20")),
					},
				},
			},
		},
	}

	checkKey := types.UpkeepKey([]byte("8|201"))
	ok, res, err := contract.CheckUpkeep(context.Background(), checkKey)

	assert.Equal(t, false, ok)
	assert.NoError(t, err)
	assert.Equal(t, checkKey, res.Key)
	assert.Equal(t, types.NotEligible, res.State)

	checkKey2 := types.UpkeepKey([]byte("11|201"))
	ok, res, err = contract.CheckUpkeep(context.Background(), checkKey2)

	assert.Equal(t, true, ok)
	assert.NoError(t, err)
	assert.Equal(t, checkKey2, res.Key)
	assert.Equal(t, types.Eligible, res.State)
}
