package flows

import (
	"log"
	"time"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
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
	logger *log.Logger,
) []service.Recoverable {
	preprocessors := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{coord}

	// runs full check pipeline on a coordinated block with coordinated upkeeps
	conditionalFinal := newFinalConditionalFlow(preprocessors, resultStore, runner, FinalConditionalInterval, proposalQ, builder, retryQ, stateUpdater, logger)

	// the sampling proposal flow takes random samples of active upkeeps, checks
	// them and surfaces the ids if the items are eligible
	conditionalProposal := newSampleProposalFlow(preprocessors, ratio, getter, metadataStore, runner, SamplingConditionInterval, logger)

	return []service.Recoverable{conditionalFinal, conditionalProposal}
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
	recoveryProposalInterval time.Duration,
	recoveryFinalInterval time.Duration,
	retryQ ocr2keepers.RetryQueue,
	proposals ocr2keepers.ProposalQueue,
	stateUpdater ocr2keepers.UpkeepStateUpdater,
	logger *log.Logger,
) []service.Recoverable {
	// all flows use the same preprocessor based on the coordinator
	// each flow can add preprocessors to this provided slice
	preprocessors := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{coord}

	// the recovery proposal flow is for nodes to surface payloads that should
	// be recovered. these values are passed to the network and the network
	// votes on the proposed values
	rcvProposal := newRecoveryProposalFlow(preprocessors, runner, metadataStore, rp, recoveryProposalInterval, stateUpdater, logger)

	// the final recovery flow takes recoverable payloads merged with the latest
	// blocks and runs the pipeline for them. these values to run are derived
	// from node coordination and it can be assumed that all values should be
	// run.
	rcvFinal := newFinalRecoveryFlow(preprocessors, resultStore, runner, retryQ, recoveryFinalInterval, proposals, builder, stateUpdater, logger)

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
