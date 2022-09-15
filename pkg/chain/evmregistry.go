package chain

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/smartcontractkit/ocr2keepers/gethwrappers/keeper_registry_v1_2"
	"github.com/smartcontractkit/ocr2keepers/internal/keepers"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

const ActiveUpkeepIDBatchSize int64 = 10000
const separator string = "|"

var (
	ErrRegistryCallFailure  = fmt.Errorf("registry chain call failure")
	ErrBlockKeyNotParsable  = fmt.Errorf("block identifier not parsable")
	ErrUpkeepKeyNotParsable = fmt.Errorf("upkeep key not parsable")
)

type evmRegistryv1_2 struct {
	registry  *keeper_registry_v1_2.KeeperRegistryCaller
	evmClient bind.ContractBackend
}

func NewEVMRegistryV1_2(address common.Address, backend bind.ContractBackend) (*evmRegistryv1_2, error) {
	caller, err := keeper_registry_v1_2.NewKeeperRegistryCaller(address, backend)
	if err != nil {
		// TODO: do better error handling here
		return nil, err
	}

	return &evmRegistryv1_2{registry: caller, evmClient: backend}, nil
}

func (r *evmRegistryv1_2) GetActiveUpkeepKeys(ctx context.Context, block types.BlockKey) ([]types.UpkeepKey, error) {
	opts, err := r.buildCallOpts(ctx, block)
	if err != nil {
		return nil, err
	}

	state, err := r.registry.GetState(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get contract state at block number %d", opts.BlockNumber.Int64())
	}

	keys := make([]types.UpkeepKey, 0)
	for int64(len(keys)) < state.State.NumUpkeeps.Int64() {
		startIndex := int64(len(keys))
		maxCount := state.State.NumUpkeeps.Int64() - int64(len(keys))

		if maxCount > ActiveUpkeepIDBatchSize {
			maxCount = ActiveUpkeepIDBatchSize
		}

		nextRawKeys, err := r.registry.GetActiveUpkeepIDs(opts, big.NewInt(startIndex), big.NewInt(maxCount))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get active upkeep IDs from index %d to %d (both inclusive)", startIndex, startIndex+maxCount-1)
		}

		nextKeys := make([]types.UpkeepKey, len(nextRawKeys))
		for i, next := range nextRawKeys {
			nextKeys[i] = []byte(fmt.Sprintf("%s%s%s", opts.BlockNumber, separator, next))
		}

		buffer := make([]types.UpkeepKey, len(keys), len(keys)+len(nextKeys))
		copy(keys, buffer)

		keys = append(buffer, nextKeys...)
	}

	return keys, nil
}

func (r *evmRegistryv1_2) CheckUpkeep(ctx context.Context, from types.Address, key types.UpkeepKey) (bool, types.UpkeepResult, error) {
	var err error

	fromAddr := common.BytesToAddress([]byte(from))

	block, upkeepId, err := blockAndIdFromKey(key)
	if err != nil {
		return false, types.UpkeepResult{}, err
	}

	opts, err := r.buildCallOpts(ctx, block)
	if err != nil {
		return false, types.UpkeepResult{}, err
	}

	rawCall := &keeper_registry_v1_2.KeeperRegistryCallerRaw{Contract: r.registry}

	/*
		checkUpkeep(uint256 id, address from)
		returns (
			bytes memory performData,
			uint256 maxLinkPayment,
			uint256 gasLimit,
			uint256 adjustedGasWei,
			uint256 linkEth
		)
	*/

	var out []interface{}
	err = rawCall.Call(opts, &out, "checkUpkeep", upkeepId, fromAddr)
	if err != nil {
		// contract implementation reverts with error if an upkeep contract returns
		// false on a checkUpkeep call and does not include performData in revert message
		noUpkeepMsg := strings.Contains(strings.ToLower(err.Error()), strings.ToLower("UpkeepNotNeeded"))
		if noUpkeepMsg {
			return false, types.UpkeepResult{}, nil
		}

		return false, types.UpkeepResult{}, fmt.Errorf("%w: checkUpkeep returned result: %s", ErrRegistryCallFailure, err)
	}

	performData := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	// other types returned from contract call that may be needed in the future
	// maxLinkPayment := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	// gasLimit := *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	// adjustedGasWei := *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	// linkEth := *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)

	return true, types.UpkeepResult{Key: key, State: keepers.Perform, PerformData: performData}, nil

}

func (r *evmRegistryv1_2) buildCallOpts(ctx context.Context, block types.BlockKey) (*bind.CallOpts, error) {
	b := new(big.Int)
	_, ok := b.SetString(string(block), 10)

	if !ok {
		return nil, fmt.Errorf("%w: requires big int", ErrBlockKeyNotParsable)
	}

	if b == nil || b.Int64() == 0 {
		// fetch the current block number so batched GetActiveUpkeepKeys calls can be performed on the same block
		header, err := r.evmClient.HeaderByNumber(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: EVM failed to fetch block header", err, ErrRegistryCallFailure)
		}

		b = header.Number
	}

	return &bind.CallOpts{
		Context:     ctx,
		BlockNumber: b,
	}, nil
}

func blockAndIdFromKey(key types.UpkeepKey) (types.BlockKey, *big.Int, error) {
	parts := strings.Split(string(key), separator)
	if len(parts) != 2 {
		return types.BlockKey(""), nil, fmt.Errorf("%w: missing data in upkeep key", ErrUpkeepKeyNotParsable)
	}

	id := new(big.Int)
	_, ok := id.SetString(parts[1], 10)
	if !ok {
		return types.BlockKey(""), nil, fmt.Errorf("%w: must be big int", ErrUpkeepKeyNotParsable)
	}

	return types.BlockKey(parts[0]), id, nil
}
