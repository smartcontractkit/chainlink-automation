package types

import (
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestLogTriggerUpkeepConfig_Validate(t *testing.T) {
	// TODO: fix string convertion from common.Address and common.Hash

	tests := []struct {
		name    string
		cfg     *LogUpkeepConfig
		errored bool
	}{
		{
			name: "happy flow",
			cfg: &LogUpkeepConfig{
				Address: string(common.BytesToAddress(randomBytes(common.AddressLength)).Bytes()),
				Topic:   string(common.BytesToHash(randomBytes(common.HashLength)).Bytes()),
				Filter1: string(randomBytes(common.HashLength / 2)), // left-padded to 32 bytes
				Filter2: string(common.BytesToHash(randomBytes(common.HashLength)).Bytes()),
			},
		},
		{
			name: "missing address",
			cfg: &LogUpkeepConfig{
				Topic: string(common.BytesToHash(randomBytes(common.HashLength)).Bytes()),
			},
			errored: true,
		},
		{
			name: "missing topic",
			cfg: &LogUpkeepConfig{
				Address: string(common.BytesToAddress(randomBytes(common.AddressLength)).Bytes()),
			},
			errored: true,
		},
		{
			name: "invalid topic length: too short",
			cfg: &LogUpkeepConfig{
				Address: string(common.BytesToAddress(randomBytes(common.AddressLength)).Bytes()),
				Topic:   string(randomBytes(common.HashLength / 2)),
			},
			errored: true,
		},
		{
			name: "invalid topic length: too long",
			cfg: &LogUpkeepConfig{
				Address: string(common.BytesToAddress(randomBytes(common.AddressLength)).Bytes()),
				Topic:   string(randomBytes(common.HashLength * 2)),
			},
			errored: true,
		},
		{
			name: "invalid topic prefix",
			cfg: &LogUpkeepConfig{
				Address: string(common.BytesToAddress(randomBytes(common.AddressLength)).Bytes()),
				Topic:   "0x" + string(randomBytes(common.HashLength-2)),
			},
			errored: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.errored {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestZeroPadding(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty",
			in:   "",
			want: "00000000000000000000000000000000",
		},
		{
			name: "short",
			in:   "129",
			want: "00000000000000000000000000000129",
		},
		{
			name: "long",
			in:   "00000000000000000000000000000129111",
			want: "00000000000000000000000000000129111",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, ensureHashLength(tc.in))
		})
	}
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}
