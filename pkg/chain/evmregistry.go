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
	"github.com/smartcontractkit/ocr2keepers/pkg/chain/gethwrappers/keeper_registry_wrapper2_0"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

const ActiveUpkeepIDBatchSize int64 = 10000
const separator string = "|"

var (
	ErrRegistryCallFailure   = fmt.Errorf("registry chain call failure")
	ErrBlockKeyNotParsable   = fmt.Errorf("block identifier not parsable")
	ErrUpkeepKeyNotParsable  = fmt.Errorf("upkeep key not parsable")
	ErrInitializationFailure = fmt.Errorf("failed to initialize registry")
)

type evmRegistryv2_0 struct {
	registry  *keeper_registry_wrapper2_0.KeeperRegistryCaller
	evmClient bind.ContractBackend
}

func NewEVMRegistryV2_0(address common.Address, backend bind.ContractBackend) (*evmRegistryv2_0, error) {
	caller, err := keeper_registry_wrapper2_0.NewKeeperRegistryCaller(address, backend)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create caller for address and backend", ErrInitializationFailure)
	}

	return &evmRegistryv2_0{registry: caller, evmClient: backend}, nil
}

func (r *evmRegistryv2_0) GetActiveUpkeepKeys(ctx context.Context, block types.BlockKey) ([]types.UpkeepKey, error) {
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

		if len(nextKeys) == 0 {
			break
		}

		buffer := make([]types.UpkeepKey, len(keys), len(keys)+len(nextKeys))
		copy(keys, buffer)

		keys = append(buffer, nextKeys...)
	}

	return keys, nil
}

func (r *evmRegistryv2_0) CheckUpkeep(ctx context.Context, key types.UpkeepKey) (bool, types.UpkeepResult, error) {
	var err error

	block, upkeepId, err := blockAndIdFromKey(key)
	if err != nil {
		return false, types.UpkeepResult{}, err
	}

	opts, err := r.buildCallOpts(ctx, block)
	if err != nil {
		return false, types.UpkeepResult{}, err
	}

	rawCall := &keeper_registry_wrapper2_0.KeeperRegistryCallerRaw{Contract: r.registry}

	/*
		checkUpkeep(uint256 id)
		returns (
		      bool upkeepNeeded,
		      bytes memory performData,
			  UpkeepFailureReason upkeepFailureReason,
		      uint256 gasUsed,
		      uint256 fastGasWei,
		      uint256 linkNative
		)
	*/

	var out []interface{}
	err = rawCall.Call(opts, &out, "checkUpkeep", upkeepId)
	if err != nil {
		return false, types.UpkeepResult{}, fmt.Errorf("%w: checkUpkeep returned result: %s", ErrRegistryCallFailure, err)
	}

	result := types.UpkeepResult{
		Key:   key,
		State: types.Perform,
	}

	upkeepNeeded := *abi.ConvertType(out[0], new(bool)).(*bool)
	rawPerformData := *abi.ConvertType(out[1], new([]byte)).(*[]byte)
	result.FailureReason = *abi.ConvertType(out[2], new(uint8)).(*uint8)
	result.GasUsed = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	result.FastGasWei = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	result.LinkNative = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)

	if !upkeepNeeded {
		result.State = types.Skip
	}

	if len(rawPerformData) > 0 {
		type performDataStruct struct {
			CheckBlockNumber uint32   `abi:"checkBlockNumber"`
			CheckBlockhash   [32]byte `abi:"checkBlockhash"`
			PerformData      []byte   `abi:"performData"`
		}

		type r struct {
			Result performDataStruct
		}

		// rawPerformData is abi encoded tuple(uint32, bytes32, bytes). We create an ABI with dummy
		// function which returns this tuple in order to decode the bytes
		pdataABI, _ := abi.JSON(strings.NewReader(`[{
			"name":"check",
			"type":"function",
			"outputs":[{
				"name":"ret",
				"type":"tuple",
				"components":[
					{"type":"uint32","name":"checkBlockNumber"},
					{"type":"bytes32","name":"checkBlockhash"},
					{"type":"bytes","name":"performData"}
					]
				}]
			}]`,
		))

		var ret0 = new(r)
		err = pdataABI.UnpackIntoInterface(ret0, "check", rawPerformData)
		if err != nil {
			return false, types.UpkeepResult{}, fmt.Errorf("%w", err)
		}

		result.CheckBlockNumber = ret0.Result.CheckBlockNumber
		result.CheckBlockHash = ret0.Result.CheckBlockhash
		result.PerformData = ret0.Result.PerformData
	}

	return upkeepNeeded, result, nil
}

func (r *evmRegistryv2_0) IdentifierFromKey(key types.UpkeepKey) (types.UpkeepIdentifier, error) {
	_, id, err := blockAndIdFromKey(key)
	if err != nil {
		return nil, err
	}

	return types.UpkeepIdentifier(id.Bytes()), nil
}

func (r *evmRegistryv2_0) buildCallOpts(ctx context.Context, block types.BlockKey) (*bind.CallOpts, error) {
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
