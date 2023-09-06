package node

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator/chain"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type upkeepStatsBuilder struct {
	src              []chain.SimulatedUpkeep
	performsByID     map[string][]string
	eligiblesByID    map[string][]string
	checksByID       map[string][]string
	trsByID          map[string][]chain.TransmitEvent
	accountTransmits map[string]int
}

func newUpkeepStatsBuilder(
	upkeeps []chain.SimulatedUpkeep,
	transmits []chain.TransmitEvent,
	checks map[string][]string,
	encoder ocr2keepers.Encoder,
) (*upkeepStatsBuilder, error) {

	// count the number of transmits per account
	accTr := make(map[string]int)

	// each perform by id
	performsByID := make(map[string][]string)
	trsByID := make(map[string][]chain.TransmitEvent)

	for _, tr := range transmits {
		block, ok := new(big.Int).SetString(tr.InBlock, 10)
		if !ok {
			return nil, fmt.Errorf("block '%s' not parsable as big int", tr.InBlock)
		}

		// increment the number of transactions for this transaction's address
		_, ok = accTr[tr.SendingAddress]
		if !ok {
			accTr[tr.SendingAddress] = 0
		}
		accTr[tr.SendingAddress]++

		reported, err := encoder.Extract(tr.Report)
		if err != nil {
			return nil, fmt.Errorf("error decoding report: %s", err)
		}

		// tr.SendingAddress
		for _, trResult := range reported {
			performsByID[trResult.UpkeepID.String()] = append(performsByID[trResult.UpkeepID.String()], block.String())
			trsByID[trResult.UpkeepID.String()] = append(trsByID[trResult.UpkeepID.String()], tr)
		}
	}

	// each eligible point for each upkeep id
	elByID := make(map[string][]string)

	for _, u := range upkeeps {
		el := make([]string, len(u.EligibleAt))
		for i, e := range u.EligibleAt {
			el[i] = e.String()
		}

		elByID[u.ID.String()] = el
	}

	return &upkeepStatsBuilder{
		src:              upkeeps,
		performsByID:     performsByID,
		eligiblesByID:    elByID,
		checksByID:       checks,
		trsByID:          trsByID,
		accountTransmits: accTr,
	}, nil
}

func (b *upkeepStatsBuilder) UpkeepIDs() []string {
	ids := []string{}

	for _, u := range b.src {
		var found bool
		for _, id := range ids {
			if id == u.ID.String() {
				found = true
			}
		}

		if !found {
			ids = append(ids, u.ID.String())
		}
	}

	return ids
}

func (b *upkeepStatsBuilder) Eligibles(id string) []string {
	ids := []string{}

	x, ok := b.eligiblesByID[id]
	if ok {
		ids = x
	}

	return ids
}

func (b *upkeepStatsBuilder) Performs(id string) []string {
	ids := []string{}

	x, ok := b.performsByID[id]
	if ok {
		ids = x
	}

	return ids
}

func (b *upkeepStatsBuilder) TransmitEvents(id string) []chain.TransmitEvent {
	ids := []chain.TransmitEvent{}

	x, ok := b.trsByID[id]
	if ok {
		ids = x
	}

	return ids
}

func (b *upkeepStatsBuilder) Checks(id string) []string {
	ids := []string{}

	x, ok := b.checksByID[id]
	if ok {
		ids = x
	}

	return ids
}

func (b *upkeepStatsBuilder) UpkeepStats(id string) upkeepStats {

	eligible := []string{}
	performed := []string{}
	checked := []string{}

	el, ok := b.eligiblesByID[id]
	if ok {
		eligible = el
	}

	p, ok := b.performsByID[id]
	if ok {
		performed = p
	}

	c, ok := b.checksByID[id]
	if ok {
		checked = c
	}

	sort.Strings(eligible)
	sort.Strings(performed)
	sort.Strings(checked)

	var pDelay float64 = -1
	var cDelay float64 = -1
	var pStartAt int
	var cStartAt int
	for i := 0; i < len(eligible); i++ {
		if pStartAt < len(performed) {
			diff := -1
			for j := pStartAt; j < len(performed); j++ {
				if performed[j] > eligible[i] {
					a, _ := new(big.Int).SetString(performed[j], 10)
					b, _ := new(big.Int).SetString(eligible[i], 10)
					diff = int(new(big.Int).Sub(a, b).Int64())
					pStartAt = j + 1
					break
				}
			}

			if diff >= 0 {
				if pDelay < 0 {
					pDelay = float64(diff)
				} else {
					pDelay = (float64(diff) + pDelay) / 2
				}
			}
		}

		if cStartAt < len(checked) {
			diff := -1
			for j := cStartAt; j < len(checked); j++ {
				if checked[j] > eligible[i] {
					a, _ := new(big.Int).SetString(checked[j], 10)
					b, _ := new(big.Int).SetString(eligible[i], 10)
					diff = int(new(big.Int).Sub(a, b).Int64())
					cStartAt = j + 1
					break
				}
			}

			if diff >= 0 {
				if cDelay < 0 {
					cDelay = float64(diff)
				} else {
					cDelay = (float64(diff) + cDelay) / 2
				}
			}
		}
	}

	stats := upkeepStats{
		Eligible:        len(eligible),
		Missed:          len(eligible) - len(performed),
		AvgPerformDelay: pDelay,
		AvgCheckDelay:   cDelay,
	}

	return stats
}

func (b *upkeepStatsBuilder) Transmits() []transmitStats {
	stats := make([]transmitStats, len(b.accountTransmits))

	sum := 0
	for _, count := range b.accountTransmits {
		sum += count
	}

	i := 0
	for account, count := range b.accountTransmits {
		stats[i] = transmitStats{
			Account: account,
			Count:   count,
			Pct:     float64(count) / float64(sum) * 100,
		}
		i++
	}

	return stats
}

type upkeepStats struct {
	Eligible        int
	Missed          int
	AvgPerformDelay float64
	AvgCheckDelay   float64
}

type transmitStats struct {
	Account string
	Count   int
	Pct     float64
}
