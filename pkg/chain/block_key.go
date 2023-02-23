package chain

import (
	"fmt"
	"math/big"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

// TODO (AUTO-2014), find a better place for these concrete types than chain package
type BlockKey string

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

func (k BlockKey) BigInt() (*big.Int, bool) {
	return big.NewInt(0).SetString(string(k), 10)
}

func (k BlockKey) Subtract(x int) (string, error) {
	a, ok := big.NewInt(0).SetString(k.String(), 10)
	if !ok {
		return "", ErrBlockKeyNotParsable
	}
	a.Sub(a, big.NewInt(int64(x)))
	if a.Cmp(big.NewInt(0)) == -1 {
		return "", fmt.Errorf("subtraction resulted in negative block key")
	}
	return a.String(), nil
}
