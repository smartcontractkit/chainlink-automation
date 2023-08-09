package simulators

import (
	"crypto/rand"
	"encoding/binary"
)

type cryptoRandSource struct{}

func newCryptoRandSource() cryptoRandSource {
	return cryptoRandSource{}
}

func (cryptoRandSource) Uint64() uint64 {
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint64(b[:])
}

func (cryptoRandSource) Seed(_ uint64) {}
