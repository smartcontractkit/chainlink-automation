package types

import "fmt"

type Trigger struct {
	// BlockNumber is the block number in which the trigger was checked
	BlockNumber BlockNumber
	// BlockHash is the block hash in which the trigger was checked
	BlockHash [32]byte
	// LogTriggerExtension is the extension for log triggers
	LogTriggerExtension *LogTriggerExtenstion
}

func NewTrigger(blockNumber BlockNumber, blockHash [32]byte) Trigger {
	return Trigger{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
	}
}

func (t Trigger) String() string {
	// TODO: check if this is the correct format
	return fmt.Sprintf("%d:%s", t.BlockNumber, t.BlockHash)
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

type LogTriggerExtenstion struct {
	// LogTxHash is the transaction hash of the log event
	LogTxHash [32]byte
	// Index is the index of the log event in the transaction
	Index uint32
	// BlockHash is the block hash in which the event occurred
	BlockHash [32]byte
	// BlockNumber is the block number in which the event occurred
	BlockNumber BlockNumber
}

func (e LogTriggerExtenstion) LogIdentifier() string {
	// TODO: check if this is the correct format
	return fmt.Sprintf("%s:%d", e.LogTxHash, e.Index)
}

func (e LogTriggerExtenstion) Validate() error {
	if len(e.LogTxHash) == 0 {
		return fmt.Errorf("log transaction hash cannot be empty")
	}
	if e.Index == 0 {
		return fmt.Errorf("log index cannot be zero")
	}
	if len(e.BlockHash) == 0 {
		return fmt.Errorf("block hash cannot be empty")
	}
	if e.BlockNumber == 0 {
		return fmt.Errorf("block number cannot be zero")
	}

	return nil
}
