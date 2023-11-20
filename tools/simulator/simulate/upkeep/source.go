package upkeep

import (
	"context"
	"log"
	"math/big"

	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/simulate/chain"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/util"
)

// Source maintains delivery of active upkeeps based on type and repeat
// return behavior.
type Source struct {
	Util

	// provided dependencies
	active    *ActiveTracker
	triggered *LogTriggerTracker
	logger    *log.Logger

	// configurations
	recoveryLookback int
}

func NewSource(
	active *ActiveTracker,
	triggered *LogTriggerTracker,
	lookback int,
	logger *log.Logger,
) *Source {
	return &Source{
		active:           active,
		triggered:        triggered,
		logger:           log.New(logger.Writer(), "[upkeep-source] ", log.Ldate|log.Ltime|log.Lshortfile),
		recoveryLookback: lookback,
	}
}

// GetActiveUpkeeps delivers all active conditional upkeep payloads. Payloads
// returned should be at least once and only when active.
func (src *Source) GetActiveUpkeeps(_ context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	upkeeps := src.active.GetAllByType(chain.ConditionalType)
	if len(upkeeps) == 0 {
		return nil, nil
	}

	payloads := make([]ocr2keepers.UpkeepPayload, len(upkeeps))

	for i, upkeep := range upkeeps {
		payloads[i] = makePayloadFromUpkeep(upkeep, src.active.GetLatestBlock())
	}

	src.logger.Printf("%d conditional upkeeps returned from source", len(payloads))

	return payloads, nil
}

// GetLatestPayloads returns payloads that have been triggered by watching logs.
// Each payload is delivered exactly once to the caller.
func (src *Source) GetLatestPayloads(_ context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	upkeeps := src.triggered.GetOnce()
	if len(upkeeps) == 0 {
		return nil, nil
	}

	payloads := make([]ocr2keepers.UpkeepPayload, len(upkeeps))
	for i, upkeep := range upkeeps {
		payloads[i] = makeLogPayloadFromUpkeep(upkeep, src.active.GetLatestBlock())
	}

	return payloads, nil
}

// GetRecoveryProposals returns payloads that have been triggered by watched
// logs and have already been returned by GetLatestPayloads but have not been
// completed. Each payload is delivered at least once but may be repeated
// multiple times.
func (src *Source) GetRecoveryProposals(_ context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	upkeeps := src.triggered.GetAfter(new(big.Int).SetInt64(int64(src.recoveryLookback)))
	if len(upkeeps) == 0 {
		return nil, nil
	}

	payloads := make([]ocr2keepers.UpkeepPayload, len(upkeeps))
	for i, upkeep := range upkeeps {
		payloads[i] = makeLogPayloadFromUpkeep(upkeep, src.active.GetLatestBlock())
	}

	return payloads, nil
}

// BuildPayloads creates payloads from proposals.
func (src *Source) BuildPayloads(_ context.Context, proposals ...ocr2keepers.CoordinatedBlockProposal) ([]ocr2keepers.UpkeepPayload, error) {
	var payloads []ocr2keepers.UpkeepPayload

	src.logger.Printf("request to build %d payloads", len(proposals))

	// TODO: consider checking for performed upkeeps
	for _, proposal := range proposals {
		// only add to payloads if proposal is in active list
		if simulated, ok := src.active.GetByUpkeepID(proposal.UpkeepID); ok {
			src.logger.Printf("block: %d", proposal.Trigger.BlockNumber)

			payload := ocr2keepers.UpkeepPayload{
				UpkeepID:  proposal.UpkeepID,
				Trigger:   proposal.Trigger,
				WorkID:    util.UpkeepWorkID(ocr2keepers.UpkeepIdentifier(simulated.UpkeepID), proposal.Trigger),
				CheckData: simulated.CheckData,
			}

			payloads = append(payloads, payload)
		}
	}

	src.logger.Printf("%d payloads returned from builder", len(payloads))

	return payloads, nil
}

func makePayloadFromUpkeep(upkeep chain.SimulatedUpkeep, block chain.Block) ocr2keepers.UpkeepPayload {
	uid := ocr2keepers.UpkeepIdentifier(upkeep.UpkeepID)
	trigger := ocr2keepers.NewTrigger(ocr2keepers.BlockNumber(block.Number.Uint64()), block.Hash)

	return ocr2keepers.UpkeepPayload{
		UpkeepID:  uid,
		Trigger:   trigger,
		WorkID:    util.UpkeepWorkID(uid, trigger),
		CheckData: upkeep.CheckData,
	}
}

func makeLogPayloadFromUpkeep(triggered triggeredUpkeep, block chain.Block) ocr2keepers.UpkeepPayload {
	uid := ocr2keepers.UpkeepIdentifier(triggered.upkeep.UpkeepID)
	trigger := ocr2keepers.NewLogTrigger(ocr2keepers.BlockNumber(block.Number.Uint64()), block.Hash, &ocr2keepers.LogTriggerExtension{
		TxHash:      triggered.chainLog.TxHash,
		Index:       triggered.chainLog.Idx,
		BlockHash:   triggered.chainLog.BlockHash,
		BlockNumber: ocr2keepers.BlockNumber(triggered.chainLog.BlockNumber.Uint64()),
	})

	return ocr2keepers.UpkeepPayload{
		UpkeepID:  uid,
		Trigger:   trigger,
		WorkID:    util.UpkeepWorkID(uid, trigger),
		CheckData: triggered.upkeep.CheckData,
	}
}
