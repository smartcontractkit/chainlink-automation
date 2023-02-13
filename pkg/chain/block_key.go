package chain

import (
	"math/big"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type BlockKey string

func (k BlockKey) After(kk types.BlockKey) (bool, error) {
	a := big.NewInt(0)
	a, ok := a.SetString(k.String(), 10)
	if !ok {
		return false, ErrBlockKeyNotParsable
	}

	b := big.NewInt(0)
	b, ok = b.SetString(kk.String(), 10)
	if !ok {
		return false, ErrBlockKeyNotParsable
	}

	if gt := a.Cmp(b); gt > 0 {
		return true, nil
	}
	return false, nil
}

func (k BlockKey) String() string {
	return string(k)
}
