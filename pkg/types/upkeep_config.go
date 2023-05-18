package types

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"
)

const (
	zeroPrefix = "0x"
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
	Topic string `cbor:"t"`
	// Filter1 is optional, needs to be left-padded to 32 bytes
	Filter1 string `cbor:"f1,omitempty"`
	// Filter2 is optional, needs to be left-padded to 32 bytes
	Filter2 string `cbor:"f2,omitempty"`
	// Filter3 is wildcard filter if missing
	Filter3 string `cbor:"f3,omitempty"`
}

var _ UpkeepConfig = &LogUpkeepConfig{}

var (
	errTopicPrefix = errors.New("invalid topic: prefixed with 0x")
)

func (cfg *LogUpkeepConfig) Validate() error {
	cfg.defaults()

	if err := validateLength(cfg.Address, common.AddressLength+len(zeroPrefix)); err != nil {
		return errors.Wrap(err, "invalid contract address")
	}
	if err := validateLength(cfg.Topic, common.HashLength); err != nil {
		return errors.Wrap(err, "invalid topic")
	}
	if strings.HasPrefix(cfg.Topic, zeroPrefix) {
		return errTopicPrefix
	}

	if err := validateLength(cfg.Filter1, common.HashLength); err != nil {
		return errors.Wrap(err, "invalid filter1")
	}
	if err := validateLength(cfg.Filter2, common.HashLength); err != nil {
		return errors.Wrap(err, "invalid filter2")
	}

	// TODO: filter3

	return nil
}

func (cfg *LogUpkeepConfig) Encode() ([]byte, error) {
	return cbor.Marshal(cfg)
}

func (cfg *LogUpkeepConfig) Decode(raw []byte) error {
	return cbor.Unmarshal(raw, cfg)
}

func (cfg *LogUpkeepConfig) defaults() {
	if !strings.HasPrefix(cfg.Address, zeroPrefix) {
		cfg.Address = fmt.Sprintf("%s%s", zeroPrefix, cfg.Address)
	}
	cfg.Filter1 = ensureHashLength(cfg.Filter1)
	cfg.Filter2 = ensureHashLength(cfg.Filter2)

	// TODO: filter3
}

// ensureHashLength ensures the string is 32 bytes long, will pad with 0s if needed,
// and will return the string as is if it's already 32 bytes long.
func ensureHashLength(s string) string {
	if len(s) < common.HashLength {
		return fmt.Sprintf("%032s", s)
	}
	return s
}

// validateLength ensures the string is n bytes long
func validateLength(s string, n int) error {
	if len(s) != n {
		return fmt.Errorf("expected %d bytes, got %d", n, len(s))
	}
	return nil
}
