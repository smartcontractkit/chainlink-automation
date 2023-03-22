package v1

import (
	"fmt"
	"math/big"

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
	validator
	upkeepProvider
	eligibilityProvider
	reportKeys []string
	packer     reportPacker
}

func NewEncoder() *encoder {
	arguments := abi.Arguments{
		{Name: "fastGasWei", Type: chain.Uint256},
		{Name: "linkNative", Type: chain.Uint256},
		{Name: "upkeepIds", Type: chain.Uint256Arr},
		{Name: "wrappedPerformDatas", Type: chain.PerformDataArr},
	}

	reportKeys := make([]string, len(arguments))
	for i, arg := range arguments {
		reportKeys[i] = arg.Name
	}

	return &encoder{
		reportKeys: reportKeys,
		packer:     arguments,
	}
}

// EncodeReport encodes a report from a list of upkeep results
func (v *encoder) EncodeReport(toReport []types.UpkeepResult, _ ...encoders.Config) ([]byte, error) {
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
		_, upkeepId, err := v.SplitUpkeepKey(result.Key)
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

	reportBytes, err := v.packer.Pack(fastGas, link, ids, data)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to pack report data", err)
	}

	return reportBytes, nil
}

// EncodeUpkeepIdentifier returns the types.UpkeepIdentifier of a types.UpkeepResult
func (v *encoder) EncodeUpkeepIdentifier(result types.UpkeepResult) (types.UpkeepIdentifier, error) {
	_, identifier, err := v.SplitUpkeepKey(result.Key)
	return identifier, err
}

// KeysFromReport extracts the upkeep keys from a report
func (v *encoder) KeysFromReport(report []byte) ([]types.UpkeepKey, error) {
	reportMap := make(map[string]interface{})
	if err := v.packer.UnpackIntoMap(reportMap, report); err != nil {
		return nil, err
	}

	for _, reportKey := range v.reportKeys {
		if _, ok := reportMap[reportKey]; !ok {
			return nil, fmt.Errorf("decoding error: %s missing from struct", reportKey)
		}
	}

	upkeepIds, ok := reportMap[v.reportKeys[2]].([]*big.Int)
	if !ok {
		return nil, fmt.Errorf("upkeep ids of incorrect type in report")
	}

	performs, ok := reportMap[v.reportKeys[3]].([]struct {
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
