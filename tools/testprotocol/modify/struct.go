package modify

import (
	"context"
	"fmt"

	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-common/pkg/types/automation"
	ocr2keeperstypes "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

type NamedModifier func(context.Context, interface{}, error) (string, []byte, error)
type ObservationModifier func(context.Context, ocr2keepers.AutomationObservation, error) ([]byte, error)
type OutcomeModifier func(context.Context, ocr2keepers.AutomationOutcome, error) ([]byte, error)
type ProposalModifier func(proposals []types.CoordinatedBlockProposal) []types.CoordinatedBlockProposal
type PerformableModifier func(performables []types.CheckResult) []types.CheckResult
type BlockHistoryModifier func(history ocr2keeperstypes.BlockHistory) ocr2keeperstypes.BlockHistory

func WithProposalUpkeepIDAs(upkeepID types.UpkeepIdentifier) ProposalModifier {
	return func(proposals []types.CoordinatedBlockProposal) []types.CoordinatedBlockProposal {
		for _, proposal := range proposals {
			proposal.UpkeepID = upkeepID
		}

		return proposals
	}
}

func WithProposalBlockAs(block types.BlockNumber) ProposalModifier {
	return func(proposals []types.CoordinatedBlockProposal) []types.CoordinatedBlockProposal {
		for _, proposal := range proposals {
			proposal.Trigger.BlockNumber = block
		}

		return proposals
	}
}

func WithPerformableBlockAs(block types.BlockNumber) PerformableModifier {
	return func(performables []types.CheckResult) []types.CheckResult {
		for _, performable := range performables {
			performable.Trigger.BlockNumber = block
		}

		return performables
	}
}

func WithBlockHistoryBlockAs(number uint64) BlockHistoryModifier {
	return func(history ocr2keeperstypes.BlockHistory) ocr2keeperstypes.BlockHistory {
		for idx := range history {
			history[idx].Number = ocr2keeperstypes.BlockNumber(number)
		}

		return history
	}
}

func AsObservation(modifiers ...interface{}) ObservationModifier {
	return func(ctx context.Context, observation ocr2keepers.AutomationObservation, err error) ([]byte, error) {
		for _, IModifier := range modifiers {
			switch modifier := IModifier.(type) {
			case ProposalModifier:
				observation.UpkeepProposals = modifier(observation.UpkeepProposals)
			case PerformableModifier:
				observation.Performable = modifier(observation.Performable)
			case BlockHistoryModifier:
				observation.BlockHistory = modifier(observation.BlockHistory)
			default:
				return nil, fmt.Errorf("unexpected modifier type")
			}
		}

		return observation.Encode()
	}
}

func AsOutcome(modifiers ...interface{}) OutcomeModifier {
	return func(ctx context.Context, outcome ocr2keepers.AutomationOutcome, err error) ([]byte, error) {
		for _, IModifier := range modifiers {
			switch modifier := IModifier.(type) {
			case ProposalModifier:
				for idx, proposals := range outcome.SurfacedProposals {
					outcome.SurfacedProposals[idx] = modifier(proposals)
				}
			case PerformableModifier:
				outcome.AgreedPerformables = modifier(outcome.AgreedPerformables)
			default:
				return nil, fmt.Errorf("unexpected modifier type")
			}
		}

		return outcome.Encode()
	}
}

func Modify(name string, iModifier interface{}) NamedModifier {
	return func(ctx context.Context, value interface{}, err error) (string, []byte, error) {
		switch modifier := iModifier.(type) {
		case ObservationModifier:
			observation, ok := value.(ocr2keepers.AutomationObservation)
			if !ok {
				return name, nil, fmt.Errorf("value not an observation")
			}

			bts, err := modifier(ctx, observation, err)

			return name, bts, err
		case OutcomeModifier:
			outcome, ok := value.(ocr2keepers.AutomationOutcome)
			if !ok {
				return name, nil, fmt.Errorf("value not an outcome")
			}

			bts, err := modifier(ctx, outcome, err)

			return name, bts, err
		default:
			return name, nil, fmt.Errorf("unrecognized modifier type")
		}
	}
}
