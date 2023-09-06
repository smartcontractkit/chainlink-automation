package random

import (
	"crypto/cipher"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_GetRandomKeySource(t *testing.T) {
	tests := []struct {
		name   string
		prefix []byte
		seq    uint64
		want   [16]byte
	}{
		{
			name:   "happy path",
			prefix: []byte{1, 2, 3, 4},
			seq:    1234,
			want:   [16]byte{0x3e, 0xb0, 0x60, 0xd8, 0x59, 0x17, 0x19, 0xd, 0x80, 0x60, 0x41, 0xd4, 0x61, 0xaa, 0x5f, 0x12},
		},
		{
			name:   "nil prefix",
			prefix: nil,
			seq:    1234,
			want:   [16]byte{0x44, 0x36, 0xac, 0xd0, 0x86, 0xda, 0xf2, 0xaf, 0xf2, 0xd9, 0x40, 0xaf, 0x64, 0xf6, 0xb8, 0x84},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, GetRandomKeySource(tc.prefix, tc.seq))
		})
	}
}

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
