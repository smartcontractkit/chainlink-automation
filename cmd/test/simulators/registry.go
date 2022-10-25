package simulators

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type SimulatedUpkeep struct {
	ID         *big.Int
	EligibleAt []*big.Int
	Performs   map[string]types.PerformLog
}

func (ct *SimulatedContract) GetActiveUpkeepKeys(ctx context.Context, key types.BlockKey) ([]types.UpkeepKey, error) {
	block := ct.lastBlock.String()
	keys := []types.UpkeepKey{}

	// TODO: filter out cancelled upkeeps
	for key := range ct.upkeeps {
		k := types.UpkeepKey([]byte(fmt.Sprintf("%s|%s", block, key)))
		keys = append(keys, k)
	}

	return keys, nil
}

func (ct *SimulatedContract) CheckUpkeep(ctx context.Context, key types.UpkeepKey) (bool, types.UpkeepResult, error) {

	parts := strings.Split(string(key), "|")
	if len(parts) != 2 {
		panic("upkeep key does not contain block and id")
	}

	blockInt, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return false, types.UpkeepResult{}, err
	}

	block := big.NewInt(int64(blockInt))

	up, ok := ct.upkeeps[parts[1]]
	if !ok {
		return false, types.UpkeepResult{}, fmt.Errorf("upkeep not registered")
	}

	for j := len(up.EligibleAt) - 1; j >= 0; j-- {
		// TODO: check that upkeep has not been recently performed between two
		// points of eligibility

		if block.Cmp(up.EligibleAt[j]) >= 0 {
			var bl [32]byte
			r := types.UpkeepResult{
				Key:     key,
				State:   types.Eligible,
				GasUsed: big.NewInt(0),
				/*
					FailureReason    uint8
				*/
				PerformData:      []byte{}, // TODO: add perform data
				FastGasWei:       big.NewInt(0),
				LinkNative:       big.NewInt(0),
				CheckBlockNumber: uint32(blockInt),
				CheckBlockHash:   bl,
			}
			return true, r, nil
		}
	}

	return false, types.UpkeepResult{}, fmt.Errorf("unimplemented")
}

func (ct *SimulatedContract) IdentifierFromKey(key types.UpkeepKey) (types.UpkeepIdentifier, error) {
	parts := strings.Split(string(key), "|")
	if len(parts) != 2 {
		panic("upkeep key does not contain block and id")
	}

	return types.UpkeepIdentifier(parts[1]), nil
}
