package chain

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"

	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type evmReportEncoder struct{}

func NewEVMReportEncoder() *evmReportEncoder {
	return &evmReportEncoder{}
}

var (
	Uint256, _                = abi.NewType("uint256", "", nil)
	Uint256Arr, _             = abi.NewType("uint256[]", "", nil)
	PerformDataMarshalingArgs = []abi.ArgumentMarshaling{
		{Name: "checkBlockNumber", Type: "uint32"},
		{Name: "checkBlockhash", Type: "bytes32"},
		{Name: "performData", Type: "bytes"},
	}
	PerformDataArr, _ = abi.NewType("tuple(uint32,bytes32,bytes)[]", "", PerformDataMarshalingArgs)
)

func (b *evmReportEncoder) EncodeReport(toReport []ktypes.UpkeepResult) ([]byte, error) {
	if len(toReport) == 0 {
		return nil, nil
	}

	reportArgs := abi.Arguments{
		{Name: "fastGasWei", Type: Uint256},
		{Name: "linkNative", Type: Uint256},
		{Name: "upkeepIds", Type: Uint256Arr},
		{Name: "wrappedPerformDatas", Type: PerformDataArr},
	}

	var baseValuesIdx int
	for i, rpt := range toReport {
		if rpt.CheckBlockNumber > uint32(baseValuesIdx) {
			baseValuesIdx = i
		}
	}

	fastGas := toReport[baseValuesIdx].FastGasWei
	link := toReport[baseValuesIdx].LinkNative
	ids := make([]*big.Int, len(toReport))
	data := make([]wrappedPerform, len(toReport))

	for i, result := range toReport {
		_, upkeepId, err := BlockAndIdFromKey(result.Key)
		if err != nil {
			return nil, fmt.Errorf("%w: report encoding error", err)
		}

		ids[i] = upkeepId
		data[i] = wrappedPerform{
			CheckBlockNumber: result.CheckBlockNumber,
			CheckBlockhash:   result.CheckBlockHash,
			PerformData:      result.PerformData,
		}
	}

	bts, err := reportArgs.Pack(fastGas, link, ids, data)
	if err != nil {
		return []byte{}, fmt.Errorf("%w: failed to pack report data", err)
	}

	return bts, nil
}

func (b *evmReportEncoder) IDsFromReport(report []byte) ([]ktypes.UpkeepIdentifier, error) {
	ids := []ktypes.UpkeepIdentifier{}

	reportArgs := abi.Arguments{
		{Name: "fastGasWei", Type: Uint256},
		{Name: "linkNative", Type: Uint256},
		{Name: "upkeepIds", Type: Uint256Arr},
		{Name: "wrappedPerformDatas", Type: PerformDataArr},
	}

	m := make(map[string]interface{})
	err := reportArgs.UnpackIntoMap(m, report)
	if err != nil {
		return ids, err
	}

	rawUkeepIds, ok := m["upkeepIds"]
	if !ok {
		return ids, fmt.Errorf("missing upkeep ids in report")
	}

	rawPerforms, ok := m["wrappedPerformDatas"]
	if !ok {
		return ids, fmt.Errorf("missing wrapped perform data structs in report")
	}

	upkeepIds, ok := rawUkeepIds.([]*big.Int)
	if !ok {
		return ids, fmt.Errorf("upkeep ids of incorrect type in report")
	}

	// TODO: a type assertion on `wrappedPerform` did not work, even with the
	// exact same struct definition as what follows. reflect was used to get the
	// struct definition. not sure yet how to clean this up.
	// ex:
	// t := reflect.TypeOf(rawPerforms)
	// fmt.Printf("%v\n", t)
	performs, ok := rawPerforms.([]struct {
		CheckBlockNumber uint32   `json:"checkBlockNumber"`
		CheckBlockhash   [32]byte `json:"checkBlockhash"`
		PerformData      []byte   `json:"performData"`
	})
	if !ok {
		return ids, fmt.Errorf("performs of incorrect structure in report")
	}

	if len(upkeepIds) != len(performs) {
		return ids, fmt.Errorf("upkeep ids and performs should have matching length")
	}

	ids = make([]ktypes.UpkeepIdentifier, len(upkeepIds))
	for i := 0; i < len(upkeepIds); i++ {
		ids[i] = ktypes.UpkeepIdentifier(upkeepIds[i].String())
	}

	return ids, nil
}

type wrappedPerform struct {
	CheckBlockNumber uint32   `abi:"checkBlockNumber"`
	CheckBlockhash   [32]byte `abi:"checkBlockhash"`
	PerformData      []byte   `abi:"performData"`
}
