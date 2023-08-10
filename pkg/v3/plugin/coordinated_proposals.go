package plugin

import (
	"math/rand"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type coordinatedProposals struct {
	allProposals    []ocr2keepers.CoordinatedProposal
	allBlockHistory []ocr2keepers.BlockHistory
}

func newCoordinatedProposals() *coordinatedProposals {
	return &coordinatedProposals{}
}

func (c *coordinatedProposals) add(ao ocr2keepersv3.AutomationObservation) {
	c.allProposals = append(c.allProposals, ao.UpkeepProposals...)
	c.allBlockHistory = append(c.allBlockHistory, ao.BlockHistory)
}

func (c *coordinatedProposals) set(outcome *ocr2keepersv3.AutomationOutcome) {
	// Find latest agreed block from allBlockHistory (with reportBlockLag applied)
	// Filter allProposals workID from existing outcome proposals
	// Remove last outcome.CoordinatedProposals if over limit
	// Append allProposals with latest agreed block to outcome.CoordinatedProposals
}

func dedupeShuffleObservations(upkeepIds []ocr2keepers.UpkeepIdentifier, keyRandSource [16]byte) []ocr2keepers.UpkeepIdentifier {
	uniqueKeys := dedupe(upkeepIds)

	rand.New(util.NewKeyedCryptoRandSource(keyRandSource)).Shuffle(len(uniqueKeys), func(i, j int) {
		uniqueKeys[i], uniqueKeys[j] = uniqueKeys[j], uniqueKeys[i]
	})

	return uniqueKeys
}

func dedupe(inputs []ocr2keepers.UpkeepIdentifier) []ocr2keepers.UpkeepIdentifier {
	output := make([]ocr2keepers.UpkeepIdentifier, 0, len(inputs))
	matched := make(map[string]struct{})

	for _, input := range inputs {
		key := input.String()
		_, ok := matched[key]
		if !ok {
			matched[key] = struct{}{}
			output = append(output, input)
		}
	}

	return output
}
