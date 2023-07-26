package plugin

import (
	"testing"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/stretchr/testify/assert"
)

func TestRecoverables(t *testing.T) {
	merger := newRecoverables(2)

	observations := []ocr2keepersv3.AutomationObservation{
		{
			Metadata: map[ocr2keepersv3.ObservationMetadataKey]interface{}{
				ocr2keepersv3.RecoveryProposalObservationKey: []ocr2keepers.CoordinatedProposal{
					{
						UpkeepID: ocr2keepers.UpkeepIdentifier("1"),
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 1,
						},
					},
					{
						UpkeepID: ocr2keepers.UpkeepIdentifier("2"),
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 2,
						},
					},
				},
			},
		},
		{
			Metadata: map[ocr2keepersv3.ObservationMetadataKey]interface{}{
				ocr2keepersv3.RecoveryProposalObservationKey: []ocr2keepers.CoordinatedProposal{
					{
						UpkeepID: ocr2keepers.UpkeepIdentifier("1"),
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 1,
						},
					},
					{
						UpkeepID: ocr2keepers.UpkeepIdentifier("3"),
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 3,
						},
					},
				},
			},
		},
		{
			Metadata: map[ocr2keepersv3.ObservationMetadataKey]interface{}{
				ocr2keepersv3.RecoveryProposalObservationKey: []ocr2keepers.CoordinatedProposal{
					{
						UpkeepID: ocr2keepers.UpkeepIdentifier("2"),
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 2,
						},
					},
					{
						UpkeepID: ocr2keepers.UpkeepIdentifier("3"),
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 3,
						},
					},
				},
			},
		},
	}

	for _, o := range observations {
		merger.add(o)
	}

	outcome := ocr2keepersv3.AutomationOutcome{
		BasicOutcome: ocr2keepersv3.BasicOutcome{
			Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{},
		},
	}
	merger.set(&outcome)

	assert.Len(t, outcome.Metadata[ocr2keepersv3.CoordinatedRecoveryProposalKey].([]ocr2keepers.CoordinatedProposal), 3, "length of outcome should match")
}
