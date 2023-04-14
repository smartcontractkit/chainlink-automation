package v1

import (
	"errors"
	"testing"

	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func TestValidator_ValidateUpkeepKey(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  types.UpkeepKey

		wantValid bool
		wantErr   error
	}{
		{
			name:      "upkeep key with valid block key and upkeep identifier is valid",
			key:       chain.UpkeepKey("123|456"),
			wantValid: true,
		},
		{
			name:      "upkeep key with a single component is invalid",
			key:       chain.UpkeepKey("123"),
			wantValid: false,
			wantErr:   errors.New("upkeep key not parsable: missing data in upkeep key"),
		},
		{
			name:      "upkeep key with invalid block key is invalid",
			key:       chain.UpkeepKey("a|123"),
			wantValid: false,
			wantErr:   errors.New("block key is not a big int"),
		},
		{
			name:      "upkeep key with negative block key is invalid",
			key:       chain.UpkeepKey("-456|123"),
			wantValid: false,
			wantErr:   errors.New("block key is not a positive integer"),
		},
		{
			name:      "upkeep key with invalid upkeep identifier is invalid",
			key:       chain.UpkeepKey("123|a"),
			wantValid: false,
			wantErr:   errors.New("upkeep identifier is not a big int"),
		},
		{
			name:      "upkeep key with negative upkeep identifier is invalid",
			key:       chain.UpkeepKey("123|-456"),
			wantValid: false,
			wantErr:   errors.New("upkeep identifier is not a positive integer"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := validator{}
			isValid, err := e.ValidateUpkeepKey(tc.key)
			if isValid != tc.wantValid {
				t.Fatalf("unexpected validity, want %T, got %T ", tc.wantValid, isValid)
			}
			if tc.wantErr != nil && tc.wantErr.Error() != err.Error() {
				t.Fatalf("unexpected error: %s", err.Error())
			}
		})
	}
}

func TestValidator_ValidateUpkeepIdentifier(t *testing.T) {
	for _, tc := range []struct {
		name       string
		identifier types.UpkeepIdentifier

		wantValid bool
		wantErr   error
	}{
		{
			name:       "positive integer upkeep identifier is valid",
			identifier: types.UpkeepIdentifier("123"),
			wantValid:  true,
		},
		{
			name:       "non numeric upkeep identifier is invalid",
			identifier: types.UpkeepIdentifier("a"),
			wantValid:  false,
			wantErr:    errors.New("upkeep identifier is not a big int"),
		},
		{
			name:       "negative upkeep identifier is invalid",
			identifier: types.UpkeepIdentifier("-456"),
			wantValid:  false,
			wantErr:    errors.New("upkeep identifier is not a positive integer"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := validator{}
			isValid, err := e.ValidateUpkeepIdentifier(tc.identifier)
			if isValid != tc.wantValid {
				t.Fatalf("unexpected validity, want %T, got %T ", tc.wantValid, isValid)
			}
			if tc.wantErr != nil && tc.wantErr.Error() != err.Error() {
				t.Fatalf("unexpected error: %s", err.Error())
			}
		})
	}
}

func TestValidator_ValidateBlockKey(t *testing.T) {
	for _, tc := range []struct {
		name     string
		blockKey types.BlockKey

		wantValid bool
		wantErr   error
	}{
		{
			name:      "positive integer block key is valid",
			blockKey:  chain.BlockKey("123"),
			wantValid: true,
		},
		{
			name:      "non numeric block key is invalid",
			blockKey:  chain.BlockKey("a"),
			wantValid: false,
			wantErr:   errors.New("block key is not a big int"),
		},
		{
			name:      "negative block key is invalid",
			blockKey:  chain.BlockKey("-456"),
			wantValid: false,
			wantErr:   errors.New("block key is not a positive integer"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := validator{}
			isValid, err := e.ValidateBlockKey(tc.blockKey)
			if isValid != tc.wantValid {
				t.Fatalf("unexpected validity, want %T, got %T ", tc.wantValid, isValid)
			}
			if tc.wantErr != nil && tc.wantErr.Error() != err.Error() {
				t.Fatalf("unexpected error: %s", err.Error())
			}
		})
	}
}
