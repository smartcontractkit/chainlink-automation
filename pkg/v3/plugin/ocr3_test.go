package plugin

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
	"github.com/stretchr/testify/assert"

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

}

type mockResultStore struct {
	ocr2keepers.ResultStore
	ViewFn func() ([]ocr2keepers.CheckResult, error)
}

func (s *mockResultStore) View() ([]ocr2keepers.CheckResult, error) {
	return s.ViewFn()
}
