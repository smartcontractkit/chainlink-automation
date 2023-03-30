package keepers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func BenchmarkDecode(b *testing.B) {
	key1 := chain.UpkeepKey([]byte("1239487928374|18768923479234987"))
	key2 := chain.UpkeepKey([]byte("1239487928374|18768923479234989"))
	key3 := chain.UpkeepKey([]byte("1239487928375|18768923479234987"))

	encoded := mustEncodeKeys([]ktypes.UpkeepKey{key1, key2, key3})

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		var keys []ktypes.UpkeepKey

		b.StartTimer()
		err := decode(encoded, &keys)
		b.StopTimer()

		if err != nil {
			b.FailNow()
		}
	}
}

func Test_encode(t *testing.T) {
	t.Run("successfully encodes a string", func(t *testing.T) {
		b, err := encode([]string{"1", "2", "3"})
		assert.Nil(t, err)
		assert.Equal(t, b, []byte(`["1","2","3"]`))
	})

	t.Run("fails to encode a channel", func(t *testing.T) {
		b, err := encode(make(chan int))
		assert.NotNil(t, err)
		assert.Nil(t, b)
	})
}
