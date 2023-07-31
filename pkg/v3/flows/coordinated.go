package flows

import (
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
)

func historyFromRingBuffer(ring []ocr2keepersv3.BasicOutcome, nextIdx int) []ocr2keepersv3.BasicOutcome {
	outcome := make([]ocr2keepersv3.BasicOutcome, len(ring))
	idx := nextIdx

	for x := 0; x < len(ring); x++ {
		outcome[x] = ring[idx]
		idx++

		if idx >= len(ring) {
			idx = 0
		}
	}

	return outcome
}
