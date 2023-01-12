package chain

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"

	"github.com/smartcontractkit/ocr2keepers/pkg/chain/gethwrappers/keeper_registry_wrapper2_0"
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

// JsonError is a rpc.jsonError interface
type JsonError interface {
	Error() string
	ErrorData() interface{}
}

/*
POC NOTES:
- need to update registry to signal to us if it's an offchain lookup or if we don't want to do a contract change we need to use the offchain config to tell us this upkeep could have an offchainlookup revert, so we can check it when upkeepNeeded is false. else we are do extra eth calls.
- the url replacing for sender and data strict and doesn't give a lot of freedom. We may want to allow a bit more flexibility?
- how many urls will we allow in the list?
- what is the http request timeout?
- spamming api calls if the user doesn’t have some rate limiting logic around their checkUpkeep
- should we rate limit failure cases?
- should we rate limit per upkeep id?
- public api may rate limit by IP causing users to block one another
- potentially 2x number of Nodes api queries on every cycle.
*/

func (r *evmRegistryv2_0) callTargetCheckUpkeep(upkeepInfo keeper_registry_wrapper2_0.UpkeepInfo, opts *bind.CallOpts, logger *log.Logger) (OffchainLookup, error) {
	fmt.Println("~~~", "starting callTargetCheckUpkeep")
	// function checkUpkeep(bytes calldata checkData) external returns (bool upkeepNeeded, bytes memory performData);
	payload, err := r.abiAutomationCompatibleInterface.Pack("checkUpkeep", upkeepInfo.CheckData)
	if err != nil {
		return OffchainLookup{}, errors.Wrapf(err, "checkUpkeep pack error:")
	}
	checkUpkeepGasLimit := uint32(200000) + uint32(6500000) + uint32(300000) + upkeepInfo.ExecuteGas

	callMsg := ethereum.CallMsg{
		From: r.address,          // registry addr
		To:   &upkeepInfo.Target, // upkeep addr
		Gas:  uint64(checkUpkeepGasLimit),
		Data: hexutil.Bytes(payload), // checkUpkeep(checkData)
	}

	fmt.Println("~~~", "calling upkeep contract directly")
	_, jsonErr := r.evmClient.CallContract(context.Background(), callMsg, opts.BlockNumber)
	if jsonErr == nil {
		fmt.Println("~~~", "jsonErr == nil")
		return OffchainLookup{}, errors.Wrapf(err, "call contract error:")
	}
	fmt.Println("~~~", "done calling upkeep contract directly")

	if _, ok := jsonErr.(JsonError); !ok {
		fmt.Println("~~~", "err was not json error")
		fmt.Println("~~~", "type of err is.....", reflect.TypeOf(jsonErr).Elem())
		return OffchainLookup{}, errors.Wrapf(err, "err is type %T no JsonError:", err)
	}

	// error OffchainLookup(address sender, string[] urls, bytes callData, bytes4 callbackFunction, bytes extraData);
	offchainLookup := OffchainLookup{}
	e := r.abiUpkeep3668.Errors["OffchainLookup"]
	fmt.Println("~~~", "error data", jsonErr.(JsonError).ErrorData().(string))
	decode, err := hexutil.Decode(jsonErr.(JsonError).ErrorData().(string))
	if err != nil {
		return OffchainLookup{}, errors.Wrapf(err, "decode jsonError error:")
	}
	unpack, err := e.Unpack(decode)
	if err != nil {
		return OffchainLookup{}, errors.Wrapf(err, "unpack error:")
	}
	errorParameters := unpack.([]interface{})

	offchainLookup.sender = *abi.ConvertType(errorParameters[0], new(common.Address)).(*common.Address)
	offchainLookup.urls = *abi.ConvertType(errorParameters[1], new([]string)).(*[]string)
	offchainLookup.callData = *abi.ConvertType(errorParameters[2], new([]byte)).(*[]byte)
	offchainLookup.callbackFunction = *abi.ConvertType(errorParameters[3], new([4]byte)).(*[4]byte)
	offchainLookup.extraData = *abi.ConvertType(errorParameters[4], new([]byte)).(*[]byte)
	return offchainLookup, nil
}

func (r *evmRegistryv2_0) offchainLookupCallback(offchainLookup OffchainLookup, offchainResp []byte, upkeepInfo keeper_registry_wrapper2_0.UpkeepInfo, opts *bind.CallOpts) (bool, []byte, error) {
	// call to the contract function specified by the 4-byte selector callbackFunction, supplying the data returned and extraData
	typ, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return false, nil, errors.Wrapf(err, "abi new type error:")
	}
	callbackArgs := abi.Arguments{
		{Name: "response", Type: typ},
		{Name: "extraData", Type: typ},
	}
	pack, err := callbackArgs.Pack(offchainResp, offchainLookup.extraData)
	if err != nil {
		return false, nil, errors.Wrapf(err, "callback args pack error:")
	}

	var callbackPayload []byte
	callbackPayload = append(callbackPayload, offchainLookup.callbackFunction[:]...)
	callbackPayload = append(callbackPayload, pack...)

	checkUpkeepGasLimit := uint32(200000) + uint32(6500000) + uint32(300000) + upkeepInfo.ExecuteGas
	callbackMsg := ethereum.CallMsg{
		From: r.address,          // registry addr
		To:   &upkeepInfo.Target, // upkeep addr
		Gas:  uint64(checkUpkeepGasLimit),
		Data: hexutil.Bytes(callbackPayload), // function callbackFunc(bytes calldata resp, bytes calldata extra) external view returns (bool, bytes memory)
	}

	callbackResp, err := r.evmClient.CallContract(context.Background(), callbackMsg, opts.BlockNumber)
	if err != nil {
		return false, nil, errors.Wrapf(err, "call contract callback error:")
	}

	boolTyp, err := abi.NewType("bool", "", nil)
	callbackOutput := abi.Arguments{
		{Name: "upkeepNeeded", Type: boolTyp},
		{Name: "performData", Type: typ},
	}
	values, err := callbackOutput.Unpack(callbackResp)
	if err != nil {
		return false, nil, errors.Wrapf(err, "callback ouput unpack error:")
	}

	upkeepNeeded := *abi.ConvertType(values[0], new(bool)).(*bool)
	if !upkeepNeeded {
		return false, nil, nil
	}
	performData := *abi.ConvertType(values[1], new([]byte)).(*[]byte)
	return true, performData, nil
}

// Query - do off chain lookup query. section from eip-3668
// 4 - Construct a request URL by replacing sender with the lowercase 0x-prefixed hexadecimal formatted sender parameter, and replacing data with the the 0x-prefixed hexadecimal formatted callData parameter. The client may choose which URLs to try in which order, but SHOULD prioritise URLs earlier in the list over those later in the list.
// 5 - Make an HTTP GET request to the request URL.
// 6 - If the response code from step (5) is in the range 400-499, return an error to the caller and stop.
// 7 - If the response code from step (5) is in the range 500-599, go back to step (5) and pick a different URL, or stop if there are no further URLs to try.
func (o *OffchainLookup) query() ([]byte, error) {
	senderString := strings.ToLower(o.sender.Hex())
	callDataString := hex.EncodeToString(o.callData)

	fmt.Println("~~~", "len(o.urls)", len(o.urls))

	for i, url := range o.urls {
		fmt.Println("~~~", "url", url)
		resp, statusCode, err := o.doRequest(url, senderString, callDataString)
		if err != nil {
			// either an error or a 4XX response
			err = errors.Wrapf(err, "error with query. statusCode: %d ;url: %s ;sender: %s ;callData: %s", statusCode, url, senderString, callDataString)
			return nil, err
		}
		if statusCode <= 299 {
			// success a 2XX response
			fmt.Printf("succesful offchain lookup on index %d with urls: %v\n", i, o.urls)
			return resp, nil
		}
		// continue trying next url
	}

	// If no successful response was received, return an error
	return nil, errors.Errorf("offchain lookup failed: %v", o.urls)
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
	isGET := strings.Contains(url, "{data}")
	queryUrl := strings.Replace(url, "{sender}", senderString, 1)
	queryUrl = strings.Replace(queryUrl, "{data}", callDataString, 1)
	fmt.Println("url: ", queryUrl)

	// Construct a request URL by replacing sender with the lowercase 0x-prefixed hexadecimal formatted sender parameter, and replacing data with the 0x-prefixed hexadecimal formatted callData parameter.
	client := http.Client{}
	var req *http.Request
	var err error
	if isGET {
		queryUrl = strings.Replace(url, "{data}", callDataString, 1)
		req, err = http.NewRequest("GET", queryUrl, nil)
		if err != nil {
			return nil, 0, errors.Wrapf(err, "get request error:")
		}

	} else {
		body := OffchainLookupBody{
			sender: senderString,
			data:   callDataString,
		}
		jsonBody, _ := json.Marshal(body)
		req, err = http.NewRequest("POST", queryUrl, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, 0, errors.Wrapf(err, "post request error:")
		}
	}
	// Make an HTTP GET request to the request URL.
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "do request error:")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "read body error:")
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
