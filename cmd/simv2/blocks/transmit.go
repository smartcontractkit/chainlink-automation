package blocks

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv2/config"
)

type TransmitLoader struct {
	mu          sync.Mutex
	queue       []*TransmitEvent
	transmitted map[string]*TransmitEvent
}

func NewTransmitLoader() *TransmitLoader {
	return &TransmitLoader{
		queue:       []*TransmitEvent{},
		transmitted: make(map[string]*TransmitEvent),
	}
}

func (l *TransmitLoader) Load(block *config.SymBlock) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.queue) > 0 {
		var lastEpoch uint32
		for _, r := range l.queue {
			if r.Epoch > lastEpoch {
				lastEpoch = r.Epoch
			}

			r.InBlock = block.BlockNumber.String()
			block.TransmittedData = append(block.TransmittedData, r.Report)
		}

		// set epoch to highest from transmitted values
		block.LatestEpoch = &lastEpoch

		// reset the queue
		l.queue = []*TransmitEvent{}
	}
}

func (l *TransmitLoader) Transmit(from string, reportBytes []byte, epoch uint32, round uint8) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	h := hash(reportBytes)

	r, ok := l.transmitted[h]
	if ok && r.Epoch == epoch {
		return fmt.Errorf("report already transmitted in epoch %d", epoch)
	}

	report := &TransmitEvent{
		SendingAddress: from,
		Report:         reportBytes,
		Hash:           h,
		Epoch:          epoch,
		Round:          round,
	}

	l.queue = append(l.queue, report)
	l.transmitted[h] = report

	return nil
}

func (l *TransmitLoader) Results() []TransmitEvent {
	events := []TransmitEvent{}
	for _, r := range l.transmitted {
		events = append(events, *r)
	}
	return events
}

type TransmitEvent struct {
	SendingAddress string
	Report         []byte
	Hash           string
	Epoch          uint32
	Round          uint8
	InBlock        string
}

func hash(b []byte) string {
	hasher := sha256.New()
	hasher.Write(b)
	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}
