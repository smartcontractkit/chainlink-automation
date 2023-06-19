package ocr2keepers

import (
	"strings"
)

type UpkeepIdentifier []byte

type BlockKey string

type UpkeepKey []byte

type UpkeepResult interface{}

func upkeepKeysToString(keys []UpkeepKey) string {
	keysStr := make([]string, len(keys))
	for i, key := range keys {
		keysStr[i] = string(key)
	}

	return strings.Join(keysStr, ", ")
}

type PerformLog struct {
	Key             UpkeepKey
	TransmitBlock   BlockKey
	Confirmations   int64
	TransactionHash string
}

type StaleReportLog struct {
	Key             UpkeepKey
	TransmitBlock   BlockKey
	Confirmations   int64
	TransactionHash string
}

type CheckResult struct {
	Eligible  bool
	Retryable bool
	Payload   UpkeepPayload
}

type ConfiguredUpkeep struct {
	// ID uniquely identifies the upkeep
	ID UpkeepIdentifier
	// Type is the event type required to initiate the upkeep
	Type int
	// Config is configuration data specific to the type
	Config interface{}
}

type UpkeepPayload struct {
	// ID uniquely identifies the upkeep payload
	ID string
	// Upkeep is all the information that identifies the upkeep
	Upkeep ConfiguredUpkeep
	// CheckData is the data used to check the upkeep
	CheckData []byte
	// Tick is the event that triggered the upkeep to be checked
	Tick interface{}
}

type BlockHistory []BlockKey
