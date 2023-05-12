package types

import "testing"

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
				Filter1: "12345678901234567890123456789012",
				Filter2: "12345678901234567890123456789012",
				Filter3: "12345678901234567890123456789012",
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
			name: "invalid address",
			cfg: &LogUpkeepConfig{
				Address: "1234567890123456789012345678901234567890",
				Topic:   "12345678901234567890123456789012",
			},
			err: ErrContractAddrNoPrefix,
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
