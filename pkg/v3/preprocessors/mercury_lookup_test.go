package preprocessors

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

func TestMercuryPreprocessor_PreProcess(t *testing.T) {
	t.Run("for a given set of payloads, mercury lookup is enabled as per the config", func(t *testing.T) {
		p := NewMercuryPreprocessor(true)
		payloads := []ocr2keepers.UpkeepPayload{
			ocr2keepers.NewUpkeepPayload(big.NewInt(1), 1, ocr2keepers.BlockKey("123"), ocr2keepers.Trigger{
				BlockNumber: 123,
				BlockHash:   "abc",
			}, []byte{}),
			ocr2keepers.NewUpkeepPayload(big.NewInt(2), 1, ocr2keepers.BlockKey("456"), ocr2keepers.Trigger{
				BlockNumber: 456,
				BlockHash:   "def",
			}, []byte{}),
		}

		assert.False(t, payloads[0].MercuryLookup)
		assert.False(t, payloads[1].MercuryLookup)
		payloads, err := p.PreProcess(context.Background(), payloads)
		assert.NoError(t, err)
		assert.True(t, payloads[0].MercuryLookup)
		assert.True(t, payloads[1].MercuryLookup)
	})

	t.Run("for a given set of payloads, mercury lookup is not enabled as per the config", func(t *testing.T) {
		p := NewMercuryPreprocessor(false)
		payloads := []ocr2keepers.UpkeepPayload{
			ocr2keepers.NewUpkeepPayload(big.NewInt(1), 1, ocr2keepers.BlockKey("123"), ocr2keepers.Trigger{
				BlockNumber: 123,
				BlockHash:   "abc",
			}, []byte{}),
			ocr2keepers.NewUpkeepPayload(big.NewInt(2), 1, ocr2keepers.BlockKey("456"), ocr2keepers.Trigger{
				BlockNumber: 456,
				BlockHash:   "def",
			}, []byte{}),
		}

		assert.False(t, payloads[0].MercuryLookup)
		assert.False(t, payloads[1].MercuryLookup)
		payloads, err := p.PreProcess(context.Background(), payloads)
		assert.NoError(t, err)
		assert.False(t, payloads[0].MercuryLookup)
		assert.False(t, payloads[1].MercuryLookup)
	})
}
