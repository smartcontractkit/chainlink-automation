package chain

import (
	"context"
	"fmt"
	"log"
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
	ErrContextCancelled      = fmt.Errorf("context was cancelled")
)

type outStruct struct {
	ok  bool
	ur  types.UpkeepResult
	err error
}

type evmRegistryv2_0 struct {
	registry                         *keeper_registry_wrapper2_0.KeeperRegistryCaller
	evmClient                        bind.ContractBackend
	address                          common.Address
	abiAutomationCompatibleInterface *abi.ABI
	abiUpkeep3668                    *abi.ABI
}

func NewEVMRegistryV2_0(address common.Address, backend bind.ContractBackend) (*evmRegistryv2_0, error) {
	keeperRegistry, err := keeper_registry_wrapper2_0.NewKeeperRegistry(address, backend)
	if err != nil {
		// TODO: do better error handling here
		return nil, err
	}
	abiAutomationCompatibleInterface, err := abi.JSON(strings.NewReader("[{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"checkData\",\"type\":\"bytes\"}],\"name\":\"checkUpkeep\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"upkeepNeeded\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"performData\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"performData\",\"type\":\"bytes\"}],\"name\":\"performUpkeep\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"))
	if err != nil {
		return nil, err
	}
	abiUpkeep3668, err := abi.JSON(strings.NewReader("[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_testRange\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_interval\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"string[]\",\"name\":\"urls\",\"type\":\"string[]\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"},{\"internalType\":\"bytes4\",\"name\":\"callbackFunction\",\"type\":\"bytes4\"},{\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"OffchainLookup\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"initialBlock\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"lastBlock\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"previousBlock\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"counter\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"resp\",\"type\":\"uint256\"}],\"name\":\"PerformingUpkeep\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"resp\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"extra\",\"type\":\"bytes\"}],\"name\":\"callback\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"checkUpkeep\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"counter\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"eligible\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initialBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"interval\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"lastBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"performData\",\"type\":\"bytes\"}],\"name\":\"performUpkeep\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"previousPerformBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_testRange\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_interval\",\"type\":\"uint256\"}],\"name\":\"setConfig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string[]\",\"name\":\"input\",\"type\":\"string[]\"}],\"name\":\"setURLs\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"testRange\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"urls\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"))
	if err != nil {
		return nil, err
	}
	return &evmRegistryv2_0{registry: &keeperRegistry.KeeperRegistryCaller, evmClient: backend, address: keeperRegistry.Address(), abiAutomationCompatibleInterface: &abiAutomationCompatibleInterface, abiUpkeep3668: &abiUpkeep3668}, nil
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
			nextKeys[i] = BlockAndIdToKey(opts.BlockNumber, next)
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

func (r *evmRegistryv2_0) check(ctx context.Context, key types.UpkeepKey, ch chan outStruct, logger *log.Logger) {
	block, upkeepId, err := BlockAndIdFromKey(key)
	if err != nil {
		ch <- outStruct{
			ur:  types.UpkeepResult{},
			err: err,
		}
		return
	}

	opts, err := r.buildCallOpts(ctx, block)
	if err != nil {
		ch <- outStruct{
			ur:  types.UpkeepResult{},
			err: err,
		}
		return
	}

	rawCall := &keeper_registry_wrapper2_0.KeeperRegistryCallerRaw{Contract: r.registry}

	var out []interface{}
	err = rawCall.Call(opts, &out, "checkUpkeep", upkeepId)
	if err != nil {
		ch <- outStruct{
			ur:  types.UpkeepResult{},
			err: fmt.Errorf("%w: checkUpkeep returned result: %s", ErrRegistryCallFailure, err),
		}
		return
	}

	result := types.UpkeepResult{
		Key:   key,
		State: types.Eligible,
	}

	upkeepNeeded := *abi.ConvertType(out[0], new(bool)).(*bool)
	rawPerformData := *abi.ConvertType(out[1], new([]byte)).(*[]byte)
	result.FailureReason = *abi.ConvertType(out[2], new(uint8)).(*uint8)
	result.GasUsed = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	result.FastGasWei = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	result.LinkNative = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)

	// ccip read
	// if reverts plus flag that eip3668, for now we can assume eip3668 for POC
	// so then we need to call the contract directly
	// should pass thru to avoid extra rpc call
	if !upkeepNeeded {
		upkeepInfo, err := r.registry.GetUpkeep(opts, upkeepId)
		if err != nil {
			logger.Println(err)
			ch <- outStruct{
				ur:  types.UpkeepResult{},
				err: err,
			}
			return
		} else {

			offchainLookup, err := r.callTargetCheckUpkeep(upkeepInfo, opts)
			if err != nil {
				logger.Println(err)
				ch <- outStruct{
					ur:  types.UpkeepResult{},
					err: err,
				}
				return
			}
			logger.Printf("\n%+v\n", offchainLookup)

			// If the sender field does not match the address of the contract that was called, stop.
			if offchainLookup.sender != upkeepInfo.Target {
				logger.Println(offchainLookup.sender, " != ", upkeepInfo.Target)
				// ch <- outStruct{
				// 	ur:  types.UpkeepResult{},
				// 	err: errors.New("OffchainLookup sender != target"),
				// }
				// return
			}

			// 	do the http calls
			offchainResp, err := offchainLookup.query()
			if err != nil {
				logger.Println(err)
				ch <- outStruct{
					ur:  types.UpkeepResult{},
					err: err,
				}
				return
			}
			logger.Println(string(offchainResp))

			needed, performData, err := r.offchainLookupCallback(offchainLookup, offchainResp, upkeepInfo, opts)
			if !needed {
				logger.Println(err)
				result.State = types.NotEligible
				ch <- outStruct{
					ur: result,
				}
				return
			}
			upkeepNeeded = needed
			result.PerformData = performData
			rawPerformData = performData
			logger.Println("OffchainLookup Success!!")
		}
	}

	// TODO: not sure it it's best to short circuit here
	if !upkeepNeeded {
		result.State = types.NotEligible
		ch <- outStruct{
			ur: result,
		}
		return
	}

	type performDataStruct struct {
		CheckBlockNumber uint32   `abi:"checkBlockNumber"`
		CheckBlockhash   [32]byte `abi:"checkBlockhash"`
		PerformData      []byte   `abi:"performData"`
	}

	type res struct {
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

	var ret0 = new(res)
	err = pdataABI.UnpackIntoInterface(ret0, "check", rawPerformData)
	if err != nil {
		ch <- outStruct{
			ur:  types.UpkeepResult{},
			err: fmt.Errorf("%w", err),
		}
		return
	}

	result.CheckBlockNumber = ret0.Result.CheckBlockNumber
	result.CheckBlockHash = ret0.Result.CheckBlockhash
	result.PerformData = ret0.Result.PerformData

	// Since checkUpkeep is true, simulate the perform upkeep to ensure it doesn't revert
	var out2 []interface{}
	err = rawCall.Call(opts, &out2, "simulatePerformUpkeep", upkeepId, result.PerformData)
	if err != nil {
		ch <- outStruct{
			ur:  types.UpkeepResult{},
			err: fmt.Errorf("%w: simulate perform upkeep returned result: %s", ErrRegistryCallFailure, err),
		}
		return
	}
	simulatePerformSuccess := *abi.ConvertType(out2[0], new(bool)).(*bool)
	if !simulatePerformSuccess {
		result.State = types.NotEligible
		ch <- outStruct{
			ur: result,
		}
		return
	}

	ch <- outStruct{
		ok: true,
		ur: result,
	}
}

func (r *evmRegistryv2_0) CheckUpkeep(ctx context.Context, key types.UpkeepKey, logger *log.Logger) (bool, types.UpkeepResult, error) {
	chResult := make(chan outStruct, 1)
	go r.check(ctx, key, chResult, logger)

	select {
	case rs := <-chResult:
		return rs.ok, rs.ur, rs.err
	case <-ctx.Done():
		// safety on context done to provide an error on context cancellation
		// contract calls through the geth wrappers are a bit of a black box
		// so this safety net ensures contexts are fully respected and contract
		// call functions have a more graceful closure outside the scope of
		// CheckUpkeep needing to return immediately.
		return false, types.UpkeepResult{}, fmt.Errorf("%w: failed to check upkeep on registry", ErrContextCancelled)
	}
}

func (r *evmRegistryv2_0) IdentifierFromKey(key types.UpkeepKey) (types.UpkeepIdentifier, error) {
	_, id, err := BlockAndIdFromKey(key)
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

func BlockAndIdFromKey(key types.UpkeepKey) (types.BlockKey, *big.Int, error) {
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

func BlockAndIdToKey(block *big.Int, id *big.Int) types.UpkeepKey {
	return types.UpkeepKey([]byte(fmt.Sprintf("%s%s%s", block, separator, id)))
}
