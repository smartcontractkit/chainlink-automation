package flows

import (
	"context"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/flows/mocks"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewSampleProposalFlow(t *testing.T) {
	r := new(mocks.MockRatio)
	pp := new(mockedPreprocessor)
	up := new(mocks.MockUpkeepProvider)
	rn := &mockedRunner{eligibleAfter: 0}
	ms := new(mocks.MockMetadataStore)
	bs := &mockBlockSubscriber{
		ch: make(chan ocr2keepers.BlockHistory),
	}

	preprocessors := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{pp}

	svc, err := newSampleProposalFlow(preprocessors, r, up, bs, ms, rn, log.New(io.Discard, "", 0))

	assert.NoError(t, err, "no error from initialization")

	var wg sync.WaitGroup
	wg.Add(1)

	go func(svc service.Recoverable, ctx context.Context) {
		assert.NoError(t, svc.Start(ctx))
		wg.Done()
	}(svc, context.Background())

	testValues := []ocr2keepers.UpkeepPayload{
		{
			Upkeep: ocr2keepers.ConfiguredUpkeep{
				ID: ocr2keepers.UpkeepIdentifier("1"),
			},
		},
		{
			Upkeep: ocr2keepers.ConfiguredUpkeep{
				ID: ocr2keepers.UpkeepIdentifier("2"),
			},
		},
		{
			Upkeep: ocr2keepers.ConfiguredUpkeep{
				ID: ocr2keepers.UpkeepIdentifier("3"),
			},
		},
		{
			Upkeep: ocr2keepers.ConfiguredUpkeep{
				ID: ocr2keepers.UpkeepIdentifier("4"),
			},
		},
	}

	up.On("GetActiveUpkeeps", mock.Anything, mock.Anything).Return(testValues, nil)
	r.On("OfInt", 4).Return(1)
	ms.On("Set", store.ProposalSampleMetadata, mock.Anything).Times(1)

	bs.ch <- ocr2keepers.BlockHistory{
		ocr2keepers.BlockKey("4"),
		ocr2keepers.BlockKey("3"),
	}

	time.Sleep(1 * time.Second)

	assert.NoError(t, svc.Close(), "no error expected on shut down")

	r.AssertExpectations(t)
	up.AssertExpectations(t)
	ms.AssertExpectations(t)

	assert.Equal(t, 1, pp.Calls())

	wg.Wait()
}

type mockBlockSubscriber struct {
	ch chan ocr2keepers.BlockHistory
}

func (_m *mockBlockSubscriber) Subscribe() (int, chan ocr2keepers.BlockHistory, error) {
	return 0, _m.ch, nil
}

func (_m *mockBlockSubscriber) Unsubscribe(int) error {
	return nil
}
