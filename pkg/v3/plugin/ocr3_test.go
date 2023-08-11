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
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func TestOcr3Plugin_Query(t *testing.T) {
	plugin := &ocr3Plugin{}
	query, err := plugin.Query(context.Background(), ocr3types.OutcomeContext{})
	assert.Nil(t, query)
	assert.Nil(t, err)
}

func TestOcr3Plugin_Outcome(t *testing.T) {
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
		name       string
		ao         types.AttributedObservation
		expectsErr bool
		wantErr    error
	}{
		{
			name:       "validating an empty observation returns an error",
			ao:         types.AttributedObservation{},
			expectsErr: true,
			wantErr:    errors.New("unexpected end of JSON input"),
		},
		{
			name: "successfully validates a well formed observation",
			ao: types.AttributedObservation{
				Observation: []byte(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":1,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`),
			},
		},
		{
			name: "gas allocated cannot be zero",
			ao: types.AttributedObservation{
				Observation: []byte(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`),
			},
			expectsErr: true,
			wantErr:    errors.New("gas allocated cannot be zero"),
		},
		{
			name: "check result cannot be ineligible and have no ineligibility reason",
			ao: types.AttributedObservation{
				Observation: []byte(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`),
			},
			expectsErr: true,
			wantErr:    errors.New("check result cannot be ineligible and have no ineligibility reason"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plugin := &ocr3Plugin{}
			err := plugin.ValidateObservation(ocr3types.OutcomeContext{}, nil, tc.ao)
			if tc.expectsErr {
				assert.Error(t, err)
				assert.Equal(t, err.Error(), tc.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
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
