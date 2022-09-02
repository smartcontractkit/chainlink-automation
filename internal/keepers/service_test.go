package keepers

import (
	"context"
	"testing"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestSimpleUpkeepService(t *testing.T) {
	t.Run("SetUpkeepState", func(t *testing.T) {
		tests := []struct {
			Key   []byte
			State types.UpkeepState
			Err   error
		}{
			{Key: []byte("test-key-1"), State: types.UpkeepState(1), Err: nil},
			{Key: []byte("test-key-2"), State: types.UpkeepState(2), Err: nil},
		}

		svc := &simpleUpkeepService{
			state: make(map[string]types.UpkeepState),
		}

		for _, test := range tests {
			err := svc.SetUpkeepState(context.Background(), types.UpkeepKey(test.Key), test.State)

			if test.Err == nil {
				assert.NoError(t, err, "should not return an error")
			} else {
				assert.Error(t, err, "should return an error")
			}

			assert.Contains(t, svc.state, string(test.Key), "internal state should contain key '%s'", test.Key)
			assert.Equal(t, svc.state[string(test.Key)], test.State, "internal state at key '%s' should be %d", test.Key, test.State)
		}
	})
}
