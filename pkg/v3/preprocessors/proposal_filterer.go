package preprocessors

import (
	"context"

	ocr2keepers "github.com/smartcontractkit/chainlink-common/pkg/types/automation"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

func NewProposalFilterer(metadata types.MetadataStore, upkeepType types.UpkeepType) ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload] {
	return &proposalFilterer{
		upkeepType: upkeepType,
		metadata:   metadata,
	}
}

type proposalFilterer struct {
	metadata   types.MetadataStore
	upkeepType types.UpkeepType
}

var _ ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload] = (*proposalFilterer)(nil)

// PreProcess returns all the payloads which don't currently exist in matadata store.
func (p *proposalFilterer) PreProcess(_ context.Context, payloads []ocr2keepers.UpkeepPayload) ([]ocr2keepers.UpkeepPayload, error) {
	all := p.metadata.ViewProposals(p.upkeepType)
	flatten := map[string]bool{}
	// can we make this more efficient? wonder if it's worth the effort
	for _, proposal := range all {
		flatten[proposal.WorkID] = true
	}
	filtered := make([]ocr2keepers.UpkeepPayload, 0)
	for _, payload := range payloads {
		if _, ok := flatten[payload.WorkID]; !ok {
			filtered = append(filtered, payload)
		}
	}

	return filtered, nil
}
