package chain

import (
	"math/big"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

// TODO (AUTO-2014), find a better place for these concrete types than chain package
type BlockKey string

func (k BlockKey) BigInt() (*big.Int, bool) {
	return big.NewInt(0).SetString(string(k), 10)
}

func (k BlockKey) After(kk types.BlockKey) (bool, error) {
	a, ok := big.NewInt(0).SetString(k.String(), 10)
	if !ok {
		return false, ErrBlockKeyNotParsable
	}

	b, ok := big.NewInt(0).SetString(kk.String(), 10)
	if !ok {
		return false, ErrBlockKeyNotParsable
	}

	gt := a.Cmp(b)
	return gt > 0, nil
}

func (k BlockKey) String() string {
	return string(k)
}
