package types

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fxamacker/cbor/v2"
)

// UpkeepConfig is the interface for all upkeep configs.
// It is used to encode and decode the config into bytes,
// and validate the config.
type UpkeepConfig interface {
	// validates the config values
	Validate() error
	// encodes the config into bytes
	Encode() ([]byte, error)
	// decodes the config from bytes
	Decode(raw []byte) error
}

// TODO: implement ConditionalUpkeepConfig

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
	ErrTopicIsMissing        = errors.New("missing required field: topic")
	ErrTopicPrefix           = errors.New("invalid topic: prefixed with 0x")
)

func (cfg *LogUpkeepConfig) Validate() error {
	cfg.defaults()

	if len(cfg.Address) == 0 {
		return ErrContractAddrIsMissing
	}
	if n := len(cfg.Topic); n == 0 {
		return ErrTopicIsMissing
	}
	if strings.Index(cfg.Topic, "0x") == 0 {
		return ErrTopicPrefix
	}

	return nil
}

func (cfg *LogUpkeepConfig) Encode() ([]byte, error) {
	return cbor.Marshal(cfg)
}

func (cfg *LogUpkeepConfig) Decode(raw []byte) error {
	return cbor.Unmarshal(raw, cfg)
}

func (cfg *LogUpkeepConfig) defaults() {
	if len(cfg.Address) > 0 && strings.Index(cfg.Address, "0x") != 0 {
		cfg.Address = fmt.Sprintf("0x%s", cfg.Address)
	}
	if len(cfg.Filter1) > 0 && len(cfg.Filter1) < 32 {
		cfg.Filter1 = zeroPadding(cfg.Filter1)
	}
	if len(cfg.Filter2) > 0 && len(cfg.Filter2) < 32 {
		cfg.Filter2 = zeroPadding(cfg.Filter2)
	}
}

// padds the string with 32 0s to the left
func zeroPadding(s string) string {
	return fmt.Sprintf("%032s", s)
}
