package v1

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	encoders "github.com/smartcontractkit/ocr2keepers/encoder"
	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

const (
	separator = "|"
)

var (
	errUpkeepKeyNotParsable = fmt.Errorf("upkeep key not parsable")
)

type wrappedPerform struct {
	CheckBlockNumber uint32   `abi:"checkBlockNumber" json:"checkBlockNumber"`
	CheckBlockhash   [32]byte `abi:"checkBlockhash" json:"checkBlockhash"`
	PerformData      []byte   `abi:"performData" json:"performData"`
}

type reportPacker interface {
	Pack(args ...interface{}) ([]byte, error)
	UnpackIntoMap(v map[string]interface{}, data []byte) error
}

type encoder struct {
	reportKeys []string
	packer     reportPacker
}

func NewEncoder() *encoder {
	return &encoder{
		reportKeys: []string{"fastGasWei", "linkNative", "upkeepIds", "wrappedPerformDatas"},
		packer: abi.Arguments{
			{Name: "fastGasWei", Type: chain.Uint256},
			{Name: "linkNative", Type: chain.Uint256},
			{Name: "upkeepIds", Type: chain.Uint256Arr},
			{Name: "wrappedPerformDatas", Type: chain.PerformDataArr},
		},
	}
}

// ValidateUpkeepKey returns true if the types.UpkeepKey is valid, false otherwise
func (e *encoder) ValidateUpkeepKey(key types.UpkeepKey) (bool, error) {
	blockKey, upkeepIdentifier, err := e.SplitUpkeepKey(key)
	if err != nil {
		return false, err
	}

	_, err = e.ValidateBlockKey(blockKey)
	if err != nil {
		return false, err
	}

	_, err = e.ValidateUpkeepIdentifier(upkeepIdentifier)
	if err != nil {
		return false, err
	}

	return true, nil
}

// ValidateUpkeepIdentifier returns true if the types.UpkeepIdentifier is valid, false otherwise
func (e *encoder) ValidateUpkeepIdentifier(identifier types.UpkeepIdentifier) (bool, error) {
	identifierInt, ok := identifier.BigInt()
	if !ok {
		return false, fmt.Errorf("upkeep identifier is not a big int")
	}
	if identifierInt.Cmp(big.NewInt(0)) == -1 {
		return false, fmt.Errorf("upkeep identifier is not a positive integer")
	}
	return true, nil
}

// ValidateBlockKey returns true if the types.BlockKey is valid, false otherwise
func (e *encoder) ValidateBlockKey(key types.BlockKey) (bool, error) {
	keyInt, ok := key.BigInt()
	if !ok {
		return false, fmt.Errorf("block key is not a big int")
	}
	if keyInt.Cmp(big.NewInt(0)) == -1 {
		return false, fmt.Errorf("block key is not a positive integer")
	}
	return true, nil
}

// MakeUpkeepKey creates a new types.UpkeepKey from a types.BlockKey and a types.UpkeepIdentifier
func (e *encoder) MakeUpkeepKey(blockKey types.BlockKey, upkeepIdentifier types.UpkeepIdentifier) types.UpkeepKey {
	return chain.UpkeepKey(fmt.Sprintf("%s%s%s", blockKey, separator, string(upkeepIdentifier)))
}

// SplitUpkeepKey splits a types.UpkeepKey into its constituent types.BlockKey and types.UpkeepIdentifier parts
func (e *encoder) SplitUpkeepKey(upkeepKey types.UpkeepKey) (types.BlockKey, types.UpkeepIdentifier, error) {
	if upkeepKey == nil {
		return nil, nil, fmt.Errorf("%w: missing data in upkeep key", errUpkeepKeyNotParsable)
	}
	components := strings.Split(upkeepKey.String(), separator)
	if len(components) != 2 {
		return nil, nil, fmt.Errorf("%w: missing data in upkeep key", errUpkeepKeyNotParsable)
	}

	return chain.BlockKey(components[0]), types.UpkeepIdentifier(components[1]), nil
}

// EncodeReport encodes a report from a list of upkeep results
func (e *encoder) EncodeReport(toReport []types.UpkeepResult, _ ...encoders.Config) ([]byte, error) {
	if len(toReport) == 0 {
		return nil, nil
	}

	var baseValuesIdx int
	for i, rpt := range toReport {
		if rpt.CheckBlockNumber > uint32(baseValuesIdx) {
			baseValuesIdx = i
		}
	}

	fastGas := toReport[baseValuesIdx].FastGasWei
	if fastGas == nil {
		return nil, fmt.Errorf("missing FastGasWei")
	}
	link := toReport[baseValuesIdx].LinkNative
	if link == nil {
		return nil, fmt.Errorf("missing LinkNative")
	}
	ids := make([]*big.Int, len(toReport))
	data := make([]wrappedPerform, len(toReport))

	for i, result := range toReport {
		_, upkeepId, err := e.SplitUpkeepKey(result.Key)
		if err != nil {
			return nil, fmt.Errorf("%w: report encoding error", err)
		}

		upkeepIdInt, ok := upkeepId.BigInt()
		if !ok {
			return nil, errUpkeepKeyNotParsable
		}

		ids[i] = upkeepIdInt
		data[i] = wrappedPerform{
			CheckBlockNumber: result.CheckBlockNumber,
			CheckBlockhash:   result.CheckBlockHash,
			PerformData:      result.PerformData,
		}
	}

	reportBytes, err := e.packer.Pack(fastGas, link, ids, data)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to pack report data", err)
	}

	return reportBytes, nil
}

// EncodeUpkeepIdentifier returns the types.UpkeepIdentifier of a types.UpkeepResult
func (e *encoder) EncodeUpkeepIdentifier(result types.UpkeepResult) (types.UpkeepIdentifier, error) {
	_, identifier, err := e.SplitUpkeepKey(result.Key)
	return identifier, err
}

// KeysFromReport extracts the upkeep keys from a report
func (e *encoder) KeysFromReport(report []byte) ([]types.UpkeepKey, error) {
	reportMap := make(map[string]interface{})
	if err := e.packer.UnpackIntoMap(reportMap, report); err != nil {
		return nil, err
	}

	for _, reportKey := range e.reportKeys {
		if _, ok := reportMap[reportKey]; !ok {
			return nil, fmt.Errorf("decoding error: %s missing from struct", reportKey)
		}
	}

	upkeepIds, ok := reportMap[e.reportKeys[2]].([]*big.Int)
	if !ok {
		return nil, fmt.Errorf("upkeep ids of incorrect type in report")
	}

	performs, ok := reportMap[e.reportKeys[3]].([]struct {
		CheckBlockNumber uint32   `json:"checkBlockNumber"`
		CheckBlockhash   [32]byte `json:"checkBlockhash"`
		PerformData      []byte   `json:"performData"`
	})
	if !ok {
		return nil, fmt.Errorf("unable to read wrappedPerformDatas")
	}

	if len(upkeepIds) != len(performs) {
		return nil, fmt.Errorf("upkeep ids and performs should have matching length")
	}

	res := make([]types.UpkeepKey, len(upkeepIds))
	for i, upkeepID := range upkeepIds {
		res[i] = chain.NewUpkeepKey(big.NewInt(int64(performs[i].CheckBlockNumber)), upkeepID)
	}

	return res, nil
}

// Eligible returns whether an upkeep result is eligible
func (e *encoder) Eligible(result types.UpkeepResult) (bool, error) {
	return result.State == types.Eligible, nil
}
