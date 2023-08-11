package plugin

import (
	"math/rand"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type AddLogRecoveryProposalsHook struct {
	metadata    types.MetadataStore
	coordinator types.Coordinator
}

func NewAddLogRecoveryProposalsHook(metadataStore types.MetadataStore, coordinator types.Coordinator) AddLogRecoveryProposalsHook {
	return AddLogRecoveryProposalsHook{metadata: metadataStore, coordinator: coordinator}
}

func (h *AddLogRecoveryProposalsHook) RunHook(obs *ocr2keepersv3.AutomationObservation, limit int, rSrc [16]byte) error {
	proposals := h.metadata.ViewLogRecoveryProposal()

	var err error
	proposals, err = h.coordinator.FilterProposals(proposals)
	if err != nil {
		return err
	}

	// Shuffle using random seed
	rand.New(util.NewKeyedCryptoRandSource(rSrc)).Shuffle(len(proposals), func(i, j int) {
		proposals[i], proposals[j] = proposals[j], proposals[i]
	})

	// take first limit
	if len(proposals) > limit {
		proposals = proposals[:limit]
	}

	obs.UpkeepProposals = append(obs.UpkeepProposals, proposals...)
	return nil
}
