package config

import "math/big"

const (
	AllExpected  = "all"
	NoneExpected = "none"
)

type EventType string

const (
	OCR3ConfigEventType     EventType = "ocr3config"
	GenerateUpkeepEventType EventType = "generateUpkeeps"
	LogTriggerEventType     EventType = "logTrigger"
)

type Event struct {
	Type         EventType `json:"type"`
	TriggerBlock *big.Int  `json:"eventBlockNumber"`
	Comment      string    `json:"comment,omitempty"`
}

type UpkeepType string

const (
	ConditionalUpkeepType UpkeepType = "conditional"
	LogTriggerUpkeepType  UpkeepType = "logTrigger"
)

// OCR3ConfigEvent is an event that indicates a new OCR config should be
// broadcast. Consult libOCR for descriptions of each config var.
type OCR3ConfigEvent struct {
	Event
	// MaxFaultyNodesF is the configurable faulty number of nodes
	MaxFaultyNodesF int `json:"maxFaultyNodes"`
	// Offchain is the encoded off chain config data. Typically this is JSON
	// encoded for the CLAutomation plugin.
	Offchain string `json:"encodedOffchainConfig"`
	// Rmax is the maximum number of rounds in an epoch
	Rmax uint64 `json:"maxRoundsPerEpoch"`
	// DeltaProgress is the OCR setting for round leader progress before forcing
	// a new epoch and leader
	DeltaProgress Duration `json:"deltaProgress"`
	DeltaResend   Duration `json:"deltaResend"`
	DeltaInitial  Duration `json:"deltaInitial"`
	// DeltaRound is the approximate time a round should complete in
	DeltaRound   Duration `json:"deltaRound"`
	DeltaGrace   Duration `json:"deltaGrace"`
	DeltaRequest Duration `json:"deltaCertifiedCommitRequest"`
	// DeltaStage is the time OCR waits before attempting a followup transmit
	DeltaStage Duration `json:"deltaStage"`
	MaxQuery   Duration `json:"maxQueryTime"`
	// MaxObservation is the maximum amount of time to provide observation to complete
	MaxObservation Duration `json:"maxObservationTime"`
	MaxAccept      Duration `json:"maxShouldAcceptTime"`
	MaxTransmit    Duration `json:"maxShouldTransmitTime"`
}

// GenerateUpkeepEvent is a configuration for creating upkeeps in bulk.
type GenerateUpkeepEvent struct {
	Event
	// Count is the total number of upkeeps to create for this event.
	Count int `json:"count"`
	// StartID is the numeric id on which to begin incrementing for the upkeep
	// id. Conflicting ids with multiple generate events will result in a
	// config error.
	StartID *big.Int `json:"startID"`
	// EligibilityFunc is a basic linear function for which to indicate
	// eligibility. This can be seen as the cadence on which each upkeep becomes
	// eligible. The values 'always' and 'never' are also valid. Empty is
	// assumed to be 'never'. Using 'always' for conditional upkeeps is invalid.
	EligibilityFunc string `json:"eligibilityFunc,omitempty"`
	// OffsetFunc is a basic linear function that determines the block reference
	// on which to apply the eligibility function. Each generated upkeep can
	// follow the same eligibility function, but start at different blocks
	// determined by the offset function. For eligibility 'always' or 'never' it
	// is preferable to leave this field empty.
	OffsetFunc string `json:"offsetFunc,omitempty"`
	// UpkeepType defines whether the generated upkeeps will be configured as
	// conditional or log trigger upkeeps.
	UpkeepType UpkeepType `json:"upkeepType"`
	// LogTriggeredBy is the log value on which to trigger the set of generated
	// upkeeps. Only applies to log trigger type upkeeps. An empty value for a
	// log triggered upkeep will result in the upkeep never being triggered.
	LogTriggeredBy string `json:"logTriggeredBy,omitempty"`
	// Expected provides customizations to upkeep perform assertions. By default
	// all eligible upkeeps are expected to be performed where the default value
	// in this configuration is 'all'. The alternative is 'none' where none of
	// generated upkeeps are expected to perform. Use the latter when creating
	// upkeeps that should perform per the eligibility configuration, but will
	// not perform due to some other network concerns such as too high network
	// delay or something that might disable the OCR3 protocol.
	Expected string `json:"expected,omitempty"`
}

// LogTriggerEvent is a configuration for simulating logs emitted from a chain
// source. Each log originates in a specified block and is defined by the
// trigger value.
type LogTriggerEvent struct {
	Event
	// TriggerValue corresponds to log trigger upkeeps with 'LogTriggeredBy' set.
	TriggerValue string `json:"triggerValue"`
}
