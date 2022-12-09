package chain

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

type OffchainLookup struct {
	sender           common.Address
	urls             []string
	callData         []byte
	callbackFunction [4]byte
	extraData        []byte
}

type OffchainLookupBody struct {
	sender string
	data   string
}

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

func (r *evmRegistryv2_0) check(ctx context.Context, key types.UpkeepKey, ch chan outStruct) {
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
	var performData []byte
	if !upkeepNeeded {
		upkeepInfo, err := r.registry.GetUpkeep(opts, upkeepId)
		if err != nil {
			ch <- outStruct{
				ur:  types.UpkeepResult{},
				err: err,
			}
			return
		} else {

			var payload []byte

			// function checkUpkeep(bytes calldata checkData) external returns (bool upkeepNeeded, bytes memory performData);
			payload, err = r.abiAutomationCompatibleInterface.Pack("checkUpkeep", upkeepInfo.CheckData)
			if err != nil {
				fmt.Println(err)
				ch <- outStruct{
					ur:  types.UpkeepResult{},
					err: err,
				}
				return
			}
			checkUpkeepGasLimit := uint32(200000) + uint32(6500000) + uint32(300000) + upkeepInfo.ExecuteGas

			callMsg := ethereum.CallMsg{
				From: r.address,          // registry addr
				To:   &upkeepInfo.Target, // upkeep addr
				Gas:  uint64(checkUpkeepGasLimit),
				Data: hexutil.Bytes(payload), // checkUpkeep(checkData)
			}

			resp, err := r.evmClient.CallContract(context.Background(), callMsg, opts.BlockNumber)
			if err != nil {
				fmt.Println(err)
				ch <- outStruct{
					ur:  types.UpkeepResult{},
					err: err,
				}
				return
			}

			// error OffchainLookup(address sender, string[] urls, bytes callData, bytes4 callbackFunction, bytes extraData);
			offchainLookup := OffchainLookup{}
			unpack, err := r.abiUpkeep3668.Unpack("OffchainLookup", resp)
			if err != nil {
				fmt.Println(err)
				ch <- outStruct{
					ur:  types.UpkeepResult{},
					err: err,
				}
				return
			}
			offchainLookup.sender = *abi.ConvertType(unpack[0], new(common.Address)).(*common.Address)
			offchainLookup.urls = *abi.ConvertType(unpack[1], new([]string)).(*[]string)
			offchainLookup.callData = *abi.ConvertType(unpack[2], new([]byte)).(*[]byte)
			offchainLookup.callbackFunction = *abi.ConvertType(unpack[3], new([4]byte)).(*[4]byte)
			offchainLookup.extraData = *abi.ConvertType(unpack[4], new([]byte)).(*[]byte)
			fmt.Printf("\n%+v\n", offchainLookup)

			// If the sender field does not match the address of the contract that was called, stop.
			if offchainLookup.sender != upkeepInfo.Target {
				fmt.Println("sender != target")
				ch <- outStruct{
					ur:  types.UpkeepResult{},
					err: err,
				}
				return
			}

			// 	do the http calls
			offchainResp, err := offchainLookup.Query()
			if err != nil {
				fmt.Println(err)
				ch <- outStruct{
					ur:  types.UpkeepResult{},
					err: err,
				}
				return
			}
			fmt.Println(string(offchainResp))

			// 	do callback
			// call to the contract function specified by the 4-byte selector callbackFunction, supplying the data returned and extraData
			typ, err := abi.NewType("bytes", "", nil)
			if err != nil {
				fmt.Println(err)
				ch <- outStruct{
					ur:  types.UpkeepResult{},
					err: err,
				}
				return
			}
			callbackArgs := abi.Arguments{
				{Name: "extraData", Type: typ},
				{Name: "response", Type: typ},
			}
			pack, err := callbackArgs.Pack()
			if err != nil {
				fmt.Println(err)
				ch <- outStruct{
					ur:  types.UpkeepResult{},
					err: err,
				}
				return
			}

			var callbackPayload []byte
			callbackPayload = append(callbackPayload, offchainLookup.callbackFunction[:]...)
			callbackPayload = append(callbackPayload, pack...)

			callbackMsg := ethereum.CallMsg{
				From: r.address,          // registry addr
				To:   &upkeepInfo.Target, // upkeep addr
				Gas:  uint64(checkUpkeepGasLimit),
				Data: hexutil.Bytes(callbackPayload), // callbackFunc(response, extraData)
			}

			callbackResp, err := r.evmClient.CallContract(context.Background(), callbackMsg, opts.BlockNumber)
			if err != nil {
				fmt.Println(err)
				ch <- outStruct{
					ur:  types.UpkeepResult{},
					err: err,
				}
				return
			}

			upkeepNeeded = *abi.ConvertType(callbackResp[0], new(bool)).(*bool)
			if !upkeepNeeded {
				result.State = types.NotEligible
				ch <- outStruct{
					ur: result,
				}
				return
			}
			performData = *abi.ConvertType(callbackResp[1], new([]byte)).(*[]byte)
			result.PerformData = performData
			rawPerformData = performData
		}
	}

	// // TODO: not sure it it's best to short circuit here
	// if !upkeepNeeded {
	// 	result.State = types.NotEligible
	// 	ch <- outStruct{
	// 		ur: result,
	// 	}
	// 	return
	// }

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

// Query - do off chain lookup query. section from eip-3668
// 4 - Construct a request URL by replacing sender with the lowercase 0x-prefixed hexadecimal formatted sender parameter, and replacing data with the the 0x-prefixed hexadecimal formatted callData parameter. The client may choose which URLs to try in which order, but SHOULD prioritise URLs earlier in the list over those later in the list.
// 5 - Make an HTTP GET request to the request URL.
// 6 - If the response code from step (5) is in the range 400-499, return an error to the caller and stop.
// 7 - If the response code from step (5) is in the range 500-599, go back to step (5) and pick a different URL, or stop if there are no further URLs to try.
func (o *OffchainLookup) Query() ([]byte, error) {
	senderString := strings.ToLower(o.sender.Hex())
	callDataString := hex.EncodeToString(o.callData)

	for _, url := range o.urls {
		resp, statusCode, err := o.doRequest(url, senderString, callDataString)
		if err != nil {
			// either an error or a 4XX response
			err = errors.Wrapf(err, "error with query. statusCode: %d ;url: %s ;sender: %s ;callData: %s", statusCode, url, senderString, callDataString)
			return nil, err
		}
		if statusCode <= 299 {
			// success a 2XX response
			return resp, nil
		}
		// continue trying next url
		fmt.Println("didnt work - ", url)
	}

	// If no successful response was received, return an error
	return nil, errors.New("offchain lookup failed")
}

// Given a URL template returned in an OffchainLookup, the URL to query is composed by replacing sender with the lowercase 0x-prefixed hexadecimal formatted sender parameter, and replacing data with the the 0x-prefixed hexadecimal formatted callData parameter.
//
// For example, if a contract returns the following data in an OffchainLookup:
// urls = ["https://example.com/gateway/{sender}/{data}.json"]
// sender = "0xaabbccddeeaabbccddeeaabbccddeeaabbccddee"
// callData = "0x00112233"
// The request URL to query is https://example.com/gateway/0xaabbccddeeaabbccddeeaabbccddeeaabbccddee/0x00112233.json.
//
// If the URL template contains the {data} substitution parameter, the client MUST send a GET request after replacing the substitution parameters as described above.
// If the URL template does not contain the {data} substitution parameter, the client MUST send a POST request after replacing the substitution parameters as described above. The POST request MUST be sent with a Content-Type of application/json, and a payload matching the following schema:
//
//	{
//	   "type": "object",
//	   "properties": {
//	       "data": {
//	           "type": "string",
//	           "description": "0x-prefixed hex string containing the `callData` from the contract"
//	       },
//	       "sender": {
//	           "type": "string",
//	           "description": "0x-prefixed hex string containing the `sender` parameter from the contract"
//	       }
//	   }
//	}.
func (o *OffchainLookup) doRequest(url string, senderString string, callDataString string) ([]byte, int, error) {
	queryUrl := strings.Replace(url, "{sender}", senderString, 1)
	isGET := strings.Contains(url, "{data}")
	fmt.Println("url: ", queryUrl)

	// Construct a request URL by replacing sender with the lowercase 0x-prefixed hexadecimal formatted sender parameter, and replacing data with the 0x-prefixed hexadecimal formatted callData parameter.
	client := http.Client{}
	var req *http.Request
	var err error
	if isGET {
		queryUrl = strings.Replace(url, "{data}", callDataString, 1)
		req, err = http.NewRequest("GET", queryUrl, nil)
		if err != nil {
			return nil, 0, err
		}

	} else {
		body := OffchainLookupBody{
			sender: senderString,
			data:   callDataString,
		}
		jsonBody, _ := json.Marshal(body)
		req, err = http.NewRequest("POST", queryUrl, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, 0, err
		}
	}
	// Make an HTTP GET request to the request URL.
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	// If the response code is in the range 400-499, return an error to the caller and stop.
	if resp.StatusCode >= 400 && resp.StatusCode <= 499 {
		return nil, resp.StatusCode, errors.Errorf("status code %d recieved, stopping offchain lookup", resp.StatusCode)
	}
	// If the response code is in the range 500-599, go back and pick a different URL, or stop if there are no further URLs to try.
	if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
		return nil, resp.StatusCode, nil
	}
	// Return the response body if the status code is between 200 and 299
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return body, resp.StatusCode, nil
	}

	return nil, resp.StatusCode, nil
}

// maybe
// var (
// 	errorSig     = []byte{0x08, 0xc3, 0x79, 0xa0} // Keccak256("Error(string)")[:4]
// 	abiString, _ = abi.NewType("string", "", nil)
// )
//
// func unpackError(result []byte) (string, error) {
// 	if !bytes.Equal(result[:4], errorSig) {
// 		return "<tx result not Error(string)>", errors.New("TX result not of type Error(string)")
// 	}
// 	vs, err := abi.Arguments{{Type: abiString}}.UnpackValues(result[4:])
// 	if err != nil {
// 		return "<invalid tx result>", errors.Wrap(err, "unpacking revert reason")
// 	}
// 	return vs[0].(string), nil
// }

func (r *evmRegistryv2_0) CheckUpkeep(ctx context.Context, key types.UpkeepKey) (bool, types.UpkeepResult, error) {
	chResult := make(chan outStruct, 1)
	go r.check(ctx, key, chResult)

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
