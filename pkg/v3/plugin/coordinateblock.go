package plugin

import (
	"fmt"
	"strconv"
	"strings"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
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
	number uint64
	hash   string
}

func (p *coordinateBlock) add(observation ocr2keepersv3.AutomationObservation) {
	rawHistory, ok := observation.Metadata[ocr2keepersv3.BlockHistoryObservationKey]
	if !ok {
		return
	}

	history, ok := rawHistory.(ocr2keepers.BlockHistory)
	if !ok {
		return
	}

	for _, key := range history.Keys() {
		// TODO: use helper function for splitting keys
		values := strings.Split(string(key), "|")
		if len(values) != 2 {
			continue
		}

		// TODO: don't use values at index
		v, err := strconv.ParseUint(values[0], 10, 64)
		if err != nil {
			continue
		}

		// TODO: don't use values at index
		height := heightHash{v, values[1]}

		if _, present := p.seen[height]; !present {
			p.seen[height] = struct{}{}
			p.recent[height]++
		}
	}
}

func (p *coordinateBlock) set(outcome *ocr2keepersv3.AutomationOutcome) {
	var (
		mostRecent heightHash
		zeroHash   string
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
