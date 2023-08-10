package simulators

import (
	"math/big"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v2"
)

type SimulatedResult struct {
	Key              ocr2keepers.UpkeepKey
	Eligible         bool
	FailureReason    uint8
	GasUsed          *big.Int
	PerformData      []byte
	FastGasWei       *big.Int
	LinkNative       *big.Int
	CheckBlockNumber uint32
	CheckBlockHash   [32]byte
	ExecuteGas       uint32
}
