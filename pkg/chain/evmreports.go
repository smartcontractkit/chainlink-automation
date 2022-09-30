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

	type w struct {
		CheckBlockNumber uint32   `abi:"checkBlockNumber"`
		CheckBlockhash   [32]byte `abi:"checkBlockhash"`
		PerformData      []byte   `abi:"performData"`
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
	data := make([]w, len(toReport))

	for i, result := range toReport {
		_, upkeepId, err := blockAndIdFromKey(result.Key)
		if err != nil {
			return nil, fmt.Errorf("%w: report encoding error", err)
		}

		ids[i] = upkeepId
		data[i] = w{
			CheckBlockNumber: result.CheckBlockNumber,
			CheckBlockhash:   result.CheckBlockHash,
			PerformData:      result.PerformData,
		}
	}

	bts, err := reportArgs.Pack(fastGas, link, ids, data)
	if err != nil {
		return []byte{}, fmt.Errorf("%w: failed to pack report data", err)
	}

	//return append([]byte("0x"), bts...), nil
	return bts, nil
}
