package util

import (
	"crypto/cipher"
	"math/rand"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestCryptoRandSource(t *testing.T) {
	t.Run("creates a new CryptoRandSource", func(t *testing.T) {
		s := NewCryptoRandSource()
		i := s.Int63()
		assert.NotEqual(t, 0, i)
	})

	t.Run("panics on Seed", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected a panic, did not panic")
			}
		}()

		s := NewCryptoRandSource()
		s.Seed(int64(123))
	})

	t.Run("an error on crand.Read causes a panic", func(t *testing.T) {
		oldRandReadFn := randReadFn
		randReadFn = func(b []byte) (n int, err error) {
			return 0, errors.New("read error")
		}
		defer func() {
			randReadFn = oldRandReadFn
		}()
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected a panic, did not panic")
			}
		}()

		s := NewCryptoRandSource()
		s.Int63()
	})
}

func TestNewKeyedCryptoRandSource(t *testing.T) {
	t.Run("creates a new KeyedCryptoRandSource", func(t *testing.T) {
		src := NewKeyedCryptoRandSource([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
		i := src.Int63()
		assert.Equal(t, i, int64(1590255750952055259))

		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected a panic, did not panic")
			}
		}()
		src.Seed(99)
	})

	t.Run("fails to create a new KeyedCryptoRandSource due to invalid key size", func(t *testing.T) {
		oldNewCipherFn := newCipherFn
		newCipherFn = func(key []byte) (cipher.Block, error) {
			return nil, errors.New("invalid key")
		}
		defer func() {
			newCipherFn = oldNewCipherFn
		}()

		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected a panic, did not panic")
			}
		}()

		NewKeyedCryptoRandSource([16]byte{1})
	})
}

func TestShuffler_Shuffle(t *testing.T) {
	shuffler := Shuffler[int]{Source: rand.NewSource(0)}
	arr := []int{1, 2, 3}
	arr = shuffler.Shuffle(arr)
	assert.Equal(t, arr, []int{2, 1, 3})
}
