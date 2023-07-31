package postprocessors

import (
	"context"
	"testing"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMetadataAddPayload(t *testing.T) {
	ms := new(MockMetadataStore)
	values := []ocr2keepers.UpkeepPayload{}

	ms.On("Set", store.ProposalRecoveryMetadata, values)

	pp := NewAddPayloadToMetadataStorePostprocessor(ms)
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
