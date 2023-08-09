package types

import (
	"bytes"
	"fmt"
)

type Trigger struct {
	// BlockNumber is the block number in which the trigger was checked
	BlockNumber BlockNumber
	// BlockHash is the block hash in which the trigger was checked
	BlockHash [32]byte
	// LogTriggerExtension is the extension for log triggers
	LogTriggerExtension *LogTriggerExtension
}

func NewTrigger(blockNumber BlockNumber, blockHash [32]byte) Trigger {
	return Trigger{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
	}
}

func NewLogTrigger(blockNumber BlockNumber, blockHash [32]byte, txHash [32]byte, index uint32) Trigger {
	return Trigger{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
		LogTriggerExtension: &LogTriggerExtension{
			TxHash: txHash,
			Index:  index,
		},
	}
}

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

type LogTriggerExtension struct {
	// LogTxHash is the transaction hash of the log event
	TxHash [32]byte
	// Index is the index of the log event in the transaction
	Index uint32
	// BlockHash is the block hash in which the event occurred
	BlockHash [32]byte
	// BlockNumber is the block number in which the event occurred
	BlockNumber BlockNumber
}

func (e LogTriggerExtension) LogIdentifier() []byte {
	return bytes.Join([][]byte{
		e.TxHash[:],
		[]byte(fmt.Sprintf("%d", e.Index)),
	}, []byte{})
}

func (e LogTriggerExtension) Validate() error {
	if len(e.TxHash) == 0 {
		return fmt.Errorf("log transaction hash cannot be empty")
	}
	if e.Index == 0 {
		return fmt.Errorf("log index cannot be zero")
	}
	// not checking block hash or block number because they might not be available
	// in case we extract ReportedUpkeep from a report, we won't have these fields

	return nil
}
