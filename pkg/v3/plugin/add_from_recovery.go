package plugin

import (
	"math/rand"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
)

type AddLogRecoveryProposalsHook struct {
	metadata store.MetadataStore
	coord    Coordinator
}

func NewAddLogRecoveryProposalsHook(ms store.MetadataStore, coord Coordinator) AddLogRecoveryProposalsHook {
	return AddLogRecoveryProposalsHook{metadata: ms, coord: coord}
}

func (h *AddLogRecoveryProposalsHook) RunHook(obs *ocr2keepersv3.AutomationObservation, limit int, rSrc [16]byte) error {
	// TODO: Read log recovery proposals from metadata store
	proposals := h.metadata.ViewLogRecoveryProposal()
	// TODO: filter proposals using coordinator
	var err error
	proposals, err = h.coord.FilterProposals(proposals)
	if err != nil {
		return err
	}

	// Shuffle using random seed
	rand.New(util.NewKeyedCryptoRandSource(rSrc)).Shuffle(len(proposals), func(i, j int) {
		proposals[i], proposals[j] = proposals[j], proposals[i]
	})

	// take first limit
	proposals = proposals[:limit]

	// TODO: Append to obs.CoordinatedProposals
	obs.UpkeepProposals = append(obs.UpkeepProposals, proposals...)

	return nil
}
