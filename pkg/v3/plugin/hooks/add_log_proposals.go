package hooks

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type AddLogProposalsHook struct {
	metadata    types.MetadataStore
	coordinator types.Coordinator
	logger      *log.Logger
}

func NewAddLogProposalsHook(metadataStore types.MetadataStore, coordinator types.Coordinator, logger *log.Logger) AddLogProposalsHook {
	return AddLogProposalsHook{
		metadata:    metadataStore,
		coordinator: coordinator,
		logger:      log.New(logger.Writer(), fmt.Sprintf("[%s | build hook:add-log-recovery-proposals]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	}
}

func (h *AddLogProposalsHook) RunHook(obs *ocr2keepersv3.AutomationObservation, limit int, rSrc [16]byte) error {
	proposals := h.metadata.ViewLogRecoveryProposal()

	var err error
	proposals, err = h.coordinator.FilterProposals(proposals)
	if err != nil {
		return err
	}

	// TODO: Sort by work ID first
	// Shuffle using random seed
	rand.New(util.NewKeyedCryptoRandSource(rSrc)).Shuffle(len(proposals), func(i, j int) {
		proposals[i], proposals[j] = proposals[j], proposals[i]
	})

	// take first limit
	if len(proposals) > limit {
		proposals = proposals[:limit]
	}

	h.logger.Printf("adding %d log recovery proposals to observation", len(proposals))
	obs.UpkeepProposals = append(obs.UpkeepProposals, proposals...)
	return nil
}
