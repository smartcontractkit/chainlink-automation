package preprocessors

import (
	"context"
	"io"
	"log"
	"sync"
	"testing"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/stretchr/testify/assert"
)

func TestLogPreProcessor(t *testing.T) {
	provider := newMockedProvider()
	preproc := NewLogPreProcessor(log.New(io.Discard, "", 0), provider)

	t.Run("pre process log events", func(t *testing.T) {
		payloads := []ocr2keepers.UpkeepPayload{
			{ID: "1"},
			{ID: "2"},
			{ID: "3"},
			{ID: "4"},
			{ID: "5"},
		}

		provider.addResults(payloads)

		preProcessedPayloads, err := preproc.PreProcess(context.Background(), nil)
		assert.NoError(t, err)
		assert.Equal(t, len(payloads), len(preProcessedPayloads))

		for i, payload := range preProcessedPayloads {
			assert.Equal(t, payloads[i].ID, payload.ID)
		}
	})

	t.Run("no log events", func(t *testing.T) {
		payloads, err := preproc.PreProcess(context.Background(), nil)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(payloads))
	})
}

type mockedProvider struct {
	lock    sync.Mutex
	results [][]ocr2keepers.UpkeepPayload
}

func newMockedProvider(results ...[]ocr2keepers.UpkeepPayload) *mockedProvider {
	return &mockedProvider{
		results: results,
	}
}

func (m *mockedProvider) GetLogs(context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if len(m.results) == 0 {
		return nil, nil
	}

	result := m.results[0]
	m.results = m.results[1:]
	return result, nil
}

func (m *mockedProvider) addResults(results []ocr2keepers.UpkeepPayload) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.results = append(m.results, results)
}
