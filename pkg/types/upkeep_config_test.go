package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogTriggerUpkeepConfig_Validate(t *testing.T) {
	tests := []struct {
		name string
		cfg  *LogUpkeepConfig
		err  error
	}{
		{
			name: "happy flow",
			cfg: &LogUpkeepConfig{
				Address: "0x1234567890123456789012345678901234567890",
				Topic:   "12345678901234567890123456789012",
				Filter1: "123456789012345678",
				Filter2: "123456789012345678901234",
			},
		},
		{
			name: "missing address",
			cfg: &LogUpkeepConfig{
				Topic: "12345678901234567890123456789012",
			},
			err: ErrContractAddrIsMissing,
		},
		{
			name: "missing topic",
			cfg: &LogUpkeepConfig{
				Address: "0x1234567890123456789012345678901234567890",
			},
			err: ErrTopicIsMissing,
		},
		{
			name: "invalid topic prefix",
			cfg: &LogUpkeepConfig{
				Address: "0x1234567890123456789012345678901234567890",
				Topic:   "0x1234567890123456789012345678901234567890",
			},
			err: ErrTopicPrefix,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if err != nil {
				if tc.err == nil {
					t.Errorf("unexpected error: %v", err)
				} else if tc.err != err {
					t.Errorf("expected error: %v, got: %v", tc.err, err)
				}
			} else if tc.err != nil {
				t.Errorf("expected error: %v, got: %v", tc.err, err)
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, zeroPadding(tc.in))
		})
	}
}
