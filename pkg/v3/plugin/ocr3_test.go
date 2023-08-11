package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
	"github.com/stretchr/testify/assert"

	ocr2keepers2 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func TestOcr3Plugin_Query(t *testing.T) {
	plugin := &ocr3Plugin{}
	query, err := plugin.Query(context.Background(), ocr3types.OutcomeContext{})
	assert.Nil(t, query)
	assert.Nil(t, err)
}

func TestOcr3Plugin_Observation(t *testing.T) {
	t.Run("first round processing, previous outcome will be nil, creates an observation with 2 performables, 2 proposals and 2 block history", func(t *testing.T) {
		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "ocr3-test-observation", 0)

		metadataStore := &mockMetadataStore{
			GetBlockHistoryFn: func() ocr2keepers.BlockHistory {
				return ocr2keepers.BlockHistory{
					{
						Number: 1,
					},
					{
						Number: 2,
					},
				}
			},
			ViewConditionalProposalFn: func() []ocr2keepers.CoordinatedProposal {
				return []ocr2keepers.CoordinatedProposal{
					{
						WorkID: "workID1",
					},
				}
			},
			ViewLogRecoveryProposalFn: func() []ocr2keepers.CoordinatedProposal {
				return []ocr2keepers.CoordinatedProposal{
					{
						WorkID: "workID2",
					},
				}
			},
		}

		resultStore := &mockResultStore{
			ViewFn: func() ([]ocr2keepers.CheckResult, error) {
				return []ocr2keepers.CheckResult{
					{
						WorkID:   "workID1",
						Eligible: false,
					},
					{
						WorkID:   "workID2",
						Eligible: false,
					},
				}, nil
			},
		}

		coordinator := &mockCoordinator{
			FilterResultsFn: func(results []ocr2keepers.CheckResult) ([]ocr2keepers.CheckResult, error) {
				return results, nil
			},
			FilterProposalsFn: func(proposals []ocr2keepers.CoordinatedProposal) ([]ocr2keepers.CoordinatedProposal, error) {
				return proposals, nil
			},
		}

		plugin := &ocr3Plugin{
			AddBlockHistoryHook:         NewAddBlockHistoryHook(metadataStore),
			AddFromStagingHook:          NewAddFromStagingHook(resultStore, logger, coordinator),
			AddFromSamplesHook:          NewAddFromSamplesHook(metadataStore, coordinator),
			AddLogRecoveryProposalsHook: NewAddLogRecoveryProposalsHook(metadataStore, coordinator),
			Logger:                      logger,
		}

		outcomeCtx := ocr3types.OutcomeContext{
			PreviousOutcome: nil,
		}

		observation, err := plugin.Observation(context.Background(), outcomeCtx, types.Query{})
		assert.Nil(t, err)
		assert.Equal(t, types.Observation(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null},{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2"},{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`), observation)
		assert.True(t, strings.Contains(logBuf.String(), "built an observation in sequence nr 0 with 2 performables, 2 upkeep proposals and 2 block history"))
	})

	t.Run("first round processing, previous outcome will be nil, creates an observation with 3 performables, 2 upkeep proposals and 3 block history", func(t *testing.T) {
		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "ocr3-test-observation", 0)

		metadataStore := &mockMetadataStore{
			GetBlockHistoryFn: func() ocr2keepers.BlockHistory {
				return ocr2keepers.BlockHistory{
					{
						Number: 1,
					},
					{
						Number: 2,
					},
					{
						Number: 3,
					},
				}
			},
			ViewConditionalProposalFn: func() []ocr2keepers.CoordinatedProposal {
				return []ocr2keepers.CoordinatedProposal{
					{
						WorkID: "workID1",
					},
				}
			},
			ViewLogRecoveryProposalFn: func() []ocr2keepers.CoordinatedProposal {
				return []ocr2keepers.CoordinatedProposal{
					{
						WorkID: "workID2",
					},
				}
			},
		}

		resultStore := &mockResultStore{
			ViewFn: func() ([]ocr2keepers.CheckResult, error) {
				return []ocr2keepers.CheckResult{
					{
						WorkID:   "workID1",
						Eligible: false,
					},
					{
						WorkID:   "workID2",
						Eligible: false,
					},
					{
						WorkID:   "workID3",
						Eligible: false,
					},
				}, nil
			},
		}

		coordinator := &mockCoordinator{
			FilterResultsFn: func(results []ocr2keepers.CheckResult) ([]ocr2keepers.CheckResult, error) {
				return results, nil
			},
			FilterProposalsFn: func(proposals []ocr2keepers.CoordinatedProposal) ([]ocr2keepers.CoordinatedProposal, error) {
				return proposals, nil
			},
		}

		plugin := &ocr3Plugin{
			AddBlockHistoryHook:         NewAddBlockHistoryHook(metadataStore),
			AddFromStagingHook:          NewAddFromStagingHook(resultStore, logger, coordinator),
			AddFromSamplesHook:          NewAddFromSamplesHook(metadataStore, coordinator),
			AddLogRecoveryProposalsHook: NewAddLogRecoveryProposalsHook(metadataStore, coordinator),
			Logger:                      logger,
		}

		outcomeCtx := ocr3types.OutcomeContext{
			PreviousOutcome: nil,
		}

		observation, err := plugin.Observation(context.Background(), outcomeCtx, types.Query{})
		assert.Nil(t, err)
		assert.Equal(t, types.Observation(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null},{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID3","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null},{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2"},{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":3,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`), observation)
		assert.True(t, strings.Contains(logBuf.String(), "built an observation in sequence nr 0 with 3 performables, 2 upkeep proposals and 3 block history"))
	})

	t.Run("first round processing, previous outcome will be nil, creates an observation with 3 performables, 0 upkeep proposals and 3 block history", func(t *testing.T) {
		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "ocr3-test-observation", 0)

		metadataStore := &mockMetadataStore{
			GetBlockHistoryFn: func() ocr2keepers.BlockHistory {
				return ocr2keepers.BlockHistory{
					{
						Number: 1,
					},
					{
						Number: 2,
					},
					{
						Number: 3,
					},
				}
			},
			ViewConditionalProposalFn: func() []ocr2keepers.CoordinatedProposal {
				return []ocr2keepers.CoordinatedProposal{}
			},
			ViewLogRecoveryProposalFn: func() []ocr2keepers.CoordinatedProposal {
				return []ocr2keepers.CoordinatedProposal{}
			},
		}

		resultStore := &mockResultStore{
			ViewFn: func() ([]ocr2keepers.CheckResult, error) {
				return []ocr2keepers.CheckResult{
					{
						WorkID:              "workID1",
						IneligibilityReason: 1,
						Eligible:            false,
					},
					{
						WorkID:   "workID2",
						Eligible: false,
					},
					{
						WorkID:   "workID3",
						Eligible: false,
					},
				}, nil
			},
		}

		coordinator := &mockCoordinator{
			FilterResultsFn: func(results []ocr2keepers.CheckResult) ([]ocr2keepers.CheckResult, error) {
				return results, nil
			},
			FilterProposalsFn: func(proposals []ocr2keepers.CoordinatedProposal) ([]ocr2keepers.CoordinatedProposal, error) {
				return proposals, nil
			},
		}

		plugin := &ocr3Plugin{
			AddBlockHistoryHook:         NewAddBlockHistoryHook(metadataStore),
			AddFromStagingHook:          NewAddFromStagingHook(resultStore, logger, coordinator),
			AddFromSamplesHook:          NewAddFromSamplesHook(metadataStore, coordinator),
			AddLogRecoveryProposalsHook: NewAddLogRecoveryProposalsHook(metadataStore, coordinator),
			Logger:                      logger,
		}

		outcomeCtx := ocr3types.OutcomeContext{
			PreviousOutcome: nil,
		}

		observation, err := plugin.Observation(context.Background(), outcomeCtx, types.Query{})
		assert.Nil(t, err)
		assert.Equal(t, types.Observation(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":1,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null},{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID3","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null},{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":null,"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":3,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`), observation)
		assert.True(t, strings.Contains(logBuf.String(), "built an observation in sequence nr 0 with 3 performables, 0 upkeep proposals and 3 block history"))
	})

	t.Run("subsequent round processing, previous outcome will be non nil, creates an observation built an observation with 2 performables, 0 upkeep proposals and 3 block history", func(t *testing.T) {
		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "ocr3-test-observation", 0)

		metadataStore := &mockMetadataStore{
			GetBlockHistoryFn: func() ocr2keepers.BlockHistory {
				return ocr2keepers.BlockHistory{
					{
						Number: 3,
					},
					{
						Number: 4,
					},
					{
						Number: 5,
					},
				}
			},
			ViewConditionalProposalFn: func() []ocr2keepers.CoordinatedProposal {
				return []ocr2keepers.CoordinatedProposal{}
			},
			ViewLogRecoveryProposalFn: func() []ocr2keepers.CoordinatedProposal {
				return []ocr2keepers.CoordinatedProposal{}
			},
		}

		resultStore := &mockResultStore{
			ViewFn: func() ([]ocr2keepers.CheckResult, error) {
				return []ocr2keepers.CheckResult{
					{
						WorkID:              "workID5",
						IneligibilityReason: 1,
						Eligible:            false,
					},
					{
						WorkID:   "workID6",
						Eligible: false,
					},
				}, nil
			},
		}

		coordinator := &mockCoordinator{
			FilterResultsFn: func(results []ocr2keepers.CheckResult) ([]ocr2keepers.CheckResult, error) {
				return results, nil
			},
			FilterProposalsFn: func(proposals []ocr2keepers.CoordinatedProposal) ([]ocr2keepers.CoordinatedProposal, error) {
				return proposals, nil
			},
		}

		queueItems := []ocr2keepers.CoordinatedProposal{}

		proposalQueue := &mockProposalQueue{
			EnqueueFn: func(items ...ocr2keepers.CoordinatedProposal) error {
				queueItems = append(queueItems, items...)
				return nil
			},
		}

		remover := &mockRemover{}

		plugin := &ocr3Plugin{
			RemoveFromStagingHook:       NewRemoveFromStaging(remover, logger),
			RemoveFromMetadataHook:      NewRemoveFromMetadataHook(remover, logger),
			AddToProposalQHook:          NewAddToProposalQHook(proposalQueue, logger),
			AddBlockHistoryHook:         NewAddBlockHistoryHook(metadataStore),
			AddFromStagingHook:          NewAddFromStagingHook(resultStore, logger, coordinator),
			AddFromSamplesHook:          NewAddFromSamplesHook(metadataStore, coordinator),
			AddLogRecoveryProposalsHook: NewAddLogRecoveryProposalsHook(metadataStore, coordinator),
			Logger:                      logger,
		}

		previousOutcome := ocr2keepers2.AutomationOutcome{
			AgreedPerformables: []ocr2keepers.CheckResult{
				{
					WorkID: "workID",
				},
			},
			AgreedProposals: [][]ocr2keepers.CoordinatedProposal{
				{
					{
						WorkID: "workID",
					},
				},
			},
		}
		previousOutcomeBytes, err := json.Marshal(previousOutcome)
		assert.NoError(t, err)
		outcomeCtx := ocr3types.OutcomeContext{
			PreviousOutcome: ocr3types.Outcome(previousOutcomeBytes),
		}

		observation, err := plugin.Observation(context.Background(), outcomeCtx, types.Query{})
		assert.NoError(t, err)
		assert.Equal(t, types.Observation(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID6","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null},{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":1,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID5","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":null,"BlockHistory":[{"Number":3,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":4,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":5,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`), observation)
		assert.True(t, strings.Contains(logBuf.String(), "built an observation in sequence nr 0 with 2 performables, 0 upkeep proposals and 3 block history"))
	})

	t.Run("subsequent round processing, previous outcome will be non nil, filters results, creates an observation built an observation with 1 performables, 0 upkeep proposals and 3 block history", func(t *testing.T) {
		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "ocr3-test-observation", 0)

		metadataStore := &mockMetadataStore{
			GetBlockHistoryFn: func() ocr2keepers.BlockHistory {
				return ocr2keepers.BlockHistory{
					{
						Number: 3,
					},
					{
						Number: 4,
					},
					{
						Number: 5,
					},
				}
			},
			ViewConditionalProposalFn: func() []ocr2keepers.CoordinatedProposal {
				return []ocr2keepers.CoordinatedProposal{}
			},
			ViewLogRecoveryProposalFn: func() []ocr2keepers.CoordinatedProposal {
				return []ocr2keepers.CoordinatedProposal{}
			},
		}

		resultStore := &mockResultStore{
			ViewFn: func() ([]ocr2keepers.CheckResult, error) {
				return []ocr2keepers.CheckResult{
					{
						WorkID:              "workID5",
						IneligibilityReason: 1,
						Eligible:            false,
					},
					{
						WorkID:   "workID6",
						Eligible: false,
					},
				}, nil
			},
		}

		coordinator := &mockCoordinator{
			FilterResultsFn: func(results []ocr2keepers.CheckResult) ([]ocr2keepers.CheckResult, error) {
				return results[:1], nil
			},
			FilterProposalsFn: func(proposals []ocr2keepers.CoordinatedProposal) ([]ocr2keepers.CoordinatedProposal, error) {
				return proposals, nil
			},
		}

		queueItems := []ocr2keepers.CoordinatedProposal{}

		proposalQueue := &mockProposalQueue{
			EnqueueFn: func(items ...ocr2keepers.CoordinatedProposal) error {
				queueItems = append(queueItems, items...)
				return nil
			},
		}

		remover := &mockRemover{}

		plugin := &ocr3Plugin{
			RemoveFromStagingHook:       NewRemoveFromStaging(remover, logger),
			RemoveFromMetadataHook:      NewRemoveFromMetadataHook(remover, logger),
			AddToProposalQHook:          NewAddToProposalQHook(proposalQueue, logger),
			AddBlockHistoryHook:         NewAddBlockHistoryHook(metadataStore),
			AddFromStagingHook:          NewAddFromStagingHook(resultStore, logger, coordinator),
			AddFromSamplesHook:          NewAddFromSamplesHook(metadataStore, coordinator),
			AddLogRecoveryProposalsHook: NewAddLogRecoveryProposalsHook(metadataStore, coordinator),
			Logger:                      logger,
		}

		previousOutcome := ocr2keepers2.AutomationOutcome{
			AgreedPerformables: []ocr2keepers.CheckResult{
				{
					WorkID: "workID",
				},
			},
			AgreedProposals: [][]ocr2keepers.CoordinatedProposal{
				{
					{
						WorkID: "workID",
					},
				},
			},
		}
		previousOutcomeBytes, err := json.Marshal(previousOutcome)
		assert.NoError(t, err)
		outcomeCtx := ocr3types.OutcomeContext{
			PreviousOutcome: ocr3types.Outcome(previousOutcomeBytes),
		}

		observation, err := plugin.Observation(context.Background(), outcomeCtx, types.Query{})
		assert.NoError(t, err)
		assert.Equal(t, types.Observation(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":1,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID5","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":null,"BlockHistory":[{"Number":3,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":4,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":5,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`), observation)
		assert.True(t, strings.Contains(logBuf.String(), "built an observation in sequence nr 0 with 1 performables, 0 upkeep proposals and 3 block history"))
	})

	t.Run("subsequent round processing, decoding an invalid previous outcome returns an error", func(t *testing.T) {
		plugin := &ocr3Plugin{}

		outcomeCtx := ocr3types.OutcomeContext{
			PreviousOutcome: ocr3types.Outcome(`invalid`),
		}

		observation, err := plugin.Observation(context.Background(), outcomeCtx, types.Query{})
		assert.Error(t, err)
		assert.Nil(t, observation)
	})
}

func TestOcr3Plugin_ValidateObservation(t *testing.T) {
	for _, tc := range []struct {
		name        string
		observation types.AttributedObservation
		expectsErr  bool
		wantErr     error
	}{
		{
			name:        "validating an empty observation returns an error",
			observation: types.AttributedObservation{},
			expectsErr:  true,
			wantErr:     errors.New("unexpected end of JSON input"),
		},
		{
			name: "successfully validates a well formed observation",
			observation: types.AttributedObservation{
				Observation: []byte(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":1,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`),
			},
		},
		{
			name: "gas allocated cannot be zero",
			observation: types.AttributedObservation{
				Observation: []byte(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`),
			},
			expectsErr: true,
			wantErr:    errors.New("gas allocated cannot be zero"),
		},
		{
			name: "check result cannot be ineligible and have no ineligibility reason",
			observation: types.AttributedObservation{
				Observation: []byte(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`),
			},
			expectsErr: true,
			wantErr:    errors.New("check result cannot be ineligible and have no ineligibility reason"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plugin := &ocr3Plugin{}
			err := plugin.ValidateObservation(ocr3types.OutcomeContext{}, nil, tc.observation)
			if tc.expectsErr {
				assert.Error(t, err)
				assert.Equal(t, err.Error(), tc.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOcr3Plugin_Outcome(t *testing.T) {
	for _, tc := range []struct {
		name         string
		observations []types.AttributedObservation
		prevOutcome  ocr3types.Outcome
		wantOutcome  ocr3types.Outcome
		expectsErr   bool
		wantErr      error
	}{
		{
			name:         "processing an empty list of observations generates an empty outcome",
			observations: []types.AttributedObservation{},
			wantOutcome:  ocr3types.Outcome([]byte(`{"AgreedPerformables":null,"AgreedProposals":[]}`)),
		},
		{
			name: "processing a well formed observation with a previous outcome generates an new outcome",
			observations: []types.AttributedObservation{
				{
					Observation: []byte(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":1,"PerformData":null,"FastGasWei":0,"LinkNative":0}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`),
				},
			},
			prevOutcome: ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"WorkID":"workID1"}],"AgreedProposals":[[{"WorkID":"workID1"}]]}`)),
			wantOutcome: ocr3types.Outcome([]byte(`{"AgreedPerformables":null,"AgreedProposals":[[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}]]}`)),
		},
		{
			name: "processing a malformed observation with a previous outcome generates an new outcome",
			observations: []types.AttributedObservation{
				{
					Observation: []byte(`invalid`),
				},
			},
			prevOutcome: ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"WorkID":"workID1"}],"AgreedProposals":[[{"WorkID":"workID1"}]]}`)),
			wantOutcome: ocr3types.Outcome([]byte(`{"AgreedPerformables":null,"AgreedProposals":[[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}]]}`)),
		},
		{
			name: "processing an invalid observation with a previous outcome generates an new outcome",
			observations: []types.AttributedObservation{
				{
					Observation: []byte(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":0,"PerformData":null,"FastGasWei":0,"LinkNative":0}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`),
				},
			},
			prevOutcome: ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"WorkID":"workID1"}],"AgreedProposals":[[{"WorkID":"workID1"}]]}`)),
			wantOutcome: ocr3types.Outcome([]byte(`{"AgreedPerformables":null,"AgreedProposals":[[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}]]}`)),
		},
		{
			name: "processing an valid observation with a malformed previous outcome returns an error",
			observations: []types.AttributedObservation{
				{
					Observation: []byte(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":1,"PerformData":null,"FastGasWei":0,"LinkNative":0}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`),
				},
			},
			prevOutcome: ocr3types.Outcome([]byte(`invalid`)),
			expectsErr:  true,
			wantErr:     errors.New("invalid character 'i' looking for beginning of value"),
		},
		// TODO when ValidateAutomationOutcome is implemented
		//{
		//	name: "processing an valid observation with an invalid previous outcome returns an error",
		//	observations: []types.AttributedObservation{
		//		{
		//			Observation: []byte(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":1,"PerformData":null,"FastGasWei":0,"LinkNative":0}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`),
		//		},
		//	},
		//	outcome: ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"WorkID":"workID1"}],"AgreedProposals":[[{"WorkID":"workID1","GasAllocated":0}]]}`)),
		//	expectsErr:  true,
		//	wantErr:     errors.New(""),
		//},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			logger := log.New(&logBuf, "ocr3-test-outcome", 0)

			plugin := &ocr3Plugin{
				Logger: logger,
			}
			outcome, err := plugin.Outcome(ocr3types.OutcomeContext{
				PreviousOutcome: tc.prevOutcome,
			}, nil, tc.observations)
			if tc.expectsErr {
				assert.Error(t, err)
				assert.Equal(t, err.Error(), tc.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantOutcome, outcome)
			}
		})
	}
}

func TestOcr3Plugin_Reports(t *testing.T) {
	for _, tc := range []struct {
		name                string
		sequenceNumber      uint64
		outcome             ocr3types.Outcome
		wantReportsWithInfo []ocr3types.ReportWithInfo[AutomationReportInfo]
		encoder             ocr2keepers.Encoder
		expectsErr          bool
		wantErr             error
	}{
		{
			name:           "an empty outcome returns an error",
			sequenceNumber: 5,
			outcome:        ocr3types.Outcome([]byte{}),
			expectsErr:     true,
			wantErr:        errors.New("unexpected end of JSON input"),
		},
		{
			name:                "an empty json object generates a nil report",
			sequenceNumber:      5,
			outcome:             ocr3types.Outcome([]byte(`{}`)),
			wantReportsWithInfo: []ocr3types.ReportWithInfo[AutomationReportInfo](nil),
		},
		{
			name:           "a well formed outcome gets encoded as a report",
			sequenceNumber: 5,
			outcome:        ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}],"AgreedProposals":[[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}]]}`)),
			encoder: &mockEncoder{
				EncodeFn: func(result ...ocr2keepers.CheckResult) ([]byte, error) {
					return json.Marshal(result)
				},
			},
			wantReportsWithInfo: []ocr3types.ReportWithInfo[AutomationReportInfo]{
				{
					Report: []byte(`[]`),
				},
				{
					Report: []byte(`[{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}]`),
				},
			},
		},
		{
			name:           "an error is returned when the encoder errors",
			sequenceNumber: 5,
			outcome:        ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}],"AgreedProposals":[[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}]]}`)),
			encoder: &mockEncoder{
				EncodeFn: func(result ...ocr2keepers.CheckResult) ([]byte, error) {
					return nil, errors.New("encode boom")
				},
			},
			expectsErr: true,
			wantErr:    errors.New("error encountered while encoding: encode boom"),
		},
		{
			name:           "an error is returned when the encoder errors",
			sequenceNumber: 5,
			outcome:        ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}],"AgreedProposals":[[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}]]}`)),
			encoder: &mockEncoder{
				EncodeFn: func(result ...ocr2keepers.CheckResult) ([]byte, error) {
					return nil, errors.New("encode boom")
				},
			},
			expectsErr: true,
			wantErr:    errors.New("error encountered while encoding: encode boom"),
		},
		{
			name:           "an error is returned when the encoder errors when there are stillv values to add",
			sequenceNumber: 5,
			outcome:        ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}],"AgreedProposals":[[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}]]}`)),
			encoder: &mockEncoder{
				EncodeFn: func(result ...ocr2keepers.CheckResult) ([]byte, error) {
					if len(result) == 0 { // the first call to encode with this test passes 0 check results, so we want to error on the second call, which gets non-zero results
						return json.Marshal(result)
					}
					return nil, errors.New("encode boom")
				},
			},
			expectsErr: true,
			wantErr:    errors.New("error encountered while encoding: encode boom"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			logger := log.New(&logBuf, "ocr3-test-reports", 0)

			plugin := &ocr3Plugin{
				Logger:        logger,
				ReportEncoder: tc.encoder,
			}
			reportsWithInfo, err := plugin.Reports(tc.sequenceNumber, tc.outcome)
			if tc.expectsErr {
				assert.Error(t, err)
				assert.Equal(t, err.Error(), tc.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantReportsWithInfo, reportsWithInfo)
			}
		})
	}
}

func TestOcr3Plugin_ShouldAcceptAttestedReport(t *testing.T) {
	for _, tc := range []struct {
		name           string
		sequenceNumber uint64
		reportWithInfo ocr3types.ReportWithInfo[AutomationReportInfo]
		encoder        ocr2keepers.Encoder
		coordinator    ocr2keepers.Coordinator
		expectsErr     bool
		wantErr        error
		wantOK         bool
	}{
		{
			name:           "when at least one upkeep should be accepted, we accept",
			sequenceNumber: 5,
			encoder: &mockEncoder{
				ExtractFn: func(i []byte) ([]ocr2keepers.ReportedUpkeep, error) {
					return []ocr2keepers.ReportedUpkeep{
						{
							WorkID: "workID1",
						},
						{
							WorkID: "workID2",
						},
						{
							WorkID: "workID3",
						},
					}, nil
				},
			},
			coordinator: &mockCoordinator{
				ShouldAcceptFn: func(upkeep ocr2keepers.ReportedUpkeep) bool {
					if upkeep.WorkID == "workID3" {
						return true
					}
					return false
				},
			},
			wantOK: true,
		},
		{
			name:           "when all upkeeps shouldn't be accepted, we don't accept",
			sequenceNumber: 5,
			encoder: &mockEncoder{
				ExtractFn: func(i []byte) ([]ocr2keepers.ReportedUpkeep, error) {
					return []ocr2keepers.ReportedUpkeep{
						{
							WorkID: "workID1",
						},
						{
							WorkID: "workID2",
						},
						{
							WorkID: "workID3",
						},
					}, nil
				},
			},
			coordinator: &mockCoordinator{
				ShouldAcceptFn: func(upkeep ocr2keepers.ReportedUpkeep) bool {
					return false
				},
			},
			wantOK: false,
		},
		{
			name:           "when extraction errors, an error is returned an we shouldn't accept",
			sequenceNumber: 5,
			encoder: &mockEncoder{
				ExtractFn: func(i []byte) ([]ocr2keepers.ReportedUpkeep, error) {
					return nil, errors.New("extract boom")
				},
			},
			wantOK:     false,
			expectsErr: true,
			wantErr:    errors.New("extract boom"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			logger := log.New(&logBuf, "ocr3-test-shouldAcceptAttestedReport", 0)

			plugin := &ocr3Plugin{
				Logger:        logger,
				ReportEncoder: tc.encoder,
				Coordinator:   tc.coordinator,
			}
			ok, err := plugin.ShouldAcceptAttestedReport(context.Background(), tc.sequenceNumber, tc.reportWithInfo)
			if tc.expectsErr {
				assert.Error(t, err)
				assert.Equal(t, err.Error(), tc.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantOK, ok)
			}
		})
	}
}

func TestOcr3Plugin_ShouldTransmitAcceptedReport(t *testing.T) {
	for _, tc := range []struct {
		name           string
		sequenceNumber uint64
		reportWithInfo ocr3types.ReportWithInfo[AutomationReportInfo]
		encoder        ocr2keepers.Encoder
		coordinator    ocr2keepers.Coordinator
		expectsErr     bool
		wantErr        error
		wantOK         bool
	}{
		{
			name:           "when at least one upkeep should be transmitted, we transmit",
			sequenceNumber: 5,
			encoder: &mockEncoder{
				ExtractFn: func(i []byte) ([]ocr2keepers.ReportedUpkeep, error) {
					return []ocr2keepers.ReportedUpkeep{
						{
							WorkID: "workID1",
						},
						{
							WorkID: "workID2",
						},
						{
							WorkID: "workID3",
						},
					}, nil
				},
			},
			coordinator: &mockCoordinator{
				ShouldTransmitFn: func(upkeep ocr2keepers.ReportedUpkeep) bool {
					if upkeep.WorkID == "workID3" {
						return true
					}
					return false
				},
			},
			wantOK: true,
		},
		{
			name:           "when all upkeeps shouldn't be transmitted, we don't transmit",
			sequenceNumber: 5,
			encoder: &mockEncoder{
				ExtractFn: func(i []byte) ([]ocr2keepers.ReportedUpkeep, error) {
					return []ocr2keepers.ReportedUpkeep{
						{
							WorkID: "workID1",
						},
						{
							WorkID: "workID2",
						},
						{
							WorkID: "workID3",
						},
					}, nil
				},
			},
			coordinator: &mockCoordinator{
				ShouldTransmitFn: func(upkeep ocr2keepers.ReportedUpkeep) bool {
					return false
				},
			},
			wantOK: false,
		},
		{
			name:           "when extraction errors, an error is returned an we shouldn't transmit",
			sequenceNumber: 5,
			encoder: &mockEncoder{
				ExtractFn: func(i []byte) ([]ocr2keepers.ReportedUpkeep, error) {
					return nil, errors.New("extract boom")
				},
			},
			wantOK:     false,
			expectsErr: true,
			wantErr:    errors.New("extract boom"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			logger := log.New(&logBuf, "ocr3-test-shouldAcceptAttestedReport", 0)

			plugin := &ocr3Plugin{
				Logger:        logger,
				ReportEncoder: tc.encoder,
				Coordinator:   tc.coordinator,
			}
			ok, err := plugin.ShouldTransmitAcceptedReport(context.Background(), tc.sequenceNumber, tc.reportWithInfo)
			if tc.expectsErr {
				assert.Error(t, err)
				assert.Equal(t, err.Error(), tc.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantOK, ok)
			}
		})
	}
}

func TestOcr3Plugin_startServices(t *testing.T) {
	var logBuf bytes.Buffer
	logger := log.New(&logBuf, "ocr3-test-shouldAcceptAttestedReport", 0)

	startedCh := make(chan struct{}, 1)
	plugin := &ocr3Plugin{
		Logger: logger,
		Services: []service.Recoverable{
			&mockRecoverable{
				StartFn: func(ctx context.Context) error {
					return errors.New("this won't prevent other services from starting")
				},
				CloseFn: func() error {
					return nil
				},
			},
			&mockRecoverable{
				StartFn: func(ctx context.Context) error {
					startedCh <- struct{}{}
					return nil
				},
				CloseFn: func() error {
					return errors.New("a service failed to close")
				},
			},
		},
	}
	plugin.startServices()

	<-startedCh

	err := plugin.Close()

	assert.Error(t, err)
	assert.Equal(t, err.Error(), "a service failed to close")
}

type mockResultStore struct {
	ocr2keepers.ResultStore
	ViewFn func() ([]ocr2keepers.CheckResult, error)
}

func (s *mockResultStore) View() ([]ocr2keepers.CheckResult, error) {
	return s.ViewFn()
}

type mockProposalQueue struct {
	EnqueueFn func(items ...ocr2keepers.CoordinatedProposal) error
	DequeueFn func(t ocr2keepers.UpkeepType, n int) ([]ocr2keepers.CoordinatedProposal, error)
}

func (s *mockProposalQueue) Enqueue(items ...ocr2keepers.CoordinatedProposal) error {
	return s.EnqueueFn(items...)
}

func (s *mockProposalQueue) Dequeue(t ocr2keepers.UpkeepType, n int) ([]ocr2keepers.CoordinatedProposal, error) {
	return s.DequeueFn(t, n)
}

type mockEncoder struct {
	EncodeFn  func(...ocr2keepers.CheckResult) ([]byte, error)
	ExtractFn func([]byte) ([]ocr2keepers.ReportedUpkeep, error)
}

func (e *mockEncoder) Encode(res ...ocr2keepers.CheckResult) ([]byte, error) {
	return e.EncodeFn(res...)
}

func (e *mockEncoder) Extract(b []byte) ([]ocr2keepers.ReportedUpkeep, error) {
	return e.ExtractFn(b)
}

type mockRecoverable struct {
	StartFn func(context.Context) error
	CloseFn func() error
}

func (e *mockRecoverable) Start(ctx context.Context) error {
	return e.StartFn(ctx)
}

func (e *mockRecoverable) Close() error {
	return e.CloseFn()
}
