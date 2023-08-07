package store

import (
	"testing"

	"github.com/smartcontractkit/ocr2keepers/pkg/util"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRecoveryProposalCacheFromMetadata(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		vg := new(mockValueGetter)

		cache := util.NewCache[ocr2keepers.CoordinatedProposal](util.DefaultCacheExpiration)

		vg.On("Get", ProposalRecoveryMetadata).Return(cache, true)

		value, err := RecoveryProposalCacheFromMetadata(vg)

		assert.NoError(t, err, "no error from call")
		assert.Equal(t, cache, value, "value return should be original cache")
	})

	t.Run("unavailable error", func(t *testing.T) {
		vg := new(mockValueGetter)

		vg.On("Get", ProposalRecoveryMetadata).Return(nil, false)

		_, err := RecoveryProposalCacheFromMetadata(vg)

		assert.ErrorIs(t, err, ErrMetadataUnavailable, "error from call")
	})

	t.Run("unavailable error", func(t *testing.T) {
		vg := new(mockValueGetter)

		vg.On("Get", ProposalRecoveryMetadata).Return("test", true)

		_, err := RecoveryProposalCacheFromMetadata(vg)

		assert.ErrorIs(t, err, ErrInvalidValueType, "error from call")
	})
}

func TestSampleProposalsFromMetadata(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		vg := new(mockValueGetter)

		expected := []ocr2keepers.UpkeepIdentifier{
			ocr2keepers.UpkeepIdentifier("1"),
		}

		vg.On("Get", ProposalSampleMetadata).Return(expected, true)

		value, err := SampleProposalsFromMetadata(vg)

		assert.NoError(t, err, "no error from call")
		assert.Equal(t, expected, value, "value return should be original array")
	})

	t.Run("unavailable error", func(t *testing.T) {
		vg := new(mockValueGetter)

		vg.On("Get", ProposalSampleMetadata).Return(nil, false)

		_, err := SampleProposalsFromMetadata(vg)

		assert.ErrorIs(t, err, ErrMetadataUnavailable, "error from call")
	})

	t.Run("unavailable error", func(t *testing.T) {
		vg := new(mockValueGetter)

		vg.On("Get", ProposalSampleMetadata).Return("test", true)

		_, err := SampleProposalsFromMetadata(vg)

		assert.ErrorIs(t, err, ErrInvalidValueType, "error from call")
	})
}

type mockValueGetter struct {
	mock.Mock
}

func (_m *mockValueGetter) Get(key MetadataKey) (interface{}, bool) {
	ret := _m.Mock.Called(key)

	return ret.Get(0), ret.Get(1).(bool)
}
