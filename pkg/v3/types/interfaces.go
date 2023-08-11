package types

import (
	"context"
)

type UpkeepTypeGetter func(uid UpkeepIdentifier) UpkeepType

//go:generate mockery --name Encoder --structname MockEncoder --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types" --case underscore --filename encoder.generated.go
type Encoder interface {
	Encode(...CheckResult) ([]byte, error)
	Extract([]byte) ([]ReportedUpkeep, error)
}

//go:generate mockery --name LogEventProvider --structname MockLogEventProvider --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types" --case underscore --filename logeventprovider.generated.go
type LogEventProvider interface {
	GetLatestPayloads(context.Context) ([]UpkeepPayload, error)
}

//go:generate mockery --name RecoverableProvider --structname MockRecoverableProvider --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types" --case underscore --filename recoverableprovider.generated.go
type RecoverableProvider interface {
	GetRecoveryProposals(context.Context) ([]UpkeepPayload, error)
}

//go:generate mockery --name TransmitEventProvider --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types" --case underscore --filename transmit_event_provider.generated.go
type TransmitEventProvider interface {
	GetLatestEvents(context.Context) ([]TransmitEvent, error)
}

//go:generate mockery --name ConditionalUpkeepProvider --structname MockConditionalUpkeepProvider --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types" --case underscore --filename conditionalupkeepprovider.generated.go
type ConditionalUpkeepProvider interface {
	GetActiveUpkeeps(context.Context) ([]UpkeepPayload, error)
}

//go:generate mockery --name PayloadBuilder --structname MockPayloadBuilder --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types" --case underscore --filename payloadbuilder.generated.go
type PayloadBuilder interface {
	// Can get payloads for a subset of proposals along with an error
	BuildPayloads(context.Context, ...CoordinatedProposal) ([]UpkeepPayload, error)
}

//go:generate mockery --name Runnable --structname MockRunnable --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types" --case underscore --filename runnable.generated.go
type Runnable interface {
	// Can get results for a subset of payloads along with an error
	CheckUpkeeps(context.Context, ...UpkeepPayload) ([]CheckResult, error)
}

type BlockSubscriber interface {
	// Subscribe provides an identifier integer, a new channel, and potentially an error
	Subscribe() (int, chan BlockHistory, error)
	// Unsubscribe requires an identifier integer and indicates the provided channel should be closed
	Unsubscribe(int) error
}

//go:generate mockery --name UpkeepStateUpdater --structname MockUpkeepStateUpdater --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types" --case underscore --filename upkeep_state_updater.generated.go
type UpkeepStateUpdater interface {
	SetUpkeepState(context.Context, CheckResult, UpkeepState) error
}

type RetryQueue interface {
	// Enqueue adds new items to the queue
	Enqueue(items ...UpkeepPayload) error
	// Dequeue returns the next n items in the queue, considering retry time schedules
	Dequeue(n int) ([]UpkeepPayload, error)
}

type ProposalQueue interface {
	// Enqueue adds new items to the queue
	Enqueue(items ...CoordinatedProposal) error
	// Dequeue returns the next n items in the queue, considering retry time schedules
	Dequeue(t UpkeepType, n int) ([]CoordinatedProposal, error)
}

//go:generate mockery --name ResultStore --structname MockResultStore --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types" --case underscore --filename result_store.generated.go
type ResultStore interface {
	Add(...CheckResult)
	Remove(...string)
	View() ([]CheckResult, error)
}

//go:generate mockery --name Coordinator --structname MockCoordinator --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types" --case underscore --filename coordinator.generated.go
type Coordinator interface {
	PreProcess(_ context.Context, payloads []UpkeepPayload) ([]UpkeepPayload, error)

	Accept(ReportedUpkeep) bool
	ShouldTransmit(ReportedUpkeep) bool
	FilterResults([]CheckResult) ([]CheckResult, error)
	FilterProposals([]CoordinatedProposal) ([]CoordinatedProposal, error)
}

//go:generate mockery --name MetadataStore --structname MockMetadataStore --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types" --case underscore --filename metadatastore.generated.go
type MetadataStore interface {
	SetBlockHistory(BlockHistory)
	GetBlockHistory() BlockHistory

	// TODO AddProposals(proposals ...CoordinatedProposal)
	ViewProposals(utype UpkeepType) []CoordinatedProposal
	// TODO RemoveProposals(proposals ...CoordinatedProposal)

	AddLogRecoveryProposal(...CoordinatedProposal)
	ViewLogRecoveryProposal() []CoordinatedProposal
	RemoveLogRecoveryProposal(...CoordinatedProposal)

	AddConditionalProposal(...CoordinatedProposal)
	ViewConditionalProposal() []CoordinatedProposal
	RemoveConditionalProposal(...CoordinatedProposal)

	Start(context.Context) error
	Close() error
}

//go:generate mockery --name Ratio --structname MockRatio --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types" --case underscore --filename ratio.generated.go
type Ratio interface {
	// OfInt should return n out of x such that n/x ~ r (ratio)
	OfInt(int) int
}
