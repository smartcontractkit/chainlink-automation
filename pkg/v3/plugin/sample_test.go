package plugin

import (
	"io"
	"log"
	"testing"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/stretchr/testify/assert"
)

func TestSamples(t *testing.T) {
	var src [16]byte

	merger := newSamples(2, src, log.New(io.Discard, "", 0))

	observations := []ocr2keepersv3.AutomationObservation{
		{
			Metadata: map[ocr2keepersv3.ObservationMetadataKey]interface{}{
				ocr2keepersv3.SampleProposalObservationKey: []ocr2keepers.UpkeepIdentifier{
					ocr2keepers.UpkeepIdentifier("1"),
					ocr2keepers.UpkeepIdentifier("2"),
				},
			},
		},
		{
			Metadata: map[ocr2keepersv3.ObservationMetadataKey]interface{}{
				ocr2keepersv3.SampleProposalObservationKey: []ocr2keepers.UpkeepIdentifier{
					ocr2keepers.UpkeepIdentifier("1"),
					ocr2keepers.UpkeepIdentifier("2"),
					ocr2keepers.UpkeepIdentifier("3"),
				},
			},
		},
		{
			Metadata: map[ocr2keepersv3.ObservationMetadataKey]interface{}{
				ocr2keepersv3.SampleProposalObservationKey: []ocr2keepers.UpkeepIdentifier{
					ocr2keepers.UpkeepIdentifier("2"),
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

	assert.Len(t, outcome.Metadata[ocr2keepersv3.CoordinatedSamplesProposalKey].([]ocr2keepers.UpkeepIdentifier), 2, "length of outcome should match")
}
