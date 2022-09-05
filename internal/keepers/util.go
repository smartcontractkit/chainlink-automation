package keepers

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	rnd "math/rand"

	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

var (
	ErrEncoding = fmt.Errorf("error encountered while encoding")
)

type cryptoRandSource struct{}

func newCryptoRandSource() cryptoRandSource {
	return cryptoRandSource{}
}

func (_ cryptoRandSource) Int63() int64 {
	var b [8]byte
	rand.Read(b[:])
	return int64(binary.LittleEndian.Uint64(b[:]) & (1<<63 - 1))
}

func (_ cryptoRandSource) Seed(_ int64) {}

func filterUpkeeps(upkeeps []*ktypes.UpkeepResult, filter ktypes.UpkeepState) []*ktypes.UpkeepResult {
	ret := make([]*ktypes.UpkeepResult, 0, len(upkeeps))

	for _, up := range upkeeps {
		if up.State == filter {
			ret = append(ret, up)
		}
	}

	return ret
}

func keyList(upkeeps []*ktypes.UpkeepResult) []ktypes.UpkeepKey {
	ret := make([]ktypes.UpkeepKey, len(upkeeps))

	for i, up := range upkeeps {
		ret[i] = up.Key
	}

	return ret
}

func encodeUpkeepKeys(keys []ktypes.UpkeepKey) ([]byte, error) {
	b, err := json.Marshal(keys)
	if err != nil {
		return b, fmt.Errorf("%w: %s", ErrEncoding, err)
	}

	return b, nil
}

type Shuffler[T any] interface {
	Shuffle([]T) []T
}

type cryptoShuffler[T any] struct{}

func (_ *cryptoShuffler[T]) Shuffle(a []T) []T {
	r := rnd.New(newCryptoRandSource())
	r.Shuffle(len(a), func(i, j int) {
		a[i], a[j] = a[j], a[i]
	})
	return a
}
