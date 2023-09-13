package node

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulate/upkeep"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/telemetry"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/util"
)

func (g *Group) WriteTransmitChart() {
	tw := table.NewWriter()
	tw.SetTitle("Transmitted Results")
	tw.AppendHeader(table.Row{"Block Number", "Round", "Sender", "Upkeep ID", "Check Block"})

	transmits := g.transmitter.Results()

	for _, transmit := range transmits {
		report, _ := util.DecodeCheckResultsFromReportBytes(transmit.Report)
		for _, result := range report {
			tw.AppendRow(
				table.Row{
					transmit.BlockNumber.String(),
					transmit.Round,
					shorten(transmit.SendingAddress, 5),
					shorten(result.UpkeepID.String(), 8),
					result.Trigger.BlockNumber,
				})
		}
	}

	fmt.Fprint(g.logger.Writer(), tw.Render())
	// the render function does not put a newline after the chart
	fmt.Fprint(g.logger.Writer(), "\n\n")
}

func (g *Group) ReportResults() {
	var keyIDLookup map[string][]string
	for _, col := range g.collectors {
		switch ct := col.(type) {
		case *telemetry.RPCCollector:
			err := ct.WriteResults()
			if err != nil {
				panic(err)
			}
		case *telemetry.ContractEventCollector:
			_, keyIDLookup = ct.Data()
		}
	}

	ub, err := newUpkeepStatsBuilder(g.upkeeps, g.transmitter.Results(), keyIDLookup, upkeep.Util{})
	if err != nil {
		g.logger.Printf("stats builder failure: %s", err)
	}

	g.logger.Println("================ summary ================")

	totalIDChecks := 0
	totalIDs := 0
	totalEligibles := 0
	totalPerforms := 0
	totalMisses := 0
	var avgPerformDelay float64 = -1
	var avgCheckDelay float64 = -1
	idCheckData := []int{}

	for _, id := range ub.UpkeepIDs() {
		stats := ub.UpkeepStats(id)
		totalIDs++
		totalEligibles += stats.Eligible
		totalMisses += stats.Missed
		totalPerforms += stats.Eligible - stats.Missed

		if stats.AvgPerformDelay >= 0 {
			if avgPerformDelay < 0 {
				avgPerformDelay = stats.AvgPerformDelay
			} else {
				avgPerformDelay = (avgPerformDelay + stats.AvgPerformDelay) / 2
			}
		}

		if stats.AvgCheckDelay >= 0 {
			if avgCheckDelay < 0 {
				avgCheckDelay = stats.AvgCheckDelay
			} else {
				avgCheckDelay = (avgCheckDelay + stats.AvgCheckDelay) / 2
			}
		}

		checks, checked := keyIDLookup[id]
		if checked {
			totalIDChecks += len(checks)
			idCheckData = append(idCheckData, len(checks))
		} else {
			idCheckData = append(idCheckData, 0)
		}

		if stats.Missed != 0 {
			g.logger.Printf("%s was missed %d times", shorten(id, 8), stats.Missed)
			g.logger.Printf("%s was eligible at %s", shorten(id, 8), strings.Join(ub.Eligibles(id), ", "))

			by := []string{}
			for _, tr := range ub.TransmitEvents(id) {
				v := fmt.Sprintf("[address=%s, round=%d, block=%s]", shorten(tr.SendingAddress, 5), tr.Round, tr.BlockNumber)
				by = append(by, v)
			}
			g.logger.Printf("%s transactions %s", shorten(id, 8), strings.Join(by, ", "))

			if checked {
				g.logger.Printf("%s was checked at %s", shorten(id, 8), strings.Join(checks, ", "))
			}
		}
	}

	g.logger.Printf("total ids: %d", totalIDs)
	g.logger.Printf("total checks by network: %d", totalIDChecks)

	g.logger.Printf(" ---- Statistics / Checks per ID ---")

	sort.Slice(idCheckData, func(i, j int) bool {
		return idCheckData[i] < idCheckData[j]
	})

	// average
	avgChecksPerID := float64(totalIDChecks) / float64(len(idCheckData))
	g.logger.Printf("average: %0.2f", avgChecksPerID)

	// median
	median, q1Data, q3Data := findMedianAndSplitData(idCheckData)
	q1, lowerOutliers, _ := findMedianAndSplitData(q1Data)
	q3, _, upperOutliers := findMedianAndSplitData(q3Data)
	iqr := q3 - q1

	g.logger.Printf("IQR: %0.2f", iqr)
	inIQR := 0
	for i := 0; i < len(idCheckData); i++ {
		if float64(idCheckData[i]) >= q1 && float64(idCheckData[i]) <= q3 {
			inIQR++
		}
	}
	g.logger.Printf("IQR percent of whole: %0.2f%s", float64(inIQR)/float64(len(idCheckData))*100, "%")

	lowest, lOutliers := findLowestAndOutliers(q1-(1.5*iqr), lowerOutliers)
	if lOutliers > 0 {
		g.logger.Printf("lowest value: %d", lowest)
		g.logger.Printf("lower outliers (count): %d", lOutliers)
	} else {
		g.logger.Printf("no outliers below lower fence")
	}

	g.logger.Printf("Lower Fence (Q1 - 1.5*IQR): %0.2f", q1-(1.5*iqr))
	g.logger.Printf("Q1: %0.2f", q1)
	g.logger.Printf("Median: %0.2f", median)
	g.logger.Printf("Q3: %0.2f", q3)
	g.logger.Printf("Upper Fence (Q3 + 1.5*IQR): %0.2f", q3+(1.5*iqr))

	highest, hOutliers := findHighestAndOutliers(q3+(1.5*iqr), upperOutliers)
	if hOutliers > 0 {
		g.logger.Printf("highest value: %d", highest)
		g.logger.Printf("upper outliers (count): %d", hOutliers)
	} else {
		g.logger.Printf("no outliers above upper fence")
	}

	g.logger.Printf(" ---- end ---")

	g.logger.Printf(" ---- Statistics / Transmits per Node (account) ---")
	accStats := ub.Transmits()
	for _, acc := range accStats {
		g.logger.Printf("account %s transmitted %d times (%.2f%s)", acc.Account, acc.Count, acc.Pct, "%")
	}
	g.logger.Printf(" ---- end ---")

	// average perform delay
	g.logger.Printf("average perform delay: %d blocks", int(math.Round(avgPerformDelay)))
	g.logger.Printf("average check delay: %d blocks", int(math.Round(avgCheckDelay)))
	g.logger.Printf("total eligibles: %d", totalEligibles)
	g.logger.Printf("total performs in a transaction: %d", totalPerforms)
	g.logger.Printf("total confirmed misses: %d", totalMisses)
	g.logger.Println("================ end ================")
}

func shorten(full string, outLen int) string {
	if utf8.RuneCountInString(full) < outLen {
		return full
	}

	return string([]byte(full)[:outLen])
}
