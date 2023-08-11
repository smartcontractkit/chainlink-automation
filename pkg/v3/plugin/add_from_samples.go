package plugin

import (
	"math/rand"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type AddFromSamplesHook struct {
	metadata types.MetadataStore
	coord    types.Coordinator
}

func NewAddFromSamplesHook(ms types.MetadataStore, coord types.Coordinator) AddFromSamplesHook {
	return AddFromSamplesHook{metadata: ms, coord: coord}
}

func (h *AddFromSamplesHook) RunHook(obs *ocr2keepersv3.AutomationObservation, limit int, rSrc [16]byte) error {
	conditionals := h.metadata.ViewConditionalProposal()
	var err error
	conditionals, err = h.coord.FilterProposals(conditionals)
	if err != nil {
		return err
	}

	// Shuffle using random seed
	rand.New(util.NewKeyedCryptoRandSource(rSrc)).Shuffle(len(conditionals), func(i, j int) {
		conditionals[i], conditionals[j] = conditionals[j], conditionals[i]
	})

	// take first limit
	if len(conditionals) > limit {
		conditionals = conditionals[:limit]
	}

	obs.UpkeepProposals = append(obs.UpkeepProposals, conditionals...)
	return nil
}
