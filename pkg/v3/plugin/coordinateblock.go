package plugin

import (
	"fmt"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type coordinateBlock struct {
	threshold int
	seen      map[heightHash]struct{}
	recent    map[heightHash]int
}

func newCoordinateBlock(threshold int) *coordinateBlock {
	return &coordinateBlock{
		threshold: threshold,
		seen:      make(map[heightHash]struct{}),
		recent:    make(map[heightHash]int),
	}
}

type heightHash struct {
	number ocr2keepers.BlockNumber
	hash   [32]byte
}

// TODO: Use this logic within cootdinated_proposal
func (p *coordinateBlock) set(outcome *ocr2keepersv3.AutomationOutcome) {
	var (
		mostRecent heightHash
		zeroHash   [32]byte
	)

	for height, count := range p.recent {
		// Perhaps an honest node could be tricked into seeing an illegitimate
		// blockhash by an eclipse attack?
		// s.t+1 observations is sufficient as it guarantees that at least one comes
		// from an honest node. We don't require 2*s.t+1 appearances of the value
		// since faulty oracles could decide not to send a valid value.
		if count > int(p.threshold) {
			if (mostRecent.hash == zeroHash) || // First consensus hash
				(height.number > mostRecent.number) || // later height
				// Matching heights. Shouldn't be necessary if t ≥ n/3, and threshold
				// honest assumption holds, since can only get quorum on one hash at a
				// given height
				(height.number == mostRecent.number &&
					height.hash == mostRecent.hash) {
				mostRecent = height
			}
		}
	}

	// TODO: use helper function for composing block key
	outcome.Metadata[ocr2keepersv3.CoordinatedBlockOutcomeKey] = fmt.Sprintf("%d%s%s", mostRecent.number, "|", mostRecent.hash)
}
