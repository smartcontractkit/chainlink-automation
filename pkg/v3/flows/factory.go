package flows

import (
	"log"
	"time"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func ConditionalTriggerFlows(
	coord ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	ratio ocr2keepers.Ratio,
	getter ocr2keepers.ConditionalUpkeepProvider,
	subscriber ocr2keepers.BlockSubscriber,
	builder ocr2keepers.PayloadBuilder,
	resultStore ocr2keepers.ResultStore,
	metadataStore ocr2keepers.MetadataStore,
	runner ocr2keepersv3.Runner,
	proposalQ ocr2keepers.ProposalQueue,
	retryQ ocr2keepers.RetryQueue,
	stateUpdater ocr2keepers.UpkeepStateUpdater,
	typeGetter ocr2keepers.UpkeepTypeGetter,
	logger *log.Logger,
) ([]service.Recoverable, error) {
	preprocessors := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{coord}

	// runs full check pipeline on a coordinated block with coordinated upkeeps
	conditionalFinal := newFinalConditionalFlow(preprocessors, resultStore, runner, time.Second, proposalQ, builder, retryQ, stateUpdater, logger)

	// the sampling proposal flow takes random samples of active upkeeps, checks
	// them and surfaces the ids if the items are eligible
	conditionalProposal, err := newSampleProposalFlow(preprocessors, ratio, getter, subscriber, metadataStore, runner, typeGetter, logger)
	if err != nil {
		return nil, err
	}

	return []service.Recoverable{conditionalFinal, conditionalProposal}, err
}

func LogTriggerFlows(
	coord ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	resultStore ocr2keepers.ResultStore,
	metadataStore ocr2keepers.MetadataStore,
	runner ocr2keepersv3.Runner,
	logProvider ocr2keepers.LogEventProvider,
	rp ocr2keepers.RecoverableProvider,
	builder ocr2keepers.PayloadBuilder,
	logInterval time.Duration,
	recoveryInterval time.Duration,
	retryQ ocr2keepers.RetryQueue,
	proposals ocr2keepers.ProposalQueue,
	stateUpdater ocr2keepers.UpkeepStateUpdater,
	typeGetter ocr2keepers.UpkeepTypeGetter,
	logger *log.Logger,
) []service.Recoverable {
	// all flows use the same preprocessor based on the coordinator
	// each flow can add preprocessors to this provided slice
	preprocessors := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{coord}

	// the recovery proposal flow is for nodes to surface payloads that should
	// be recovered. these values are passed to the network and the network
	// votes on the proposed values
	rcvProposal := newRecoveryProposalFlow(preprocessors, runner, metadataStore, rp, recoveryInterval, typeGetter, logger)

	// the final recovery flow takes recoverable payloads merged with the latest
	// blocks and runs the pipeline for them. these values to run are derived
	// from node coordination and it can be assumed that all values should be
	// run.
	rcvFinal := newFinalRecoveryFlow(preprocessors, resultStore, runner, retryQ, recoveryInterval, proposals, builder, stateUpdater, logger)

	// the log trigger flow is the happy path for log trigger payloads. all
	// retryables that are encountered in this flow are elevated to the retry
	// flow
	logTrigger := newLogTriggerFlow(preprocessors, resultStore, runner, logProvider, logInterval, retryQ, stateUpdater, logger)

	return []service.Recoverable{
		rcvProposal,
		rcvFinal,
		logTrigger,
	}
}
