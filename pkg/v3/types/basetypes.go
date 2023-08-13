package types

import (
	"fmt"
	"math/big"
)

type UpkeepType uint8

const (
	// Exploratory AUTO 4335: add type for unknown
	ConditionTrigger UpkeepType = iota
	LogTrigger
)

type TransmitEventType int

const (
	UnknownEvent TransmitEventType = iota
	PerformEvent
	StaleReportEvent
	ReorgReportEvent
	InsufficientFundsReportEvent
)

// UpkeepState is a final state of some unit of work.
type UpkeepState uint8

const (
	UnknownState UpkeepState = iota
	// Performed means the upkeep was performed
	Performed
	// Ineligible means the upkeep was not eligible to be performed
	Ineligible
)

// UpkeepIdentifier is a unique identifier for the upkeep, represented as uint256 in the contract.
type UpkeepIdentifier [32]byte

func (u UpkeepIdentifier) String() string {
	return u.BigInt().String()
}

func (u UpkeepIdentifier) BigInt() *big.Int {
	return big.NewInt(0).SetBytes(u[:])
}

// FromBigInt sets the upkeep identifier from a big.Int,
// returning true if the big.Int is valid and false otherwise.
// in case of an invalid big.Int the upkeep identifier is set to 32 zeros.
func (u *UpkeepIdentifier) FromBigInt(i *big.Int) bool {
	*u = [32]byte{}
	if i.Cmp(big.NewInt(0)) == -1 {
		return false
	}
	b := i.Bytes()
	if len(b) == 0 {
		return true
	}
	if len(b) <= 32 {
		copy(u[32-len(b):], i.Bytes())
		return true
	}
	return false
}

type BlockNumber uint64

// BlockKey represent a block (number and hash)
// NOTE: This struct is sent on the p2p network as part of observations to get quorum
// Any change here should be backwards compatible and should keep validation and
// quorum requirements in mind
type BlockKey struct {
	Number BlockNumber
	Hash   [32]byte
}

type TransmitEvent struct {
	// Type describes the type of event
	Type TransmitEventType
	// TransmitBlock is the block height of the transmit event
	TransmitBlock BlockNumber
	// Confirmations is the block height behind latest
	Confirmations int64
	// TransactionHash is the hash for the transaction where the event originated
	TransactionHash [32]byte
	// UpkeepID uniquely identifies the upkeep in the registry
	UpkeepID UpkeepIdentifier
	// WorkID uniquely identifies the unit of work for the specified upkeep
	WorkID string
	// CheckBlock is the block value that the upkeep was originally checked at
	CheckBlock BlockNumber
}

// NOTE: This struct is sent on the p2p network as part of observations to get quorum
// Any change here should be backwards compatible and should keep validation and
// quorum requirements in mind
type CheckResult struct {
	// zero if success, else indicates an error code
	PipelineExecutionState uint8
	// if PipelineExecutionState is non zero, then retryable indicates that the same
	// payload can be processed again in order to get a successful execution
	Retryable bool
	// Rest of these fields are only applicable if PipelineExecutionState is zero
	// Eligible indicates whether this result is eligible to be performed
	Eligible bool
	// If result is not eligible then the reason it failed. Should be 0 if eligible
	IneligibilityReason uint8
	// Upkeep is all the information that identifies the upkeep
	UpkeepID UpkeepIdentifier
	// Trigger is the event that triggered the upkeep to be checked
	Trigger Trigger
	// WorkID represents the unit of work for the check result
	// Exploratory: Make workID an internal field and an external WorkID() function which generates WID
	WorkID string
	// GasAllocated is the gas to provide an upkeep in a report
	GasAllocated uint64
	// PerformData is the raw data returned when simulating an upkeep perform
	PerformData []byte
	// todo: add comment
	FastGasWei *big.Int
	// todo: add comment
	LinkNative *big.Int
}

// UniqueID returns a unique identifier for the check result.
// It is used to achieve quorum on results before being sent within a report.
func (r CheckResult) UniqueID() string {
	var resultBytes []byte
	// TODO: Discuss if we should keep all fields here for simplicity and avoiding
	// undefined behaviour for other fields when achieveing quorum on this struct
	// json.marshall might be just simpler? (but slower)
	// Alternatively the rest of the fields can be zeroed out for consistent behaviour
	resultBytes = append(resultBytes, r.PipelineExecutionState)
	resultBytes = append(resultBytes, []byte(fmt.Sprintf("%+v", r.Retryable))...)
	resultBytes = append(resultBytes, []byte(fmt.Sprintf("%+v", r.Eligible))...)
	resultBytes = append(resultBytes, r.IneligibilityReason)
	resultBytes = append(resultBytes, r.UpkeepID[:]...)
	resultBytes = append(resultBytes, r.Trigger.BlockHash[:]...)
	resultBytes = append(resultBytes, big.NewInt(int64(r.Trigger.BlockNumber)).Bytes()...)
	if r.Trigger.LogTriggerExtension != nil {
		// Note: We encode the whole trigger extension so the behaiour of
		// LogTriggerExtentsion.BlockNumber and LogTriggerExtentsion.BlockHash should be
		// consistent across nodes
		resultBytes = append(resultBytes, []byte(fmt.Sprintf("%+v", r.Trigger.LogTriggerExtension))...)
	}
	resultBytes = append(resultBytes, r.WorkID[:]...)
	resultBytes = append(resultBytes, big.NewInt(int64(r.GasAllocated)).Bytes()...)
	resultBytes = append(resultBytes, r.PerformData[:]...)
	resultBytes = append(resultBytes, r.FastGasWei.Bytes()...)
	resultBytes = append(resultBytes, r.LinkNative.Bytes()...)
	return fmt.Sprintf("%x", resultBytes)
}

// BlockHistory is a list of block keys
type BlockHistory []BlockKey

func (bh BlockHistory) Latest() (BlockKey, error) {
	if len(bh) == 0 {
		return BlockKey{}, fmt.Errorf("empty block history")
	}

	return bh[0], nil
}

type UpkeepPayload struct {
	// Upkeep is all the information that identifies the upkeep
	UpkeepID UpkeepIdentifier
	// Trigger is the event that triggered the upkeep to be checked
	Trigger Trigger
	// WorkID uniquely identifies the unit of work for the specified upkeep
	WorkID string
	// CheckData is the data used to check the upkeep
	CheckData []byte
}

// CoordinatedBlockProposal is used to represent a unit of work that can be performed
// after a check block has been coordinated between nodes.
// NOTE: This struct is sent on the p2p network as part of observations to get quorum
// Any change here should be backwards compatible and should keep validation and
// quorum requirements in mind.
// NOTE: Only the trigger.BlockHash and trigger.BlockNumber are coordinated across
// the network to get a quorum. Rest of the fields here SHOULD NOT BE TRUSTED as they
// can be manipulated by a single malicious node.
type CoordinatedBlockProposal struct {
	// UpkeepID is the id of the proposed upkeep
	UpkeepID UpkeepIdentifier
	// Trigger represents the event that triggered the upkeep to be checked
	Trigger Trigger
	// WorkID represents the unit of work for the coordinated proposal
	WorkID string
}

// ReportedUpkeep contains details of an upkeep for which a report was generated.
type ReportedUpkeep struct {
	// UpkeepID id of the underlying upkeep
	UpkeepID UpkeepIdentifier
	// Trigger data for the upkeep
	Trigger Trigger
	// WorkID represents the unit of work for the reported upkeep
	WorkID string
}
