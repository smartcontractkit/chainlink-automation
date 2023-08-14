package plugin

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
	"github.com/stretchr/testify/assert"

	ocr2keepers2 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/plugin/hooks"
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
			ViewProposalsFn: func(upkeepType ocr2keepers.UpkeepType) []ocr2keepers.CoordinatedBlockProposal {
				return []ocr2keepers.CoordinatedBlockProposal{
					{
						WorkID: "workID1",
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
			FilterProposalsFn: func(proposals []ocr2keepers.CoordinatedBlockProposal) ([]ocr2keepers.CoordinatedBlockProposal, error) {
				return proposals, nil
			},
		}

		plugin := &ocr3Plugin{
			UpkeepTypeGetter:            mockUpkeepTypeGetter,
			WorkIDGenerator:             mockWorkIDGenerator,
			AddBlockHistoryHook:         hooks.NewAddBlockHistoryHook(metadataStore, logger),
			AddFromStagingHook:          hooks.NewAddFromStagingHook(resultStore, coordinator, logger),
			AddConditionalProposalsHook: hooks.NewAddConditionalProposalsHook(metadataStore, coordinator, logger),
			AddLogProposalsHook:         hooks.NewAddLogProposalsHook(metadataStore, coordinator, logger),
			Logger:                      logger,
		}

		outcomeCtx := ocr3types.OutcomeContext{
			PreviousOutcome: nil,
		}

		observation, err := plugin.Observation(context.Background(), outcomeCtx, types.Query{})
		assert.Nil(t, err)
		assert.Equal(t, types.Observation(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null},{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"},{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`), observation)
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
			ViewProposalsFn: func(upkeepType ocr2keepers.UpkeepType) []ocr2keepers.CoordinatedBlockProposal {
				return []ocr2keepers.CoordinatedBlockProposal{
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
			FilterProposalsFn: func(proposals []ocr2keepers.CoordinatedBlockProposal) ([]ocr2keepers.CoordinatedBlockProposal, error) {
				return proposals, nil
			},
		}

		plugin := &ocr3Plugin{
			UpkeepTypeGetter:            mockUpkeepTypeGetter,
			WorkIDGenerator:             mockWorkIDGenerator,
			AddBlockHistoryHook:         hooks.NewAddBlockHistoryHook(metadataStore, logger),
			AddFromStagingHook:          hooks.NewAddFromStagingHook(resultStore, coordinator, logger),
			AddConditionalProposalsHook: hooks.NewAddConditionalProposalsHook(metadataStore, coordinator, logger),
			AddLogProposalsHook:         hooks.NewAddLogProposalsHook(metadataStore, coordinator, logger),
			Logger:                      logger,
		}

		outcomeCtx := ocr3types.OutcomeContext{
			PreviousOutcome: nil,
		}

		observation, err := plugin.Observation(context.Background(), outcomeCtx, types.Query{})
		assert.Nil(t, err)
		assert.Equal(t, types.Observation(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null},{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null},{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID3","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2"},{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":3,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`), observation)
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
			ViewProposalsFn: func(upkeepType ocr2keepers.UpkeepType) []ocr2keepers.CoordinatedBlockProposal {
				return []ocr2keepers.CoordinatedBlockProposal{}
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
			FilterProposalsFn: func(proposals []ocr2keepers.CoordinatedBlockProposal) ([]ocr2keepers.CoordinatedBlockProposal, error) {
				return proposals, nil
			},
		}

		plugin := &ocr3Plugin{
			UpkeepTypeGetter:            mockUpkeepTypeGetter,
			WorkIDGenerator:             mockWorkIDGenerator,
			AddBlockHistoryHook:         hooks.NewAddBlockHistoryHook(metadataStore, logger),
			AddFromStagingHook:          hooks.NewAddFromStagingHook(resultStore, coordinator, logger),
			AddConditionalProposalsHook: hooks.NewAddConditionalProposalsHook(metadataStore, coordinator, logger),
			AddLogProposalsHook:         hooks.NewAddLogProposalsHook(metadataStore, coordinator, logger),
			Logger:                      logger,
		}

		outcomeCtx := ocr3types.OutcomeContext{
			PreviousOutcome: nil,
		}

		observation, err := plugin.Observation(context.Background(), outcomeCtx, types.Query{})
		assert.Nil(t, err)
		assert.Equal(t, types.Observation(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":1,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null},{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null},{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID3","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":null,"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":3,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`), observation)
		assert.True(t, strings.Contains(logBuf.String(), "built an observation in sequence nr 0 with 3 performables, 0 upkeep proposals and 3 block history"))
	})

	t.Run("ineligible check result returns an error", func(t *testing.T) {
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
			ViewProposalsFn: func(upkeepType ocr2keepers.UpkeepType) []ocr2keepers.CoordinatedBlockProposal {
				return []ocr2keepers.CoordinatedBlockProposal{}
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
			RemoveFn: func(s ...string) {
				// no op
			},
		}

		coordinator := &mockCoordinator{
			FilterResultsFn: func(results []ocr2keepers.CheckResult) ([]ocr2keepers.CheckResult, error) {
				return results, nil
			},
			FilterProposalsFn: func(proposals []ocr2keepers.CoordinatedBlockProposal) ([]ocr2keepers.CoordinatedBlockProposal, error) {
				return proposals, nil
			},
		}

		queueItems := []ocr2keepers.CoordinatedBlockProposal{}

		proposalQueue := &mockProposalQueue{
			EnqueueFn: func(items ...ocr2keepers.CoordinatedBlockProposal) error {
				queueItems = append(queueItems, items...)
				return nil
			},
		}

		plugin := &ocr3Plugin{
			UpkeepTypeGetter:            mockUpkeepTypeGetter,
			WorkIDGenerator:             mockWorkIDGenerator,
			RemoveFromStagingHook:       hooks.NewRemoveFromStagingHook(resultStore, logger),
			RemoveFromMetadataHook:      hooks.NewRemoveFromMetadataHook(metadataStore, logger),
			AddToProposalQHook:          hooks.NewAddToProposalQHook(proposalQueue, logger),
			AddBlockHistoryHook:         hooks.NewAddBlockHistoryHook(metadataStore, logger),
			AddFromStagingHook:          hooks.NewAddFromStagingHook(resultStore, coordinator, logger),
			AddConditionalProposalsHook: hooks.NewAddConditionalProposalsHook(metadataStore, coordinator, logger),
			AddLogProposalsHook:         hooks.NewAddLogProposalsHook(metadataStore, coordinator, logger),
			Logger:                      logger,
		}

		previousOutcome := ocr2keepers2.AutomationOutcome{
			AgreedPerformables: []ocr2keepers.CheckResult{
				{
					WorkID: "workID",
				},
			},
			SurfacedProposals: [][]ocr2keepers.CoordinatedBlockProposal{
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

		_, err = plugin.Observation(context.Background(), outcomeCtx, types.Query{})
		assert.Error(t, err)
		assert.Equal(t, "check result cannot be ineligible", err.Error())
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
			ViewProposalsFn: func(upkeepType ocr2keepers.UpkeepType) []ocr2keepers.CoordinatedBlockProposal {
				return []ocr2keepers.CoordinatedBlockProposal{}
			},
			RemoveProposalsFn: func(proposal ...ocr2keepers.CoordinatedBlockProposal) {
				// no op
			},
		}

		resultStore := &mockResultStore{
			ViewFn: func() ([]ocr2keepers.CheckResult, error) {
				return []ocr2keepers.CheckResult{
					{
						WorkID:   "workID5",
						Eligible: true,
					},
					{
						WorkID:   "workID6",
						Eligible: true,
					},
				}, nil
			},
			RemoveFn: func(s ...string) {
				// no op
			},
		}

		coordinator := &mockCoordinator{
			FilterResultsFn: func(results []ocr2keepers.CheckResult) ([]ocr2keepers.CheckResult, error) {
				return results, nil
			},
			FilterProposalsFn: func(proposals []ocr2keepers.CoordinatedBlockProposal) ([]ocr2keepers.CoordinatedBlockProposal, error) {
				return proposals, nil
			},
		}

		queueItems := []ocr2keepers.CoordinatedBlockProposal{}

		proposalQueue := &mockProposalQueue{
			EnqueueFn: func(items ...ocr2keepers.CoordinatedBlockProposal) error {
				queueItems = append(queueItems, items...)
				return nil
			},
		}

		plugin := &ocr3Plugin{
			UpkeepTypeGetter: mockUpkeepTypeGetter,
			WorkIDGenerator: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				var triggerExtBytes []byte
				if trigger.LogTriggerExtension != nil {
					triggerExtBytes = trigger.LogTriggerExtension.LogIdentifier()
				}
				uid := &ocr2keepers.UpkeepIdentifier{}
				hash := crypto.Keccak256(append(uid[:], triggerExtBytes...))
				return hex.EncodeToString(hash[:])
			},
			RemoveFromStagingHook:       hooks.NewRemoveFromStagingHook(resultStore, logger),
			RemoveFromMetadataHook:      hooks.NewRemoveFromMetadataHook(metadataStore, logger),
			AddToProposalQHook:          hooks.NewAddToProposalQHook(proposalQueue, logger),
			AddBlockHistoryHook:         hooks.NewAddBlockHistoryHook(metadataStore, logger),
			AddFromStagingHook:          hooks.NewAddFromStagingHook(resultStore, coordinator, logger),
			AddConditionalProposalsHook: hooks.NewAddConditionalProposalsHook(metadataStore, coordinator, logger),
			AddLogProposalsHook:         hooks.NewAddLogProposalsHook(metadataStore, coordinator, logger),
			Logger:                      logger,
		}

		previousOutcome := ocr2keepers2.AutomationOutcome{
			AgreedPerformables: []ocr2keepers.CheckResult{
				{
					WorkID:       "290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563",
					Eligible:     true,
					GasAllocated: 1,
				},
			},
			SurfacedProposals: [][]ocr2keepers.CoordinatedBlockProposal{
				{
					{
						WorkID: "290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563",
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
		assert.Equal(t, types.Observation(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID5","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null},{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID6","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":null,"BlockHistory":[{"Number":3,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":4,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":5,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`), observation)
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
			ViewProposalsFn: func(upkeepType ocr2keepers.UpkeepType) []ocr2keepers.CoordinatedBlockProposal {
				return []ocr2keepers.CoordinatedBlockProposal{}
			},
			RemoveProposalsFn: func(proposal ...ocr2keepers.CoordinatedBlockProposal) {
				// no op
			},
		}

		resultStore := &mockResultStore{
			ViewFn: func() ([]ocr2keepers.CheckResult, error) {
				return []ocr2keepers.CheckResult{
					{
						WorkID:              "workID5",
						IneligibilityReason: 1,
						Eligible:            true,
					},
					{
						WorkID:   "workID6",
						Eligible: true,
					},
				}, nil
			},
			RemoveFn: func(s ...string) {
				// no op
			},
		}

		coordinator := &mockCoordinator{
			FilterResultsFn: func(results []ocr2keepers.CheckResult) ([]ocr2keepers.CheckResult, error) {
				return results[:1], nil
			},
			FilterProposalsFn: func(proposals []ocr2keepers.CoordinatedBlockProposal) ([]ocr2keepers.CoordinatedBlockProposal, error) {
				return proposals, nil
			},
		}

		queueItems := []ocr2keepers.CoordinatedBlockProposal{}

		proposalQueue := &mockProposalQueue{
			EnqueueFn: func(items ...ocr2keepers.CoordinatedBlockProposal) error {
				queueItems = append(queueItems, items...)
				return nil
			},
		}

		plugin := &ocr3Plugin{
			UpkeepTypeGetter: mockUpkeepTypeGetter,
			WorkIDGenerator: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				var triggerExtBytes []byte
				if trigger.LogTriggerExtension != nil {
					triggerExtBytes = trigger.LogTriggerExtension.LogIdentifier()
				}
				uid := &ocr2keepers.UpkeepIdentifier{}
				hash := crypto.Keccak256(append(uid[:], triggerExtBytes...))
				return hex.EncodeToString(hash[:])
			},
			RemoveFromStagingHook:       hooks.NewRemoveFromStagingHook(resultStore, logger),
			RemoveFromMetadataHook:      hooks.NewRemoveFromMetadataHook(metadataStore, logger),
			AddToProposalQHook:          hooks.NewAddToProposalQHook(proposalQueue, logger),
			AddBlockHistoryHook:         hooks.NewAddBlockHistoryHook(metadataStore, logger),
			AddFromStagingHook:          hooks.NewAddFromStagingHook(resultStore, coordinator, logger),
			AddConditionalProposalsHook: hooks.NewAddConditionalProposalsHook(metadataStore, coordinator, logger),
			AddLogProposalsHook:         hooks.NewAddLogProposalsHook(metadataStore, coordinator, logger),
			Logger:                      logger,
		}

		previousOutcome := ocr2keepers2.AutomationOutcome{
			AgreedPerformables: []ocr2keepers.CheckResult{
				{
					WorkID:       "290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563",
					Eligible:     true,
					GasAllocated: 1,
				},
			},
			SurfacedProposals: [][]ocr2keepers.CoordinatedBlockProposal{
				{
					{
						WorkID: "290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563",
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
		assert.Equal(t, types.Observation(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":1,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID5","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":null,"BlockHistory":[{"Number":3,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":4,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":5,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`), observation)
		assert.True(t, strings.Contains(logBuf.String(), "built an observation in sequence nr 0 with 1 performables, 0 upkeep proposals and 3 block history"))
	})

	t.Run("subsequent round processing, previous outcome will be non nil, when AddFromStagingHook errors, an error is returned", func(t *testing.T) {
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
			ViewProposalsFn: func(upkeepType ocr2keepers.UpkeepType) []ocr2keepers.CoordinatedBlockProposal {
				return []ocr2keepers.CoordinatedBlockProposal{}
			},
			RemoveProposalsFn: func(proposal ...ocr2keepers.CoordinatedBlockProposal) {
				// no op
			},
		}

		resultStore := &mockResultStore{
			ViewFn: func() ([]ocr2keepers.CheckResult, error) {
				return nil, errors.New("result store view boom")
			},
			RemoveFn: func(s ...string) {
				// no op
			},
		}

		coordinator := &mockCoordinator{
			FilterResultsFn: func(results []ocr2keepers.CheckResult) ([]ocr2keepers.CheckResult, error) {
				return results[:1], nil
			},
			FilterProposalsFn: func(proposals []ocr2keepers.CoordinatedBlockProposal) ([]ocr2keepers.CoordinatedBlockProposal, error) {
				return proposals, nil
			},
		}

		queueItems := []ocr2keepers.CoordinatedBlockProposal{}

		proposalQueue := &mockProposalQueue{
			EnqueueFn: func(items ...ocr2keepers.CoordinatedBlockProposal) error {
				queueItems = append(queueItems, items...)
				return nil
			},
		}

		plugin := &ocr3Plugin{
			UpkeepTypeGetter: mockUpkeepTypeGetter,
			WorkIDGenerator: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				var triggerExtBytes []byte
				if trigger.LogTriggerExtension != nil {
					triggerExtBytes = trigger.LogTriggerExtension.LogIdentifier()
				}
				uid := &ocr2keepers.UpkeepIdentifier{}
				hash := crypto.Keccak256(append(uid[:], triggerExtBytes...))
				return hex.EncodeToString(hash[:])
			},
			RemoveFromStagingHook:  hooks.NewRemoveFromStagingHook(resultStore, logger),
			RemoveFromMetadataHook: hooks.NewRemoveFromMetadataHook(metadataStore, logger),
			AddToProposalQHook:     hooks.NewAddToProposalQHook(proposalQueue, logger),
			AddBlockHistoryHook:    hooks.NewAddBlockHistoryHook(metadataStore, logger),
			AddFromStagingHook:     hooks.NewAddFromStagingHook(resultStore, coordinator, logger),
			Logger:                 logger,
		}

		previousOutcome := ocr2keepers2.AutomationOutcome{
			AgreedPerformables: []ocr2keepers.CheckResult{
				{
					WorkID:       "290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563",
					Eligible:     true,
					GasAllocated: 1,
				},
			},
			SurfacedProposals: [][]ocr2keepers.CoordinatedBlockProposal{
				{
					{
						WorkID: "290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563",
					},
				},
			},
		}
		previousOutcomeBytes, err := json.Marshal(previousOutcome)
		assert.NoError(t, err)
		outcomeCtx := ocr3types.OutcomeContext{
			PreviousOutcome: ocr3types.Outcome(previousOutcomeBytes),
		}

		_, err = plugin.Observation(context.Background(), outcomeCtx, types.Query{})
		assert.Error(t, err)
		assert.Equal(t, "result store view boom", err.Error())
	})

	t.Run("subsequent round processing, previous outcome will be non nil, when AddLogProposalsHook errors, an error is returned", func(t *testing.T) {
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
			ViewProposalsFn: func(upkeepType ocr2keepers.UpkeepType) []ocr2keepers.CoordinatedBlockProposal {
				return []ocr2keepers.CoordinatedBlockProposal{}
			},
			RemoveProposalsFn: func(proposal ...ocr2keepers.CoordinatedBlockProposal) {
				// no op
			},
		}

		resultStore := &mockResultStore{
			ViewFn: func() ([]ocr2keepers.CheckResult, error) {
				return []ocr2keepers.CheckResult{
					{
						WorkID:              "workID5",
						IneligibilityReason: 1,
						Eligible:            true,
					},
					{
						WorkID:   "workID6",
						Eligible: true,
					},
				}, nil
			},
			RemoveFn: func(s ...string) {
				// no op
			},
		}

		coordinator := &mockCoordinator{
			FilterResultsFn: func(results []ocr2keepers.CheckResult) ([]ocr2keepers.CheckResult, error) {
				return results[:1], nil
			},
			FilterProposalsFn: func(proposals []ocr2keepers.CoordinatedBlockProposal) ([]ocr2keepers.CoordinatedBlockProposal, error) {
				return nil, errors.New("filter proposals error")
			},
		}

		queueItems := []ocr2keepers.CoordinatedBlockProposal{}

		proposalQueue := &mockProposalQueue{
			EnqueueFn: func(items ...ocr2keepers.CoordinatedBlockProposal) error {
				queueItems = append(queueItems, items...)
				return nil
			},
		}

		plugin := &ocr3Plugin{
			UpkeepTypeGetter: mockUpkeepTypeGetter,
			WorkIDGenerator: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				var triggerExtBytes []byte
				if trigger.LogTriggerExtension != nil {
					triggerExtBytes = trigger.LogTriggerExtension.LogIdentifier()
				}
				uid := &ocr2keepers.UpkeepIdentifier{}
				hash := crypto.Keccak256(append(uid[:], triggerExtBytes...))
				return hex.EncodeToString(hash[:])
			},
			RemoveFromStagingHook:       hooks.NewRemoveFromStagingHook(resultStore, logger),
			RemoveFromMetadataHook:      hooks.NewRemoveFromMetadataHook(metadataStore, logger),
			AddToProposalQHook:          hooks.NewAddToProposalQHook(proposalQueue, logger),
			AddBlockHistoryHook:         hooks.NewAddBlockHistoryHook(metadataStore, logger),
			AddFromStagingHook:          hooks.NewAddFromStagingHook(resultStore, coordinator, logger),
			AddLogProposalsHook:         hooks.NewAddLogProposalsHook(metadataStore, coordinator, logger),
			AddConditionalProposalsHook: hooks.NewAddConditionalProposalsHook(metadataStore, coordinator, logger),
			Logger:                      logger,
		}

		previousOutcome := ocr2keepers2.AutomationOutcome{
			AgreedPerformables: []ocr2keepers.CheckResult{
				{
					WorkID:       "290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563",
					Eligible:     true,
					GasAllocated: 1,
				},
			},
			SurfacedProposals: [][]ocr2keepers.CoordinatedBlockProposal{
				{
					{
						WorkID: "290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563",
					},
				},
			},
		}
		previousOutcomeBytes, err := json.Marshal(previousOutcome)
		assert.NoError(t, err)
		outcomeCtx := ocr3types.OutcomeContext{
			PreviousOutcome: ocr3types.Outcome(previousOutcomeBytes),
		}

		_, err = plugin.Observation(context.Background(), outcomeCtx, types.Query{})
		assert.Error(t, err)
		assert.Equal(t, "filter proposals error", err.Error())
	})

	t.Run("subsequent round processing, previous outcome will be non nil, when AddConditionalProposalsHook errors, an error is returned", func(t *testing.T) {
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
			ViewProposalsFn: func(upkeepType ocr2keepers.UpkeepType) []ocr2keepers.CoordinatedBlockProposal {
				switch upkeepType {
				case ocr2keepers.ConditionTrigger:
					return []ocr2keepers.CoordinatedBlockProposal{
						{
							UpkeepID: [32]byte{1},
							Trigger: ocr2keepers.Trigger{
								BlockNumber: 1,
								BlockHash:   [32]byte{2},
							},
							WorkID: "workID1",
						},
					}
				}
				return []ocr2keepers.CoordinatedBlockProposal{}
			},
			RemoveProposalsFn: func(proposal ...ocr2keepers.CoordinatedBlockProposal) {
				// no op
			},
		}

		resultStore := &mockResultStore{
			ViewFn: func() ([]ocr2keepers.CheckResult, error) {
				return []ocr2keepers.CheckResult{
					{
						WorkID:              "workID5",
						IneligibilityReason: 1,
						Eligible:            true,
					},
					{
						WorkID:   "workID6",
						Eligible: true,
					},
				}, nil
			},
			RemoveFn: func(s ...string) {
				// no op
			},
		}

		coordinator := &mockCoordinator{
			FilterResultsFn: func(results []ocr2keepers.CheckResult) ([]ocr2keepers.CheckResult, error) {
				return results[:1], nil
			},
			FilterProposalsFn: func(proposals []ocr2keepers.CoordinatedBlockProposal) ([]ocr2keepers.CoordinatedBlockProposal, error) {
				if len(proposals) == 1 {
					return nil, errors.New("filter proposals error")
				}
				return []ocr2keepers.CoordinatedBlockProposal{}, nil
			},
		}

		queueItems := []ocr2keepers.CoordinatedBlockProposal{}

		proposalQueue := &mockProposalQueue{
			EnqueueFn: func(items ...ocr2keepers.CoordinatedBlockProposal) error {
				queueItems = append(queueItems, items...)
				return nil
			},
		}

		plugin := &ocr3Plugin{
			UpkeepTypeGetter: mockUpkeepTypeGetter,
			WorkIDGenerator: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				var triggerExtBytes []byte
				if trigger.LogTriggerExtension != nil {
					triggerExtBytes = trigger.LogTriggerExtension.LogIdentifier()
				}
				uid := &ocr2keepers.UpkeepIdentifier{}
				hash := crypto.Keccak256(append(uid[:], triggerExtBytes...))
				return hex.EncodeToString(hash[:])
			},
			RemoveFromStagingHook:       hooks.NewRemoveFromStagingHook(resultStore, logger),
			RemoveFromMetadataHook:      hooks.NewRemoveFromMetadataHook(metadataStore, logger),
			AddToProposalQHook:          hooks.NewAddToProposalQHook(proposalQueue, logger),
			AddBlockHistoryHook:         hooks.NewAddBlockHistoryHook(metadataStore, logger),
			AddFromStagingHook:          hooks.NewAddFromStagingHook(resultStore, coordinator, logger),
			AddLogProposalsHook:         hooks.NewAddLogProposalsHook(metadataStore, coordinator, logger),
			AddConditionalProposalsHook: hooks.NewAddConditionalProposalsHook(metadataStore, coordinator, logger),
			Logger:                      logger,
		}

		previousOutcome := ocr2keepers2.AutomationOutcome{
			AgreedPerformables: []ocr2keepers.CheckResult{
				{
					WorkID:       "290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563",
					Eligible:     true,
					GasAllocated: 1,
				},
			},
			SurfacedProposals: [][]ocr2keepers.CoordinatedBlockProposal{
				{
					{
						WorkID: "290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563",
					},
				},
			},
		}
		previousOutcomeBytes, err := json.Marshal(previousOutcome)
		assert.NoError(t, err)
		outcomeCtx := ocr3types.OutcomeContext{
			PreviousOutcome: ocr3types.Outcome(previousOutcomeBytes),
		}

		_, err = plugin.Observation(context.Background(), outcomeCtx, types.Query{})
		assert.Error(t, err)
		assert.Equal(t, "filter proposals error", err.Error())
	})

	t.Run("subsequent round processing, decoding an invalid previous outcome returns an error", func(t *testing.T) {
		plugin := &ocr3Plugin{
			Logger: log.New(io.Discard, "", 1),
		}

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
		wg          ocr2keepers.WorkIDGenerator
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
			wg: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				return "workID1"
			},
			observation: types.AttributedObservation{
				Observation: []byte(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":10,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,1],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":1,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`),
			},
		},
		{
			name: "gas allocated cannot be zero",
			observation: types.AttributedObservation{
				Observation: []byte(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":10,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,1],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`),
			},
			wg: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				return "workID1"
			},
			expectsErr: true,
			wantErr:    errors.New("gas allocated cannot be zero"),
		},
		{
			name: "mismatch in generated work ID",
			observation: types.AttributedObservation{
				Observation: []byte(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":10,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,1],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`),
			},
			wg: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				return "invalid work ID"
			},
			expectsErr: true,
			wantErr:    errors.New("incorrect workID within result"),
		},
		{
			name: "check result cannot be ineligible and have no ineligibility reason",
			observation: types.AttributedObservation{
				Observation: []byte(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`),
			},
			expectsErr: true,
			wantErr:    errors.New("check result cannot be ineligible"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plugin := &ocr3Plugin{
				Logger:           log.New(io.Discard, "ocr3-validate-observation-test", log.Ldate),
				UpkeepTypeGetter: mockUpkeepTypeGetter,
				WorkIDGenerator:  tc.wg,
			}
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
		wg           ocr2keepers.WorkIDGenerator
		wantOutcome  ocr3types.Outcome
		expectsErr   bool
		wantErr      error
	}{
		{
			name:         "processing an empty list of observations generates an empty outcome",
			observations: []types.AttributedObservation{},
			wantOutcome:  ocr3types.Outcome([]byte(`{"AgreedPerformables":null,"SurfacedProposals":[]}`)),
		},
		{
			name: "processing a well formed observation with a previous outcome generates an new outcome",
			observations: []types.AttributedObservation{
				{
					Observation: []byte(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":1,"PerformData":null,"FastGasWei":0,"LinkNative":0}],"UpkeepProposals":[],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`),
				},
			},
			wg: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				return "workID1"
			},
			prevOutcome: ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"Eligible":true,"GasAllocated":1,"WorkID":"workID1"}],"SurfacedProposals":[[{"WorkID":"workID1"}]]}`)),
			wantOutcome: ocr3types.Outcome([]byte(`{"AgreedPerformables":null,"SurfacedProposals":[[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}]]}`)),
		},
		{
			name: "processing a malformed observation with a previous outcome generates an new outcome",
			observations: []types.AttributedObservation{
				{
					Observation: []byte(`invalid`),
				},
			},
			wg: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				return "workID1"
			},
			prevOutcome: ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"Eligible":true,"GasAllocated":1,"WorkID":"workID1"}],"SurfacedProposals":[[{"WorkID":"workID1"}]]}`)),
			wantOutcome: ocr3types.Outcome([]byte(`{"AgreedPerformables":null,"SurfacedProposals":[[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}]]}`)),
		},
		{
			name: "processing an invalid observation with a previous outcome generates an new outcome",
			observations: []types.AttributedObservation{
				{
					Observation: []byte(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":0,"PerformData":null,"FastGasWei":0,"LinkNative":0}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`),
				},
			},
			wg: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				return "workID1"
			},
			prevOutcome: ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"Eligible":true,"GasAllocated":1,"WorkID":"workID1"}],"SurfacedProposals":[[{"WorkID":"workID1"}]]}`)),
			wantOutcome: ocr3types.Outcome([]byte(`{"AgreedPerformables":null,"SurfacedProposals":[[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}]]}`)),
		},
		{
			name: "processing an valid observation with a malformed previous outcome returns an error",
			observations: []types.AttributedObservation{
				{
					Observation: []byte(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":1,"PerformData":null,"FastGasWei":0,"LinkNative":0}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`),
				},
			},
			wg: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				return "workID1"
			},
			prevOutcome: ocr3types.Outcome([]byte(`invalid`)),
			expectsErr:  true,
			wantErr:     errors.New("invalid character 'i' looking for beginning of value"),
		},
		{
			name: "processing an valid observation with an invalid previous outcome returns an error",
			observations: []types.AttributedObservation{
				{
					Observation: []byte(`{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":1,"PerformData":null,"FastGasWei":0,"LinkNative":0}],"UpkeepProposals":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID2"}],"BlockHistory":[{"Number":1,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},{"Number":2,"Hash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]}`),
				},
			},
			wg: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				return "workID1"
			},
			prevOutcome: ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"WorkID":"workID1"}],"SurfacedProposals":[[{"WorkID":"workID1","GasAllocated":0}]]}`)),
			expectsErr:  true,
			wantErr:     errors.New("check result cannot be ineligible"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			logger := log.New(&logBuf, "ocr3-test-outcome", 0)

			plugin := &ocr3Plugin{
				UpkeepTypeGetter: mockUpkeepTypeGetter,
				WorkIDGenerator:  tc.wg,
				Logger:           logger,
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

// TODO: add tests for repeated upkeepIDs - is this possible to recreate at this level when we catch duplicate workIDs?
func TestOcr3Plugin_Reports(t *testing.T) {
	for _, tc := range []struct {
		name                string
		sequenceNumber      uint64
		outcome             ocr3types.Outcome
		wantReportsWithInfo []ocr3types.ReportWithInfo[AutomationReportInfo]
		encoder             ocr2keepers.Encoder
		utg                 ocr2keepers.UpkeepTypeGetter
		wg                  ocr2keepers.WorkIDGenerator
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
			name:           "a well formed but invalid outcome returns an error",
			sequenceNumber: 5,
			outcome:        ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}],"SurfacedProposals":[[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}]]}`)),
			encoder: &mockEncoder{
				EncodeFn: func(result ...ocr2keepers.CheckResult) ([]byte, error) {
					return json.Marshal(result)
				},
			},
			expectsErr: true,
			wantErr:    errors.New("check result cannot be ineligible"),
		},
		{
			name:           "when an invalid work ID is generated, an error is returned",
			sequenceNumber: 5,
			outcome:        ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"Eligible":true,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}],"SurfacedProposals":[[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}]]}`)),
			encoder: &mockEncoder{
				EncodeFn: func(result ...ocr2keepers.CheckResult) ([]byte, error) {
					return json.Marshal(result)
				},
			},
			utg: func(identifier ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.ConditionTrigger
			},
			wg: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				return "invalid work ID"
			},
			expectsErr: true,
			wantErr:    errors.New("incorrect workID within result"),
		},
		{
			name:           "when gas allocated is 0, an error is returned",
			sequenceNumber: 5,
			outcome:        ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"Eligible":true,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}],"SurfacedProposals":[[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}]]}`)),
			encoder: &mockEncoder{
				EncodeFn: func(result ...ocr2keepers.CheckResult) ([]byte, error) {
					return json.Marshal(result)
				},
			},
			utg: func(identifier ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.ConditionTrigger
			},
			wg: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				return "workID1"
			},
			expectsErr: true,
			wantErr:    errors.New("gas allocated cannot be zero"),
		},
		{
			name:           "a well formed report is encoded without error",
			sequenceNumber: 5,
			outcome:        ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"FastGasWei":1,"LinkNative":2,"GasAllocated":1,"Eligible":true,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}],"SurfacedProposals":[[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}]]}`)),
			encoder: &mockEncoder{
				EncodeFn: func(result ...ocr2keepers.CheckResult) ([]byte, error) {
					return json.Marshal(result)
				},
			},
			utg: func(identifier ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.ConditionTrigger
			},
			wg: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				return "workID1"
			},
			wantReportsWithInfo: []ocr3types.ReportWithInfo[AutomationReportInfo]{
				{
					Report: []byte(`[]`),
				},
				{
					Report: []byte(`[{"PipelineExecutionState":0,"Retryable":false,"Eligible":true,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1","GasAllocated":1,"PerformData":null,"FastGasWei":1,"LinkNative":2}]`),
				},
			},
		},
		{
			name:           "agreed performables with duplicate workIDs returns an error",
			sequenceNumber: 5,
			outcome:        ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"FastGasWei":1,"LinkNative":2,"GasAllocated":1,"Eligible":true,"UpkeepID":[1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"},{"FastGasWei":1,"LinkNative":2,"GasAllocated":1,"Eligible":true,"UpkeepID":[1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}],"SurfacedProposals":[[{"UpkeepID":[1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"},{"UpkeepID":[1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}]]}`)),
			encoder: &mockEncoder{
				EncodeFn: func(result ...ocr2keepers.CheckResult) ([]byte, error) {
					return json.Marshal(result)
				},
			},
			utg: func(identifier ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.ConditionTrigger
			},
			wg: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				return "workID1"
			},
			expectsErr: true,
			wantErr:    errors.New("agreed performable cannot have duplicate workIDs"),
		},
		{
			name:           "an error is returned when the encoder errors",
			sequenceNumber: 5,
			outcome:        ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"FastGasWei":1,"LinkNative":2,"GasAllocated":1,"Eligible":true,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}],"SurfacedProposals":[[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}]]}`)),
			encoder: &mockEncoder{
				EncodeFn: func(result ...ocr2keepers.CheckResult) ([]byte, error) {
					return nil, errors.New("encode boom")
				},
			},
			utg: func(identifier ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.ConditionTrigger
			},
			wg: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				return "workID1"
			},
			expectsErr: true,
			wantErr:    errors.New("error encountered while encoding: encode boom"),
		},
		{
			name:           "an error is returned when the encoder errors when there are still values to add",
			sequenceNumber: 5,
			outcome:        ocr3types.Outcome([]byte(`{"AgreedPerformables":[{"FastGasWei":1,"LinkNative":2,"GasAllocated":1,"Eligible":true,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}],"SurfacedProposals":[[{"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"workID1"}]]}`)),
			encoder: &mockEncoder{
				EncodeFn: func(result ...ocr2keepers.CheckResult) ([]byte, error) {
					if len(result) == 0 { // the first call to encode with this test passes 0 check results, so we want to error on the second call, which gets non-zero results
						return json.Marshal(result)
					}
					return nil, errors.New("encode boom")
				},
			},
			utg: func(identifier ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.ConditionTrigger
			},
			wg: func(identifier ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
				return "workID1"
			},
			expectsErr: true,
			wantErr:    errors.New("error encountered while encoding: encode boom"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			logger := log.New(&logBuf, "ocr3-test-reports", 0)

			plugin := &ocr3Plugin{
				Logger:           logger,
				ReportEncoder:    tc.encoder,
				UpkeepTypeGetter: tc.utg,
				WorkIDGenerator:  tc.wg,
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
					return upkeep.WorkID == "workID3"
				},
				AcceptFn: func(upkeep ocr2keepers.ReportedUpkeep) bool {
					return true
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
				AcceptFn: func(upkeep ocr2keepers.ReportedUpkeep) bool {
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
					return upkeep.WorkID == "workID3"
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
	ViewFn   func() ([]ocr2keepers.CheckResult, error)
	RemoveFn func(...string)
}

func (s *mockResultStore) View() ([]ocr2keepers.CheckResult, error) {
	return s.ViewFn()
}

func (s *mockResultStore) Remove(r ...string) {
	s.RemoveFn(r...)
}

type mockProposalQueue struct {
	EnqueueFn func(items ...ocr2keepers.CoordinatedBlockProposal) error
	DequeueFn func(t ocr2keepers.UpkeepType, n int) ([]ocr2keepers.CoordinatedBlockProposal, error)
}

func (s *mockProposalQueue) Enqueue(items ...ocr2keepers.CoordinatedBlockProposal) error {
	return s.EnqueueFn(items...)
}

func (s *mockProposalQueue) Dequeue(t ocr2keepers.UpkeepType, n int) ([]ocr2keepers.CoordinatedBlockProposal, error) {
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

type mockMetadataStore struct {
	ocr2keepers.MetadataStore
	ViewProposalsFn   func(upkeepType ocr2keepers.UpkeepType) []ocr2keepers.CoordinatedBlockProposal
	GetBlockHistoryFn func() ocr2keepers.BlockHistory
	RemoveProposalsFn func(...ocr2keepers.CoordinatedBlockProposal)
}

func (s *mockMetadataStore) ViewProposals(utype ocr2keepers.UpkeepType) []ocr2keepers.CoordinatedBlockProposal {
	return s.ViewProposalsFn(utype)

}

func (s *mockMetadataStore) GetBlockHistory() ocr2keepers.BlockHistory {
	return s.GetBlockHistoryFn()
}

func (s *mockMetadataStore) RemoveProposals(p ...ocr2keepers.CoordinatedBlockProposal) {
	s.RemoveProposalsFn(p...)
}

type mockCoordinator struct {
	ocr2keepers.Coordinator
	FilterProposalsFn func([]ocr2keepers.CoordinatedBlockProposal) ([]ocr2keepers.CoordinatedBlockProposal, error)
	FilterResultsFn   func([]ocr2keepers.CheckResult) ([]ocr2keepers.CheckResult, error)
	ShouldAcceptFn    func(ocr2keepers.ReportedUpkeep) bool
	ShouldTransmitFn  func(ocr2keepers.ReportedUpkeep) bool
	AcceptFn          func(ocr2keepers.ReportedUpkeep) bool
}

func (s *mockCoordinator) FilterProposals(p []ocr2keepers.CoordinatedBlockProposal) ([]ocr2keepers.CoordinatedBlockProposal, error) {
	return s.FilterProposalsFn(p)
}

func (s *mockCoordinator) FilterResults(res []ocr2keepers.CheckResult) ([]ocr2keepers.CheckResult, error) {
	return s.FilterResultsFn(res)
}

func (s *mockCoordinator) ShouldAccept(upkeep ocr2keepers.ReportedUpkeep) bool {
	return s.ShouldAcceptFn(upkeep)
}

func (s *mockCoordinator) ShouldTransmit(upkeep ocr2keepers.ReportedUpkeep) bool {
	return s.ShouldTransmitFn(upkeep)
}

func (s *mockCoordinator) Accept(r ocr2keepers.ReportedUpkeep) bool {
	return s.AcceptFn(r)
}

func mockWorkIDGenerator(id ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
	wid := string(id[:])
	if trigger.LogTriggerExtension != nil {
		wid += string(trigger.LogTriggerExtension.LogIdentifier())
	}
	return wid
}

func mockUpkeepTypeGetter(id ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
	if id.BigInt().Int64() < 10 {
		return ocr2keepers.ConditionTrigger
	}
	return ocr2keepers.LogTrigger
}
