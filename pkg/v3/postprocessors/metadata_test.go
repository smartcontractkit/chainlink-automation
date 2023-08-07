package postprocessors

import (
	"context"
	"testing"

	"github.com/smartcontractkit/ocr2keepers/pkg/util"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMetadataAddPayload(t *testing.T) {
	ms := new(MockMetadataStore)
	values := []ocr2keepers.UpkeepPayload{
		{
			ID: "test",
			Upkeep: ocr2keepers.ConfiguredUpkeep{
				ID: ocr2keepers.UpkeepIdentifier("1"),
			},
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 1,
				BlockHash:   "test",
			},
		},
		{
			ID: "test1",
			Upkeep: ocr2keepers.ConfiguredUpkeep{
				ID: ocr2keepers.UpkeepIdentifier("2"),
			},
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 2,
				BlockHash:   "test2",
			},
		},
	}

	expected := []ocr2keepers.CoordinatedProposal{
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier("1"),
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 1,
				BlockHash:   "test",
			},
		},
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier("2"),
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 2,
				BlockHash:   "test2",
			},
		},
	}

	ar := util.NewCache[ocr2keepers.CoordinatedProposal](util.DefaultCacheExpiration)

	ms.On("Get", store.ProposalRecoveryMetadata).Return(ar, true)

	pp := NewAddPayloadToMetadataStorePostprocessor(ms)
	err := pp.PostProcess(context.Background(), values)

	assert.NoError(t, err, "no error expected from post processor")

	ms.AssertExpectations(t)

	result := make([]ocr2keepers.CoordinatedProposal, 0)
	for _, key := range ar.Keys() {
		v, _ := ar.Get(key)
		result = append(result, v)
	}

	assert.Len(t, result, len(expected), "values in synced array should match input")
}

func TestMetadataAddSamples(t *testing.T) {
	ms := new(MockMetadataStore)
	values := []ocr2keepers.CheckResult{
		{
			Eligible: true,
			Payload: ocr2keepers.UpkeepPayload{
				ID: "test",
				Upkeep: ocr2keepers.ConfiguredUpkeep{
					ID: ocr2keepers.UpkeepIdentifier("1"),
				},
			},
		},
		{
			Eligible: true,
			Payload: ocr2keepers.UpkeepPayload{
				ID: "test1",
				Upkeep: ocr2keepers.ConfiguredUpkeep{
					ID: ocr2keepers.UpkeepIdentifier("2"),
				},
			},
		},
		{
			Eligible: false,
			Payload: ocr2keepers.UpkeepPayload{
				ID: "test2",
				Upkeep: ocr2keepers.ConfiguredUpkeep{
					ID: ocr2keepers.UpkeepIdentifier("3"),
				},
			},
		},
	}

	expected := []ocr2keepers.UpkeepIdentifier{
		ocr2keepers.UpkeepIdentifier("1"),
		ocr2keepers.UpkeepIdentifier("2"),
	}

	ms.On("Set", store.ProposalSampleMetadata, expected)

	pp := NewAddSamplesToMetadataStorePostprocessor(ms)
	err := pp.PostProcess(context.Background(), values)

	assert.NoError(t, err, "no error expected from post processor")

	ms.AssertExpectations(t)
}

type MockMetadataStore struct {
	mock.Mock
}

func (_m *MockMetadataStore) Set(key store.MetadataKey, value interface{}) {
	_m.Called(key, value)
}

func (_m *MockMetadataStore) Get(key store.MetadataKey) (interface{}, bool) {
	ret := _m.Called(key)

	return ret.Get(0), ret.Get(1).(bool)
}
