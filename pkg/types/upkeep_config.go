package types

import (
	"errors"
	"strings"

	"github.com/fxamacker/cbor/v2"
)

// UpkeepConfig is the interface for all upkeep configs
type UpkeepConfig interface {
	Validate() error
	Encode() ([]byte, error)
	Decode(raw []byte) error
}

// LogUpkeepConfig holds the settings for a log upkeep
type LogUpkeepConfig struct {
	// Address is required, contract address w/ 0x prefix
	Address string `cbor:"a"`
	// Topic is required, 32 bytes, w/o 0x prefixed
	Topic string `cbor:"sig"`
	// Filter1 is optional, needs to be left-padded to 32 bytes
	Filter1 string `cbor:"f1,omitempty"`
	// Filter2 is optional, needs to be left-padded to 32 bytes
	Filter2 string `cbor:"f2,omitempty"`
	// Filter3 is wildcard filter if missing
	Filter3 string `cbor:"f3,omitempty"`
}

var _ UpkeepConfig = &LogUpkeepConfig{}

var (
	ErrContractAddrIsMissing = errors.New("missing required field: contract address")
	ErrContractAddrNoPrefix  = errors.New("invalid contract address: not prefixed with 0x")
	ErrTopicIsMissing        = errors.New("missing required field: topic")
	ErrTopicPrefix           = errors.New("invalid topic: prefixed with 0x")
)

func (cfg *LogUpkeepConfig) Validate() error {
	if len(cfg.Address) == 0 {
		return ErrContractAddrIsMissing
	}
	if strings.Index(cfg.Address, "0x") != 0 {
		// cfg.Address = fmt.Sprintf("0x%s", cfg.Address)
		return ErrContractAddrNoPrefix
	}
	if n := len(cfg.Topic); n == 0 {
		return ErrTopicIsMissing
	}
	if strings.Index(cfg.Topic, "0x") == 0 {
		return ErrTopicPrefix
	}

	// TODO: validate filters

	return nil
}

func (cfg *LogUpkeepConfig) Encode() ([]byte, error) {
	return cbor.Marshal(cfg)
}

func (cfg *LogUpkeepConfig) Decode(raw []byte) error {
	return cbor.Unmarshal(raw, cfg)
}
