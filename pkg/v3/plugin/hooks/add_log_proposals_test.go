package hooks

import (
	"bytes"
	"errors"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func TestAddLogRecoveryProposalsHook_RunHook(t *testing.T) {
	for _, tc := range []struct {
		name             string
		metadata         types.MetadataStore
		coordinator      types.Coordinator
		proposals        []types.CoordinatedBlockProposal
		limit            int
		src              [16]byte
		wantNumProposals int
		expectErr        bool
		wantErr          error
	}{
		{
			name: "proposals aren't filtered and are added to the observation",
			metadata: &mockMetadataStore{
				ViewLogRecoveryProposalFn: func() []types.CoordinatedBlockProposal {
					return []types.CoordinatedBlockProposal{
						{
							WorkID: "workID1",
						},
					}
				},
			},
			coordinator: &mockCoordinator{
				FilterProposalsFn: func(proposals []types.CoordinatedBlockProposal) ([]types.CoordinatedBlockProposal, error) {
					assert.Equal(t, 1, len(proposals))
					return proposals, nil
				},
			},
			limit:            5,
			src:              [16]byte{1},
			wantNumProposals: 1,
		},
		{
			name: "proposals are filtered and are added to the observation",
			metadata: &mockMetadataStore{
				ViewLogRecoveryProposalFn: func() []types.CoordinatedBlockProposal {
					return []types.CoordinatedBlockProposal{
						{
							WorkID: "workID1",
						},
						{
							WorkID: "workID2",
						},
					}
				},
			},
			coordinator: &mockCoordinator{
				FilterProposalsFn: func(proposals []types.CoordinatedBlockProposal) ([]types.CoordinatedBlockProposal, error) {
					assert.Equal(t, 2, len(proposals))
					return proposals[:1], nil
				},
			},
			limit:            5,
			src:              [16]byte{1},
			wantNumProposals: 1,
		},
		{
			name: "proposals aren't filtered but are limited and are added to the observation",
			metadata: &mockMetadataStore{
				ViewLogRecoveryProposalFn: func() []types.CoordinatedBlockProposal {
					return []types.CoordinatedBlockProposal{
						{
							WorkID: "workID1",
						},
						{
							WorkID: "workID2",
						},
						{
							WorkID: "workID3",
						},
						{
							WorkID: "workID4",
						},
					}
				},
			},
			coordinator: &mockCoordinator{
				FilterProposalsFn: func(proposals []types.CoordinatedBlockProposal) ([]types.CoordinatedBlockProposal, error) {
					assert.Equal(t, 4, len(proposals))
					return proposals, nil
				},
			},
			limit:            2,
			src:              [16]byte{0},
			wantNumProposals: 2,
		},
		{
			name: "if an error is encountered filtering proposals, an error is returned",
			metadata: &mockMetadataStore{
				ViewLogRecoveryProposalFn: func() []types.CoordinatedBlockProposal {
					return []types.CoordinatedBlockProposal{
						{
							WorkID: "workID1",
						},
						{
							WorkID: "workID2",
						},
						{
							WorkID: "workID3",
						},
						{
							WorkID: "workID4",
						},
					}
				},
			},
			coordinator: &mockCoordinator{
				FilterProposalsFn: func(proposals []types.CoordinatedBlockProposal) ([]types.CoordinatedBlockProposal, error) {
					return nil, errors.New("filter proposals boom")
				},
			},
			limit:            2,
			src:              [16]byte{0},
			wantNumProposals: 0,
			expectErr:        true,
			wantErr:          errors.New("filter proposals boom"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			logger := log.New(&logBuf, "", 0)
			processor := NewAddLogProposalsHook(tc.metadata, tc.coordinator, logger)
			observation := &ocr2keepers.AutomationObservation{
				UpkeepProposals: tc.proposals,
			}
			err := processor.RunHook(observation, tc.limit, tc.src)
			if tc.expectErr {
				assert.Error(t, err)
				assert.Equal(t, tc.wantErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.wantNumProposals, len(observation.UpkeepProposals))
		})
	}
}

type mockMetadataStore struct {
	types.MetadataStore
	ViewLogRecoveryProposalFn func() []types.CoordinatedBlockProposal
	ViewConditionalProposalFn func() []types.CoordinatedBlockProposal
	GetBlockHistoryFn         func() types.BlockHistory
}

func (s *mockMetadataStore) ViewLogRecoveryProposal() []types.CoordinatedBlockProposal {
	return s.ViewLogRecoveryProposalFn()
}

func (s *mockMetadataStore) ViewConditionalProposal() []types.CoordinatedBlockProposal {
	return s.ViewConditionalProposalFn()
}

func (s *mockMetadataStore) GetBlockHistory() types.BlockHistory {
	return s.GetBlockHistoryFn()
}

type mockCoordinator struct {
	types.Coordinator
	FilterProposalsFn func([]types.CoordinatedBlockProposal) ([]types.CoordinatedBlockProposal, error)
	FilterResultsFn   func([]types.CheckResult) ([]types.CheckResult, error)
	ShouldAcceptFn    func(types.ReportedUpkeep) bool
	ShouldTransmitFn  func(types.ReportedUpkeep) bool
}

func (s *mockCoordinator) FilterProposals(p []types.CoordinatedBlockProposal) ([]types.CoordinatedBlockProposal, error) {
	return s.FilterProposalsFn(p)
}

func (s *mockCoordinator) FilterResults(res []types.CheckResult) ([]types.CheckResult, error) {
	return s.FilterResultsFn(res)
}

func (s *mockCoordinator) ShouldAccept(upkeep types.ReportedUpkeep) bool {
	return s.ShouldAcceptFn(upkeep)
}

func (s *mockCoordinator) ShouldTransmit(upkeep types.ReportedUpkeep) bool {
	return s.ShouldTransmitFn(upkeep)
}
