package loader

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"log"
	"sync"

	"github.com/smartcontractkit/chainlink-automation/tools/simulator/config"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/simulate/chain"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/util"
)

const (
	transmitProgressNamespace         = "Collecting upkeep perform events"
	transmitNegativeProgressNamespace = "No upkeep perform events expected"
)

type OCR3TransmitLoader struct {
	// provided dependencies
	progress ProgressTelemetry
	logger   *log.Logger

	// internal state values
	mu          sync.RWMutex
	queue       []*chain.TransmitEvent
	transmitted map[string]*chain.TransmitEvent
	namespace   string
}

// NewOCR3TransmitLoader accepts report bytes and adds them to incoming blocks
// as TransmitEvent transactions.
func NewOCR3TransmitLoader(plan config.SimulationPlan, progress ProgressTelemetry, logger *log.Logger) (*OCR3TransmitLoader, error) {
	namespace := transmitProgressNamespace
	if progress != nil {
		expected, err := calculateExpectedPerformEvents(plan)
		if err != nil {
			return nil, err
		}

		if expected == 0 {
			namespace = transmitNegativeProgressNamespace
		}

		if err := progress.Register(namespace, expected); err != nil {
			return nil, err
		}
	}

	return &OCR3TransmitLoader{
		progress:    progress,
		logger:      log.New(logger.Writer(), "[ocr3-transmit-loader] ", log.Ldate|log.Ltime|log.Lshortfile),
		queue:       make([]*chain.TransmitEvent, 0),
		transmitted: make(map[string]*chain.TransmitEvent),
		namespace:   namespace,
	}, nil
}

func (tl *OCR3TransmitLoader) Load(block *chain.Block) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if len(tl.queue) == 0 {
		return
	}

	transmits := make([]chain.TransmitEvent, 0, len(tl.queue))
	var performs int64

	for i := range tl.queue {
		tl.queue[i].BlockNumber = block.Number
		tl.queue[i].BlockHash = block.Hash

		performs += countPerformEvents(tl.queue[i].Report)

		transmits = append(transmits, *tl.queue[i])
	}

	tl.queue = []*chain.TransmitEvent{}

	block.Transactions = append(block.Transactions, chain.PerformUpkeepTransaction{
		Transmits: transmits,
	})

	if tl.progress != nil {
		tl.progress.Increment(tl.namespace, performs)
	}
}

func (tl *OCR3TransmitLoader) Transmit(from string, reportBytes []byte, round uint64) error {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	report := chain.TransmitEvent{
		Report: reportBytes,
		Round:  round,
	}

	var idHashBts bytes.Buffer
	if err := gob.NewEncoder(&idHashBts).Encode(report); err != nil {
		return err
	}

	transmitKey := hash(idHashBts.Bytes())

	report.SendingAddress = from

	var bts bytes.Buffer
	if err := gob.NewEncoder(&bts).Encode(report); err != nil {
		return err
	}

	report.Hash = rawHash(bts.Bytes())

	if _, ok := tl.transmitted[transmitKey]; ok {
		return fmt.Errorf("report already transmitted")
	}

	tl.logger.Printf("transmit sent from %s in round %d", from, round)

	tl.queue = append(tl.queue, &report)
	tl.transmitted[transmitKey] = &report

	return nil
}

func (tl *OCR3TransmitLoader) Results() []chain.TransmitEvent {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	events := []chain.TransmitEvent{}
	for _, r := range tl.transmitted {
		events = append(events, *r)
	}

	tl.logger.Printf("%d transmits returned in final results", len(events))

	return events
}

func calculateExpectedPerformEvents(plan config.SimulationPlan) (int64, error) {
	var count int64

	upkeeps, err := chain.GenerateAllUpkeeps(plan)
	if err != nil {
		return count, err
	}

	logs, err := chain.GenerateLogTriggers(plan)
	if err != nil {
		return count, err
	}

	for _, upkeep := range upkeeps {
		if !upkeep.Expected {
			continue
		}

		switch upkeep.Type {
		case chain.ConditionalType:
			count += int64(len(upkeep.EligibleAt))
		case chain.LogTriggerType:
			for _, log := range logs {
				if logTriggersUpkeep(log, upkeep) {
					count++
				}
			}
		}
	}

	return count, nil
}

func logTriggersUpkeep(log chain.SimulatedLog, upkeep chain.SimulatedUpkeep) bool {
	if log.TriggerAt.Cmp(upkeep.CreateInBlock) >= 0 && log.TriggerValue == upkeep.TriggeredBy {
		if upkeep.AlwaysEligible {
			return true
		} else {
			for _, block := range upkeep.EligibleAt {
				if block.Cmp(log.TriggerAt) >= 0 {
					return true
				}
			}
		}
	}

	return false
}

func countPerformEvents(report []byte) int64 {
	results, err := util.DecodeCheckResultsFromReportBytes(report)
	if err != nil {
		return 0
	}

	return int64(len(results))
}

func rawHash(b []byte) [32]byte {
	hasher := sha256.New()
	hasher.Write(b)

	var sum [32]byte

	copy(sum[:], hasher.Sum(nil)[:])

	return sum
}

func hash(b []byte) string {
	hasher := sha256.New()
	hasher.Write(b)

	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}
