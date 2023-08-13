package plugin

import (
	"log"
	"math/rand"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type coordinatedBlockProposals struct {
	quorumBlockthreshold int
	roundHistoryLimit    int
	perRoundLimit        int
	keyRandSource        [16]byte
	logger               *log.Logger
	recentBlocks         map[ocr2keepers.BlockKey]int
	allNewProposals      []ocr2keepers.CoordinatedBlockProposal
}

func newCoordinatedBlockProposals(quorumBlockthreshold int, roundHistoryLimit int, perRoundLimit int, rSrc [16]byte, logger *log.Logger) *coordinatedBlockProposals {
	return &coordinatedBlockProposals{
		quorumBlockthreshold: quorumBlockthreshold,
		roundHistoryLimit:    roundHistoryLimit,
		perRoundLimit:        perRoundLimit,
		keyRandSource:        rSrc,
		logger:               logger,
		recentBlocks:         make(map[ocr2keepers.BlockKey]int),
	}
}

// SurfacedProposals gets constructed from a set of VALID observations and previous
// outcome.
//  1. History of proposals from previous outcome is carried over.
//  2. Those proposals which got an agreed performable are removed.
//  3. A latest quorum block is determined from the recent blocks in observations.
//  4. If no block achieves quorum then no new proposals are surfaced and we exit.
//  5. Oldest round's proposals are dropped if over limit to make room for new
//     surfaced proposals.
//  6. New proposals surfaced in this round are deduped, filtered from existing
//     proposals and performables and then added to the outcome with quorum block
//
// NOTE: Quorum is NOT applied on surfaced proposals apart from the block number
// A single node can surface a malicious proposal, it is expected that the nodes
// when acting on proposals will keep that in account when processing them.
func (c *coordinatedBlockProposals) add(ao ocr2keepersv3.AutomationObservation) {
	c.allNewProposals = append(c.allNewProposals, ao.UpkeepProposals...)
	for _, val := range ao.BlockHistory {
		_, present := c.recentBlocks[val]
		if present {
			c.recentBlocks[val]++
		} else {
			c.recentBlocks[val] = 1
		}
	}
}

func (c *coordinatedBlockProposals) set(outcome *ocr2keepersv3.AutomationOutcome, prevOutcome ocr2keepersv3.AutomationOutcome) {
	outcome.SurfacedProposals = [][]ocr2keepers.CoordinatedBlockProposal{}
	for _, round := range prevOutcome.SurfacedProposals {
		roundProposals := []ocr2keepers.CoordinatedBlockProposal{}
		for _, proposal := range round {
			if !performableExists(outcome.AgreedPerformables, proposal) {
				roundProposals = append(roundProposals, proposal)
			}
		}
		outcome.SurfacedProposals = append(outcome.SurfacedProposals, roundProposals)
	}
	latestQuorumBlock, ok := c.getLatestQuorumBlock()
	if !ok {
		// Can't coordinate new proposals without a quorum block, return with existing proposals
		return
	}
	// If existing outcome has more than roundHistoryLimit proposals, remove oldest ones
	// and make room to add one more
	if len(outcome.SurfacedProposals) >= c.roundHistoryLimit {
		outcome.SurfacedProposals = outcome.SurfacedProposals[:c.roundHistoryLimit-1]
	}
	latestProposals := []ocr2keepers.CoordinatedBlockProposal{}
	added := make(map[string]bool)
	for _, proposal := range c.allNewProposals {
		if proposalExists(outcome.SurfacedProposals, proposal) {
			// proposal already exists in history
			continue
		}
		if performableExists(outcome.AgreedPerformables, proposal) {
			// already being performed in this round, no need to propose
			continue
		}
		if added[proposal.WorkID] {
			// proposal already added in this round
			continue
		}

		// Coordinate the proposal on latest quorum block
		newProposal := proposal
		newProposal.Trigger.BlockNumber = latestQuorumBlock.Number
		newProposal.Trigger.BlockHash = latestQuorumBlock.Hash
		if newProposal.Trigger.LogTriggerExtension != nil {
			// Zero out blocknumber/hash for log triggers as these fields are not used
			// in the proposal
			newProposal.Trigger.LogTriggerExtension.BlockNumber = 0
			newProposal.Trigger.LogTriggerExtension.BlockHash = [32]byte{}
		}

		// TODO: Add logging here
		latestProposals = append(latestProposals, newProposal)
		added[proposal.WorkID] = true
	}

	// TODO: Sort by work ID first
	// Apply limit here on new proposals with random seed shuffling
	rand.New(util.NewKeyedCryptoRandSource(c.keyRandSource)).Shuffle(len(latestProposals), func(i, j int) {
		latestProposals[i], latestProposals[j] = latestProposals[j], latestProposals[i]
	})
	if len(latestProposals) > c.perRoundLimit {
		latestProposals = latestProposals[:c.perRoundLimit]
	}
	outcome.SurfacedProposals = append([][]ocr2keepers.CoordinatedBlockProposal{latestProposals}, outcome.SurfacedProposals...)
}

func (c *coordinatedBlockProposals) getLatestQuorumBlock() (ocr2keepers.BlockKey, bool) {
	var (
		mostRecent ocr2keepers.BlockKey
		zeroHash   [32]byte
	)

	for block, count := range c.recentBlocks {
		// Perhaps an honest node could be tricked into seeing an illegitimate
		// blockhash by an eclipse attack?
		if count > int(c.quorumBlockthreshold) {
			if (mostRecent.Hash == zeroHash) || // First consensus hash
				(block.Number > mostRecent.Number) || // later height
				(block.Number == mostRecent.Number && // Matching heights
					string(block.Hash[:]) > string(mostRecent.Hash[:])) { // Just need a defined ordered here
				mostRecent = block
			}
		}
	}
	return mostRecent, mostRecent.Hash != zeroHash
}

func proposalExists(existing [][]ocr2keepers.CoordinatedBlockProposal, new ocr2keepers.CoordinatedBlockProposal) bool {
	for _, round := range existing {
		for _, proposal := range round {
			if proposal.WorkID == new.WorkID {
				return true
			}
		}
	}
	return false
}

func performableExists(performables []ocr2keepers.CheckResult, proposal ocr2keepers.CoordinatedBlockProposal) bool {
	for _, performable := range performables {
		if proposal.WorkID == performable.WorkID {
			return true
		}
	}
	return false
}
