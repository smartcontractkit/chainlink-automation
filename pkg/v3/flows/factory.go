package flows

import (
	"log"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	conditionalflows "github.com/smartcontractkit/chainlink-automation/pkg/v3/flows/conditional"
	logflows "github.com/smartcontractkit/chainlink-automation/pkg/v3/flows/log"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

func NewConditionalTriggerFlows(
	coord ocr2keepersv3.PreProcessor[common.UpkeepPayload],
	ratio types.Ratio,
	getter common.ConditionalUpkeepProvider,
	subscriber common.BlockSubscriber,
	builder common.PayloadBuilder,
	resultStore types.ResultStore,
	metadataStore types.MetadataStore,
	runner ocr2keepersv3.Runner,
	proposalQ types.ProposalQueue,
	retryQ types.RetryQueue,
	stateUpdater common.UpkeepStateUpdater,
	logger *log.Logger,
) []service.Recoverable {
	preprocessors := []ocr2keepersv3.PreProcessor[common.UpkeepPayload]{coord}

	// the sampling proposal flow takes random samples of active upkeeps, checks
	// them and surfaces the ids if the items are eligible
	conditionalProposal := conditionalflows.NewSampleProposalFlow(preprocessors, ratio, getter, metadataStore, runner, logger)

	// runs full check pipeline on a coordinated block with coordinated upkeeps
	conditionalFinal := conditionalflows.NewFinalConditionalFlow(preprocessors, resultStore, runner, proposalQ, builder, retryQ, stateUpdater, logger)

	return []service.Recoverable{conditionalProposal, conditionalFinal}
}

func NewLogTriggerFlows(
	coord ocr2keepersv3.PreProcessor[common.UpkeepPayload],
	resultStore types.ResultStore,
	metadataStore types.MetadataStore,
	runner ocr2keepersv3.Runner,
	logProvider common.LogEventProvider,
	rp common.RecoverableProvider,
	builder common.PayloadBuilder,
	retryQ types.RetryQueue,
	proposals types.ProposalQueue,
	stateUpdater common.UpkeepStateUpdater,
	logger *log.Logger,
) []service.Recoverable {
	// all flows use the same preprocessor based on the coordinator
	// each flow can add preprocessors to this provided slice
	preprocessors := []ocr2keepersv3.PreProcessor[common.UpkeepPayload]{coord}

	// the log trigger flow is the happy path for log trigger payloads. all
	// retryables that are encountered in this flow are elevated to the retry
	// flow
	logTriggerFlow := logflows.NewLogTriggerFlow(preprocessors, resultStore, runner, logProvider, retryQ, stateUpdater, logger)

	// the recovery proposal flow is for nodes to surface payloads that should
	// be recovered. these values are passed to the network and the network
	// votes on the proposed values
	recoveryProposalFlow := logflows.NewRecoveryProposalFlow(preprocessors, runner, metadataStore, rp, stateUpdater, logger)

	// the final recovery flow takes recoverable payloads merged with the latest
	// blocks and runs the pipeline for them. these values to run are derived
	// from node coordination and it can be assumed that all values should be
	// run.
	finalRecoveryFlow := logflows.NewFinalRecoveryFlow(preprocessors, resultStore, runner, retryQ, proposals, builder, stateUpdater, logger)

	return []service.Recoverable{
		logTriggerFlow,
		recoveryProposalFlow,
		finalRecoveryFlow,
	}
}
