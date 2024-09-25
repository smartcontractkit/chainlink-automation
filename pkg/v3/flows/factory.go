package flows

import (
	"log"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	conditionalflows "github.com/smartcontractkit/chainlink-automation/pkg/v4/workflows/conditional"
	logflows "github.com/smartcontractkit/chainlink-automation/pkg/v4/workflows/log"
	recoveryflows "github.com/smartcontractkit/chainlink-automation/pkg/v4/workflows/log/recovery"
	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

func NewConditionalTriggerFlows(
	coord ocr2keepersv3.PreProcessor,
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
	preprocessors := []ocr2keepersv3.PreProcessor{coord}

	// the sampling proposal flow takes random samples of active upkeeps, checks
	// them and surfaces the ids if the items are eligible
	conditionalProposal := conditionalflows.NewConditionalProposalFlow(preprocessors, ratio, getter, metadataStore, runner, logger)

	// runs full check pipeline on a coordinated block with coordinated upkeeps
	conditionalFinal := conditionalflows.NewConditionalFinalWorkflow(preprocessors, resultStore, runner, proposalQ, builder, retryQ, logger)

	return []service.Recoverable{conditionalProposal, conditionalFinal}
}

func NewLogTriggerFlows(
	coord ocr2keepersv3.PreProcessor,
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
	preprocessors := []ocr2keepersv3.PreProcessor{coord}

	// the log trigger flow is the happy path for log trigger payloads. all
	// retryables that are encountered in this flow are elevated to the retry
	// flow
	logTriggerFlow := logflows.NewLogTriggerFlow(preprocessors, logProvider, retryQ, resultStore, stateUpdater, runner, logger)

	// the recovery proposal flow is for nodes to surface payloads that should
	// be recovered. these values are passed to the network and the network
	// votes on the proposed values
	recoveryProposalFlow := recoveryflows.NewProposalRecoveryFlow(preprocessors, metadataStore, stateUpdater, runner, rp, logger)

	// the final recovery flow takes recoverable payloads merged with the latest
	// blocks and runs the pipeline for them. these values to run are derived
	// from node coordination and it can be assumed that all values should be
	// run.
	finalRecoveryFlow := recoveryflows.NewFinalRecoveryFlow(preprocessors, resultStore, stateUpdater, retryQ, proposals, builder, runner, logger)

	return []service.Recoverable{
		logTriggerFlow,
		recoveryProposalFlow,
		finalRecoveryFlow,
	}
}
