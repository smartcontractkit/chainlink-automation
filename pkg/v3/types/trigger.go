package types

import (
	"bytes"
	"fmt"
)

// Trigger represents a trigger for an upkeep.
// It contains an extension per trigger type, and the block number + hash
// in which the trigger was checked.
// NOTE: This struct is sent on the p2p network as part of observations to get quorum
// Any change here should be backwards compatible and should keep validation and
// quorum requirements in mind
type Trigger struct {
	// BlockNumber is the block number in which the trigger was checked
	BlockNumber BlockNumber
	// BlockHash is the block hash in which the trigger was checked
	BlockHash [32]byte
	// LogTriggerExtension is the extension for log triggers
	LogTriggerExtension *LogTriggerExtension
}

// NewTrigger returns a new basic trigger w/o extension
func NewTrigger(blockNumber BlockNumber, blockHash [32]byte) Trigger {
	return Trigger{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
	}
}

func NewLogTrigger(blockNumber BlockNumber, blockHash [32]byte, logTriggerExtension *LogTriggerExtension) Trigger {
	return Trigger{
		BlockNumber:         blockNumber,
		BlockHash:           blockHash,
		LogTriggerExtension: logTriggerExtension,
	}
}

// Validate validates the trigger fields, and any extensions if present.
func (t Trigger) Validate() error {
	if t.BlockNumber == 0 {
		return fmt.Errorf("block number cannot be zero")
	}
	if len(t.BlockHash) == 0 {
		return fmt.Errorf("block hash cannot be empty")
	}

	if t.LogTriggerExtension != nil {
		if err := t.LogTriggerExtension.Validate(); err != nil {
			return fmt.Errorf("log trigger extension invalid: %w", err)
		}
	}

	return nil
}

// LogTriggerExtension is the extension used for log triggers,
// It contains information of the log event that was triggered.
// NOTE: This struct is sent on the p2p network as part of observations to get quorum
// Any change here should be backwards compatible and should keep validation and
// quorum requirements in mind
type LogTriggerExtension struct {
	// LogTxHash is the transaction hash of the log event
	TxHash [32]byte
	// Index is the index of the log event in the transaction
	Index uint32
	// BlockHash is the block hash in which the event occurred
	// NOTE: This field might be empty. If relying on this field check
	// it is non empty, if it's empty derive from txHash
	BlockHash [32]byte
	// BlockNumber is the block number in which the event occurred
	// NOTE: This field might be empty. If relying on this field check
	// it is non empty, if it's empty derive from txHash
	BlockNumber BlockNumber
}

// LogIdentifier returns a unique identifier for the log event,
// composed of the transaction hash and the log index bytes.
func (e LogTriggerExtension) LogIdentifier() []byte {
	return bytes.Join([][]byte{
		e.TxHash[:],
		[]byte(fmt.Sprintf("%d", e.Index)),
	}, []byte{})
}

// Validate validates the log trigger extension fields.
// NOTE: not checking block hash or block number because they might not be available (e.g. ReportedUpkeep)
func (e LogTriggerExtension) Validate() error {
	if len(e.TxHash) == 0 {
		return fmt.Errorf("log transaction hash cannot be empty")
	}
	if e.Index == 0 {
		return fmt.Errorf("log index cannot be zero")
	}

	return nil
}
