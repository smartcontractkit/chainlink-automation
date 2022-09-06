package chain

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"

	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type evmReportEncoder struct{}

func NewEVMReportEncoder() *evmReportEncoder {
	return &evmReportEncoder{}
}

func (b *evmReportEncoder) EncodeReport(toReport []ktypes.UpkeepResult) ([]byte, error) {
	reportArgs := abi.Arguments{
		{Type: mustType(abi.NewType("uint256[]", "", nil))},
		{Type: mustType(abi.NewType("bytes[]", "", nil))},
	}

	ids := make([]*big.Int, len(toReport))
	data := make([][]byte, len(toReport))

	for i, result := range toReport {
		_, upkeepId, err := blockAndIdFromKey(result.Key)
		if err != nil {
			// TODO: maybe this should be a warning??
			continue
		}

		ids[i] = upkeepId
		data[i] = result.PerformData
	}

	return reportArgs.Pack(ids, data)
}

func mustType(tp abi.Type, err error) abi.Type {
	if err != nil {
		panic(err)
	}
	return tp
}
