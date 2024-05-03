package estimates

import (
	"encoding/json"
	"math/big"
	"testing"

	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	commontypes "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
	"github.com/stretchr/testify/assert"
)

func TestObservationLength(t *testing.T) {
	for _, tc := range []struct {
		name         string
		observation  *ocr2keepers.AutomationObservation
		expectedJSON string
		expectedSize int
	}{
		{
			name:         "Empty observation has 63 bytes of JSON",
			observation:  &ocr2keepers.AutomationObservation{},
			expectedJSON: `{"Performable":null,"UpkeepProposals":null,"BlockHistory":null}`,
			expectedSize: 63,
		},
		{
			name: "With non-nil performables, 61 bytes of JSON",
			observation: &ocr2keepers.AutomationObservation{
				Performable: []commontypes.CheckResult{},
			},
			expectedJSON: `{"Performable":[],"UpkeepProposals":null,"BlockHistory":null}`,
			expectedSize: 61,
		},
		{
			name: "With non-nil upkeep proposals, 61 bytes of JSON",
			observation: &ocr2keepers.AutomationObservation{
				UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
			},
			expectedJSON: `{"Performable":null,"UpkeepProposals":[],"BlockHistory":null}`,
			expectedSize: 61,
		},
		{
			name: "With non-nil block history, 61 bytes of JSON",
			observation: &ocr2keepers.AutomationObservation{
				BlockHistory: commontypes.BlockHistory{},
			},
			expectedJSON: `{"Performable":null,"UpkeepProposals":null,"BlockHistory":[]}`,
			expectedSize: 61,
		},
		{
			name: "With non-nil performable, upkeep proposals and block history, 57 bytes of JSON",
			observation: &ocr2keepers.AutomationObservation{
				Performable:     []commontypes.CheckResult{},
				UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
				BlockHistory:    commontypes.BlockHistory{},
			},
			expectedJSON: `{"Performable":[],"UpkeepProposals":[],"BlockHistory":[]}`,
			expectedSize: 57,
		},
		{
			name: "With one empty performable, upkeep proposals and block history, 438 bytes of JSON",
			observation: &ocr2keepers.AutomationObservation{
				Performable: []commontypes.CheckResult{
					{},
				},
				UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
				BlockHistory:    commontypes.BlockHistory{},
			},
			expectedJSON: `{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[],"BlockHistory":[]}`,
			expectedSize: 438,
		},
		{
			name: "With two empty performables, empty upkeep proposals and block history, 820 bytes of JSON",
			observation: &ocr2keepers.AutomationObservation{
				Performable: []commontypes.CheckResult{
					{},
					{},
				},
				UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
				BlockHistory:    commontypes.BlockHistory{},
			},
			expectedJSON: `{"Performable":[{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null},{"PipelineExecutionState":0,"Retryable":false,"Eligible":false,"IneligibilityReason":0,"UpkeepID":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Trigger":{"BlockNumber":0,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"LogTriggerExtension":null},"WorkID":"","GasAllocated":0,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[],"BlockHistory":[]}`,
			expectedSize: 820,
		},
		{
			name: "With one partially populated performable, empty upkeep proposals and block history, 473 bytes of JSON",
			observation: &ocr2keepers.AutomationObservation{
				Performable: []commontypes.CheckResult{
					{
						PipelineExecutionState: 10,
						Retryable:              true,
						Eligible:               true,
						IneligibilityReason:    100,
						UpkeepID:               commontypes.UpkeepIdentifier([32]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}),
						Trigger: commontypes.Trigger{
							BlockNumber: 102003244343430,
							BlockHash:   [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
						},
						WorkID:       "workID",
						GasAllocated: 102003244343430,
					},
				},
				UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
				BlockHistory:    commontypes.BlockHistory{},
			},
			expectedJSON: `{"Performable":[{"PipelineExecutionState":10,"Retryable":true,"Eligible":true,"IneligibilityReason":100,"UpkeepID":[1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Trigger":{"BlockNumber":102003244343430,"BlockHash":[1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"LogTriggerExtension":null},"WorkID":"workID","GasAllocated":102003244343430,"PerformData":null,"FastGasWei":null,"LinkNative":null}],"UpkeepProposals":[],"BlockHistory":[]}`,
			expectedSize: 473,
		},
		{
			name: "With one fully populated performable, empty upkeep proposals and block history, 684 bytes of JSON",
			observation: &ocr2keepers.AutomationObservation{
				Performable: []commontypes.CheckResult{
					{
						PipelineExecutionState: 10,
						Retryable:              true,
						Eligible:               true,
						IneligibilityReason:    100,
						UpkeepID:               commontypes.UpkeepIdentifier([32]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}),
						Trigger: commontypes.Trigger{
							BlockNumber: 102003244343430,
							BlockHash:   [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
							LogTriggerExtension: &commontypes.LogTriggerExtension{
								TxHash:      [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								Index:       1,
								BlockHash:   [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								BlockNumber: 102003244343430,
							},
						},
						WorkID:       "workID",
						GasAllocated: 102003244343430,
						PerformData:  []byte{1, 2, 3, 4},
						FastGasWei:   big.NewInt(3242352),
						LinkNative:   big.NewInt(4535654656436435),
					},
				},
				UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
				BlockHistory:    commontypes.BlockHistory{},
			},
			expectedJSON: `{"Performable":[{"PipelineExecutionState":10,"Retryable":true,"Eligible":true,"IneligibilityReason":100,"UpkeepID":[1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Trigger":{"BlockNumber":102003244343430,"BlockHash":[1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"LogTriggerExtension":{"TxHash":[1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Index":1,"BlockHash":[1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"BlockNumber":102003244343430}},"WorkID":"workID","GasAllocated":102003244343430,"PerformData":"AQIDBA==","FastGasWei":3242352,"LinkNative":4535654656436435}],"UpkeepProposals":[],"BlockHistory":[]}`,
			expectedSize: 684,
		},
		{
			name: "With one fully populated performable, empty upkeep proposals and block history, 692 bytes of JSON",
			observation: &ocr2keepers.AutomationObservation{
				Performable: []commontypes.CheckResult{
					{
						PipelineExecutionState: 10,
						Retryable:              true,
						Eligible:               true,
						IneligibilityReason:    100,
						UpkeepID:               commontypes.UpkeepIdentifier([32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}),
						Trigger: commontypes.Trigger{
							BlockNumber: 102003244343430,
							BlockHash:   [32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
							LogTriggerExtension: &commontypes.LogTriggerExtension{
								TxHash:      [32]byte{11, 2, 3, 4, 255, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								Index:       1,
								BlockHash:   [32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								BlockNumber: 102003244343430,
							},
						},
						WorkID:       "workID",
						GasAllocated: 102003244343430,
						PerformData:  []byte{11, 255, 255, 4},
						FastGasWei:   big.NewInt(3242352),
						LinkNative:   big.NewInt(4535654656436435),
					},
				},
				UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
				BlockHistory:    commontypes.BlockHistory{},
			},
			expectedJSON: `{"Performable":[{"PipelineExecutionState":10,"Retryable":true,"Eligible":true,"IneligibilityReason":100,"UpkeepID":[11,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Trigger":{"BlockNumber":102003244343430,"BlockHash":[11,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"LogTriggerExtension":{"TxHash":[11,2,3,4,255,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Index":1,"BlockHash":[11,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"BlockNumber":102003244343430}},"WorkID":"workID","GasAllocated":102003244343430,"PerformData":"C///BA==","FastGasWei":3242352,"LinkNative":4535654656436435}],"UpkeepProposals":[],"BlockHistory":[]}`,
			expectedSize: 692,
		},
		{
			name: "With one fully populated performable, empty upkeep proposals and non empty block history, 955 bytes of JSON",
			observation: &ocr2keepers.AutomationObservation{
				Performable: []commontypes.CheckResult{
					{
						PipelineExecutionState: 10,
						Retryable:              true,
						Eligible:               true,
						IneligibilityReason:    100,
						UpkeepID:               commontypes.UpkeepIdentifier([32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}),
						Trigger: commontypes.Trigger{
							BlockNumber: 102003244343430,
							BlockHash:   [32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
							LogTriggerExtension: &commontypes.LogTriggerExtension{
								TxHash:      [32]byte{11, 2, 3, 4, 255, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								Index:       1,
								BlockHash:   [32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								BlockNumber: 102003244343430,
							},
						},
						WorkID:       "workID",
						GasAllocated: 102003244343430,
						PerformData:  []byte{11, 255, 255, 4},
						FastGasWei:   big.NewInt(3242352),
						LinkNative:   big.NewInt(4535654656436435),
					},
				},
				UpkeepProposals: []commontypes.CoordinatedBlockProposal{},
				BlockHistory: commontypes.BlockHistory{
					commontypes.BlockKey{
						Number: 1,
						Hash:   [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
					},
					commontypes.BlockKey{
						Number: 2,
						Hash:   [32]byte{2, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
					},
					commontypes.BlockKey{
						Number: 3,
						Hash:   [32]byte{3, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
					},
				},
			},
			expectedJSON: `{"Performable":[{"PipelineExecutionState":10,"Retryable":true,"Eligible":true,"IneligibilityReason":100,"UpkeepID":[11,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Trigger":{"BlockNumber":102003244343430,"BlockHash":[11,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"LogTriggerExtension":{"TxHash":[11,2,3,4,255,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Index":1,"BlockHash":[11,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"BlockNumber":102003244343430}},"WorkID":"workID","GasAllocated":102003244343430,"PerformData":"C///BA==","FastGasWei":3242352,"LinkNative":4535654656436435}],"UpkeepProposals":[],"BlockHistory":[{"Number":1,"Hash":[1,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8]},{"Number":2,"Hash":[2,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8]},{"Number":3,"Hash":[3,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8]}]}`,
			expectedSize: 955,
		},
		{
			name: "With one fully populated performable, upkeep proposals and block history, 2283 bytes of JSON",
			observation: &ocr2keepers.AutomationObservation{
				Performable: []commontypes.CheckResult{
					{
						PipelineExecutionState: 10,
						Retryable:              true,
						Eligible:               true,
						IneligibilityReason:    100,
						UpkeepID:               commontypes.UpkeepIdentifier([32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}),
						Trigger: commontypes.Trigger{
							BlockNumber: 102003244343430,
							BlockHash:   [32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
							LogTriggerExtension: &commontypes.LogTriggerExtension{
								TxHash:      [32]byte{11, 2, 3, 4, 255, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								Index:       1,
								BlockHash:   [32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								BlockNumber: 102003244343430,
							},
						},
						WorkID:       "workID",
						GasAllocated: 102003244343430,
						PerformData:  []byte{11, 255, 255, 4},
						FastGasWei:   big.NewInt(3242352),
						LinkNative:   big.NewInt(4535654656436435),
					},
				},
				UpkeepProposals: []commontypes.CoordinatedBlockProposal{
					{
						UpkeepID: commontypes.UpkeepIdentifier([32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}),
						Trigger: commontypes.Trigger{
							BlockNumber: 102003244343430,
							BlockHash:   [32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
							LogTriggerExtension: &commontypes.LogTriggerExtension{
								TxHash:      [32]byte{11, 2, 3, 4, 255, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								Index:       1,
								BlockHash:   [32]byte{11, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								BlockNumber: 102003244343430,
							},
						},
						WorkID: "WorkID1",
					},
					{
						UpkeepID: commontypes.UpkeepIdentifier([32]byte{22, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}),
						Trigger: commontypes.Trigger{
							BlockNumber: 202003244343430,
							BlockHash:   [32]byte{22, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
							LogTriggerExtension: &commontypes.LogTriggerExtension{
								TxHash:      [32]byte{22, 2, 3, 4, 255, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								Index:       1,
								BlockHash:   [32]byte{22, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								BlockNumber: 202003244343430,
							},
						},
						WorkID: "WorkID2",
					},
					{
						UpkeepID: commontypes.UpkeepIdentifier([32]byte{33, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}),
						Trigger: commontypes.Trigger{
							BlockNumber: 302003244343430,
							BlockHash:   [32]byte{33, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
							LogTriggerExtension: &commontypes.LogTriggerExtension{
								TxHash:      [32]byte{33, 2, 3, 4, 255, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								Index:       1,
								BlockHash:   [32]byte{33, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
								BlockNumber: 302003244343430,
							},
						},
						WorkID: "WorkID3",
					},
				},
				BlockHistory: commontypes.BlockHistory{
					commontypes.BlockKey{
						Number: 1,
						Hash:   [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
					},
					commontypes.BlockKey{
						Number: 2,
						Hash:   [32]byte{2, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
					},
					commontypes.BlockKey{
						Number: 3,
						Hash:   [32]byte{3, 2, 3, 4, 5, 6, 7, 8, 1, 244, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
					},
				},
			},
			expectedJSON: `{"Performable":[{"PipelineExecutionState":10,"Retryable":true,"Eligible":true,"IneligibilityReason":100,"UpkeepID":[11,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Trigger":{"BlockNumber":102003244343430,"BlockHash":[11,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"LogTriggerExtension":{"TxHash":[11,2,3,4,255,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Index":1,"BlockHash":[11,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"BlockNumber":102003244343430}},"WorkID":"workID","GasAllocated":102003244343430,"PerformData":"C///BA==","FastGasWei":3242352,"LinkNative":4535654656436435}],"UpkeepProposals":[{"UpkeepID":[11,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Trigger":{"BlockNumber":102003244343430,"BlockHash":[11,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"LogTriggerExtension":{"TxHash":[11,2,3,4,255,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Index":1,"BlockHash":[11,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"BlockNumber":102003244343430}},"WorkID":"WorkID1"},{"UpkeepID":[22,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Trigger":{"BlockNumber":202003244343430,"BlockHash":[22,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"LogTriggerExtension":{"TxHash":[22,2,3,4,255,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Index":1,"BlockHash":[22,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"BlockNumber":202003244343430}},"WorkID":"WorkID2"},{"UpkeepID":[33,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Trigger":{"BlockNumber":302003244343430,"BlockHash":[33,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"LogTriggerExtension":{"TxHash":[33,2,3,4,255,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"Index":1,"BlockHash":[33,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8],"BlockNumber":302003244343430}},"WorkID":"WorkID3"}],"BlockHistory":[{"Number":1,"Hash":[1,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8]},{"Number":2,"Hash":[2,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8]},{"Number":3,"Hash":[3,2,3,4,5,6,7,8,1,244,3,4,5,6,7,8,1,2,3,4,5,6,7,8,1,2,3,4,5,6,7,8]}]}`,
			expectedSize: 2283,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.observation)
			assert.NoError(t, err)
			assert.Equal(t, len(b), ObservationLength(tc.observation))
			assert.Equal(t, tc.expectedJSON, string(b))
			assert.Equal(t, tc.expectedSize, len(b))
		})

	}
}
