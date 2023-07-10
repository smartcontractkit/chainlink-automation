package simulators

import (
	"math/big"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

type SimulatedResult struct {
	ID               string
	UpkeepID         ocr2keepers.UpkeepIdentifier
	CheckBlock       ocr2keepers.BlockKey
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
