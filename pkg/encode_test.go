package ocr2keepers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkDecode(b *testing.B) {
	key1 := UpkeepKey([]byte("1239487928374|18768923479234987"))
	key2 := UpkeepKey([]byte("1239487928374|18768923479234989"))
	key3 := UpkeepKey([]byte("1239487928375|18768923479234987"))

	encoded := mustEncodeKeys([]UpkeepKey{key1, key2, key3})

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		var keys []UpkeepKey

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
		// the encoder appends a new line to the output string
		assert.Equal(t, b, []byte(`["1","2","3"]
`))
	})

	t.Run("fails to encode a channel", func(t *testing.T) {
		b, err := encode(make(chan int))
		assert.NotNil(t, err)
		assert.Nil(t, b)
	})
}

func mustEncodeKeys(keys []UpkeepKey) []byte {
	b, _ := encode(keys)
	return b
}
