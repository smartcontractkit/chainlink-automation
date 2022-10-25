package config

import (
	"math/big"
	"time"
)

type RunBook struct {
	BlockCadence Blocks
	Configs      map[uint64]Config
	Upkeeps      []Upkeep
}

type Blocks struct {
	Genesis *big.Int
	Cadence time.Duration
	// Duration is the number of blocks to simulate before blocks should stop
	// broadcasting
	Duration int
}

type Config struct {
	Count           int
	Signers         [][]byte
	Transmitters    []string
	F               int
	Onchain         []byte
	OffchainVersion int
	Offchain        []byte
}

type SymBlock struct {
	BlockNumber     *big.Int
	TransmittedData []byte
	LatestEpoch     *uint32
}

type Upkeep struct {
	Count        int
	StartID      *big.Int
	GenerateFunc string
	OffsetFunc   string
}
