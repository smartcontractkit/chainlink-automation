package plugin

import (
	"math/rand"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type coordinatedProposals struct {
	threshold         int
	roundHistoryLimit int
	perRoundLimit     int
	keyRandSource     [16]byte
	recentBlocks      map[ocr2keepers.BlockKey]int
	allNewProposals   []ocr2keepers.CoordinatedProposal
}

func newCoordinatedProposals(threshold int, roundHistoryLimit int, perRoundLimit int, rSrc [16]byte) *coordinatedProposals {
	return &coordinatedProposals{
		threshold:         threshold,
		roundHistoryLimit: roundHistoryLimit,
		perRoundLimit:     perRoundLimit,
		keyRandSource:     rSrc,
		recentBlocks:      make(map[ocr2keepers.BlockKey]int),
	}
}

func (c *coordinatedProposals) add(ao ocr2keepersv3.AutomationObservation) {
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

// TODO: Make the code more elegant if possible
func (c *coordinatedProposals) set(outcome *ocr2keepersv3.AutomationOutcome, prevOutcome ocr2keepersv3.AutomationOutcome) {
	// Keep proposals from previous outcome that haven't achieved quorum performable
	outcome.AgreedProposals = [][]ocr2keepers.CoordinatedProposal{}
	for _, round := range prevOutcome.AgreedProposals {
		roundProposals := []ocr2keepers.CoordinatedProposal{}
		for _, proposal := range round {
			if !performableExists(outcome.AgreedPerformables, proposal) {
				roundProposals = append(roundProposals, proposal)
			}
		}
		outcome.AgreedProposals = append(outcome.AgreedProposals, roundProposals)
	}
	latestQuorumBlock, ok := c.getLatestQuorumBlock()
	if !ok {
		// Can't coordinate new proposals without a quorum block, return with existing proposals
		return
	}
	// If existing outcome has more than roundHistoryLimit proposals, remove oldest ones
	// and make room to add one more
	if len(outcome.AgreedProposals) >= c.roundHistoryLimit {
		outcome.AgreedProposals = outcome.AgreedProposals[:c.roundHistoryLimit-1]
	}
	latestProposals := []ocr2keepers.CoordinatedProposal{}
	added := make(map[string]bool)
	for _, proposal := range c.allNewProposals {
		if proposalExists(outcome.AgreedProposals, proposal) {
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

		latestProposals = append(latestProposals, newProposal)
		added[proposal.WorkID] = true
	}

	// Apply limit here on new proposals with random seed shuffling
	rand.New(util.NewKeyedCryptoRandSource(c.keyRandSource)).Shuffle(len(latestProposals), func(i, j int) {
		latestProposals[i], latestProposals[j] = latestProposals[j], latestProposals[i]
	})
	if len(latestProposals) > c.perRoundLimit {
		latestProposals = latestProposals[:c.perRoundLimit]
	}

	outcome.AgreedProposals = append([][]ocr2keepers.CoordinatedProposal{latestProposals}, outcome.AgreedProposals...)
}

func (c *coordinatedProposals) getLatestQuorumBlock() (ocr2keepers.BlockKey, bool) {
	var (
		mostRecent ocr2keepers.BlockKey
		zeroHash   [32]byte
	)

	for block, count := range c.recentBlocks {
		// Perhaps an honest node could be tricked into seeing an illegitimate
		// blockhash by an eclipse attack?
		if count > int(c.threshold) {
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

func proposalExists(existing [][]ocr2keepers.CoordinatedProposal, new ocr2keepers.CoordinatedProposal) bool {
	for _, round := range existing {
		for _, proposal := range round {
			if proposal.WorkID == new.WorkID {
				return true
			}
		}
	}
	return false
}

func performableExists(performables []ocr2keepers.CheckResult, proposal ocr2keepers.CoordinatedProposal) bool {
	for _, performable := range performables {
		if proposal.WorkID == performable.WorkID {
			return true
		}
	}
	return false
}
