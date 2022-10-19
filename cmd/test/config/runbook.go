package config

import (
	"math/big"
	"time"
)

type RunBook struct {
	BlockCadence Blocks
	Configs      map[uint64]Config
}

type Blocks struct {
	Genesis *big.Int
	Cadence time.Duration
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
	BlockNumber *big.Int
}
