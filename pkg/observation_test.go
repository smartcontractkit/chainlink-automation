package ocr2keepers

import (
	"encoding/json"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestObservation_UnmarshalJSON(t *testing.T) {
	t.Run("valid bytes unmarshal successfully", func(t *testing.T) {
		var observation Observation
		err := json.Unmarshal([]byte(`{"1":"123","2":["NDU2"]}`), &observation)
		assert.Nil(t, err)
		assert.Equal(t, Observation{
			BlockKey: BlockKey("123"),
			UpkeepIdentifiers: []UpkeepIdentifier{
				UpkeepIdentifier("456"),
			},
		}, observation)
	})

	t.Run("a mismatch in the original block key string and its parsed value causes an error", func(t *testing.T) {
		var observation Observation
		err := json.Unmarshal([]byte(`{"1":"06","2":["NDU2"]}`), &observation)
		assert.Equal(t, err, ErrInvalidBlockKey)
	})

	t.Run("unparsable block key results in an error", func(t *testing.T) {
		var observation Observation
		err := json.Unmarshal([]byte(`{"1":"invalid","2":["NDU2"]}`), &observation)
		assert.Equal(t, err, ErrBlockKeyNotParsable)
	})

	t.Run("a negative block key in an error", func(t *testing.T) {
		var observation Observation
		err := json.Unmarshal([]byte(`{"1":"-1","2":["NDU2"]}`), &observation) // "456"
		assert.Equal(t, err, ErrInvalidBlockKey)
	})

	t.Run("an invalid upkeep identifier causes an error", func(t *testing.T) {
		var observation Observation
		err := json.Unmarshal([]byte(`{"1":"123","2":["aW52YWxpZA=="]}`), &observation) // "invalid"
		assert.Equal(t, err, ErrUpkeepKeyNotParsable)
	})

	t.Run("a mismatch in the original upkeep id string and its parsed value causes an error", func(t *testing.T) {
		var observation Observation
		err := json.Unmarshal([]byte(`{"1":"123","2":["MDY="]}`), &observation) // "06"
		assert.Equal(t, err, ErrInvalidUpkeepIdentifier)
	})

	t.Run("a negative upkeep ID causes an error", func(t *testing.T) {
		var observation Observation
		err := json.Unmarshal([]byte(`{"1":"123","2":["LTE="]}`), &observation) // "-1"
		assert.Equal(t, err, ErrInvalidUpkeepIdentifier)
	})

	t.Run("an error in json unmarshalling results in an error", func(t *testing.T) {
		unmarshalErr := errors.New("unmarshal error")
		oldUnmarshalFn := unmarshalFn
		unmarshalFn = func(data []byte, v any) error {
			return unmarshalErr
		}
		defer func() {
			unmarshalFn = oldUnmarshalFn
		}()
		var observation Observation
		err := json.Unmarshal([]byte(`{"1":"123","2":["NDU2"]}`), &observation)
		assert.Equal(t, err, unmarshalErr)
	})
}
