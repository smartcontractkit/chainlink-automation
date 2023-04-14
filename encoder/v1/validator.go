package v1

import (
	"fmt"
	"math/big"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type validator struct{}

// ValidateUpkeepKey returns true if the types.UpkeepKey is valid, false otherwise
func (v validator) ValidateUpkeepKey(key types.UpkeepKey) (bool, error) {
	blockKey, upkeepIdentifier, err := key.BlockKeyAndUpkeepID()
	if err != nil {
		return false, err
	}

	_, err = v.ValidateBlockKey(blockKey)
	if err != nil {
		return false, err
	}

	_, err = v.ValidateUpkeepIdentifier(upkeepIdentifier)
	if err != nil {
		return false, err
	}

	return true, nil
}

// ValidateUpkeepIdentifier returns true if the types.UpkeepIdentifier is valid, false otherwise
func (v validator) ValidateUpkeepIdentifier(identifier types.UpkeepIdentifier) (bool, error) {
	identifierInt, ok := identifier.BigInt()
	if !ok {
		return false, fmt.Errorf("upkeep identifier is not a big int")
	}
	if identifierInt.Cmp(big.NewInt(0)) == -1 {
		return false, fmt.Errorf("upkeep identifier is not a positive integer")
	}
	return true, nil
}

// ValidateBlockKey returns true if the types.BlockKey is valid, false otherwise
func (v validator) ValidateBlockKey(key types.BlockKey) (bool, error) {
	keyInt, ok := key.BigInt()
	if !ok {
		return false, fmt.Errorf("block key is not a big int")
	}
	if keyInt.Cmp(big.NewInt(0)) == -1 {
		return false, fmt.Errorf("block key is not a positive integer")
	}
	return true, nil
}
