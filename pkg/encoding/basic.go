package encoding

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

const separator string = "|"

var (
	ErrInvalidBlockKey         = fmt.Errorf("invalid block key")
	ErrInvalidUpkeepIdentifier = fmt.Errorf("invalid upkeep identifier")
	ErrUpkeepKeyNotParsable    = fmt.Errorf("upkeep key not parsable")
)

type BasicEncoder struct{}

// MakeUpkeepKey creates a new types.UpkeepKey from a types.BlockKey and a types.UpkeepIdentifier
func (kb BasicEncoder) MakeUpkeepKey(blockKey ocr2keepers.BlockKey, upkeepIdentifier ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepKey {
	return ocr2keepers.UpkeepKey(fmt.Sprintf("%d%s%s", blockKey.Block, separator, string(upkeepIdentifier)))
}

// SplitUpkeepKey splits a types.UpkeepKey into its constituent types.BlockKey and types.UpkeepIdentifier parts
func (kb BasicEncoder) SplitUpkeepKey(upkeepKey ocr2keepers.UpkeepKey) (ocr2keepers.BlockKey, ocr2keepers.UpkeepIdentifier, error) {
	if upkeepKey == nil {
		return ocr2keepers.BlockKey{}, nil, fmt.Errorf("%w: missing data in upkeep key", ErrUpkeepKeyNotParsable)
	}

	components := strings.Split(string(upkeepKey), separator)
	if len(components) != 2 {
		return ocr2keepers.BlockKey{}, nil, fmt.Errorf("%w: missing data in upkeep key", ErrUpkeepKeyNotParsable)
	}

	block, ok := big.NewInt(0).SetString(components[0], 10)
	if !ok {
		return ocr2keepers.BlockKey{}, nil, fmt.Errorf("%w: missing data in upkeep key", ErrUpkeepKeyNotParsable)
	}
	return ocr2keepers.BlockKey{Block: block.Uint64()}, ocr2keepers.UpkeepIdentifier(components[1]), nil
}

// ValidateUpkeepKey returns true if the types.UpkeepKey is valid, false otherwise
func (kb BasicEncoder) ValidateUpkeepKey(key ocr2keepers.UpkeepKey) (bool, error) {
	blockKey, upkeepIdentifier, err := kb.SplitUpkeepKey(key)
	if err != nil {
		return false, err
	}

	if _, err := kb.ValidateBlockKey(blockKey); err != nil {
		return false, err
	}

	if _, err := kb.ValidateUpkeepIdentifier(upkeepIdentifier); err != nil {
		return false, err
	}

	return true, nil
}

// ValidateUpkeepIdentifier returns true if the types.UpkeepIdentifier is valid, false otherwise
func (kb BasicEncoder) ValidateUpkeepIdentifier(identifier ocr2keepers.UpkeepIdentifier) (bool, error) {
	maxUpkeepIdentifer := new(big.Int)
	maxUpkeepIdentifer, _ = maxUpkeepIdentifer.SetString("115792089237316195423570985008687907853269984665640564039457584007913129639935", 10) // 2 ** 256 -1

	identifierInt, ok := new(big.Int).SetString(string(identifier), 10)
	if !ok {
		return false, fmt.Errorf("%w: upkeep identifier is not a big int", ErrInvalidUpkeepIdentifier)
	}

	if identifierInt.String() != string(identifier) {
		return false, fmt.Errorf("%w: upkeep identifier stringify mismatch", ErrInvalidUpkeepIdentifier)
	}

	if identifierInt.Cmp(big.NewInt(0)) == -1 || identifierInt.Cmp(maxUpkeepIdentifer) > 0 {
		return false, fmt.Errorf("%w: upkeep identifier exceeds lower or upper bounds", ErrInvalidUpkeepIdentifier)
	}

	return true, nil
}

// ValidateBlockKey returns true if the types.BlockKey is valid, false otherwise
func (kb BasicEncoder) ValidateBlockKey(key ocr2keepers.BlockKey) (bool, error) {
	maxBlockNumber := new(big.Int)
	maxBlockNumber, _ = maxBlockNumber.SetString("18446744073709551615", 10) // 2 ** 64 -1

	keyInt := new(big.Int).SetUint64(key.Block)
	if keyInt.Uint64() != key.Block {
		return false, fmt.Errorf("%w: block key stringify mismatch", ErrInvalidBlockKey)
	}

	if keyInt.Cmp(big.NewInt(0)) == -1 || keyInt.Cmp(maxBlockNumber) > 0 {
		return false, fmt.Errorf("%w: block key exceeds lower or upper bounds", ErrInvalidBlockKey)
	}

	return true, nil
}

func (kb BasicEncoder) GetMedian(values []ocr2keepers.BlockKey) ocr2keepers.BlockKey {
	blockNumbers := make([]*big.Int, 0, len(values))
	for _, val := range values {
		in := new(big.Int).SetUint64(val.Block)
		blockNumbers = append(blockNumbers, in)
	}

	sort.Slice(blockNumbers, func(i, j int) bool {
		return blockNumbers[i].Cmp(blockNumbers[j]) < 0
	})

	// this is a crude median calculation; for a list of an odd number of elements, e.g. [10, 20, 30], the center value
	// is chosen as the median. for a list of an even number of elements, a true median calculation would average the
	// two center elements, e.g. [10, 20, 30, 40] = (20 + 30) / 2 = 25, but we want to constrain our median block to
	// one of the block numbers reported, e.g. either 20 or 30. right now we want to choose the higher block number, e.g.
	// 30. for this reason, the logic for selecting the median value from an odd number of elements is the same as the
	// logic for selecting the median value from an even number of elements
	var median *big.Int
	if l := len(blockNumbers); l == 0 {
		median = big.NewInt(0)
	} else {
		median = blockNumbers[l/2]
	}

	return ocr2keepers.BlockKey{Block: median.Uint64()}
}

// After a is after b
func (kb BasicEncoder) After(a ocr2keepers.BlockKey, b ocr2keepers.BlockKey) (bool, error) {
	return a.Block > b.Block, nil
}

// Increment ...
func (kb BasicEncoder) Increment(value ocr2keepers.BlockKey) (ocr2keepers.BlockKey, error) {
	val := new(big.Int).SetUint64(value.Block)

	newVal := new(big.Int).Add(val, big.NewInt(1))

	return ocr2keepers.BlockKey{Block: newVal.Uint64()}, nil
}
