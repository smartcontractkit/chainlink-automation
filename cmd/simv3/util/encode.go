package util

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

const (
	// upkeepTypeStartIndex is the index where the upkeep type bytes start.
	// for 2.1 we use 11 zeros (reserved bytes for future use)
	// and 1 byte to represent the type, with index equal upkeepTypeByteIndex
	upkeepTypeStartIndex = 4
	// upkeepTypeByteIndex is the index of the byte that holds the upkeep type.
	upkeepTypeByteIndex = 15
)

var (
	ErrInvalidUpkeepID = fmt.Errorf("invalid upkeepID")
)

func EncodeCheckResultsToReportBytes(results []ocr2keepers.CheckResult) ([]byte, error) {
	if len(results) == 0 {
		return []byte{}, nil
	}

	bts, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal check results: %w", err)
	}

	return bts, nil
}

func DecodeCheckResultsFromReportBytes(bts []byte) ([]ocr2keepers.CheckResult, error) {
	if len(bts) == 0 {
		return []ocr2keepers.CheckResult{}, nil
	}

	var results []ocr2keepers.CheckResult

	if err := json.Unmarshal(bts, &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal check results from bytes: %w", err)
	}

	return results, nil
}

// GetUpkeepType returns the upkeep type from the given ID.
// it follows the same logic as the contract, but performs it locally.
func GetUpkeepType(id ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
	for i := upkeepTypeStartIndex; i < upkeepTypeByteIndex; i++ {
		if id[i] != 0 { // old id
			return ocr2keepers.ConditionTrigger
		}
	}

	typeByte := id[upkeepTypeByteIndex]

	return ocr2keepers.UpkeepType(typeByte)
}

func UpkeepWorkID(uid ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
	var triggerExtBytes []byte

	if trigger.LogTriggerExtension != nil {
		triggerExtBytes = trigger.LogTriggerExtension.LogIdentifier()
	}

	hash := crypto.Keccak256(append(uid[:], triggerExtBytes...))

	return hex.EncodeToString(hash[:])
}
