package v1

import (
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func TestEncoder_ValidateUpkeepKey(t *testing.T) {
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
			e := NewEncoder()
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

func TestEncoder_ValidateUpkeepIdentifier(t *testing.T) {
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
			e := NewEncoder()
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

func TestEncoder_ValidateBlockKey(t *testing.T) {
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
			e := NewEncoder()
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

func TestEncoder_MakeUpkeepKey(t *testing.T) {
	for _, tc := range []struct {
		name             string
		blockKey         types.BlockKey
		upkeepIdentifier types.UpkeepIdentifier

		wantUpkeepKey types.UpkeepKey
	}{
		{
			name:             "a block key of 123 and an upkeep identifier of 456 create a key of 123|456",
			blockKey:         chain.BlockKey("123"),
			upkeepIdentifier: types.UpkeepIdentifier("456"),
			wantUpkeepKey:    chain.UpkeepKey("123|456"),
		},
		{
			name:             "a block key of 999 and an upkeep identifier of 456 create a key of 999|456",
			blockKey:         chain.BlockKey("999"),
			upkeepIdentifier: types.UpkeepIdentifier("456"),
			wantUpkeepKey:    chain.UpkeepKey("999|456"),
		},
		{
			name:             "a block key of 999 and an upkeep identifier of 999 create a key of 999|999",
			blockKey:         chain.BlockKey("999"),
			upkeepIdentifier: types.UpkeepIdentifier("999"),
			wantUpkeepKey:    chain.UpkeepKey("999|999"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := NewEncoder()
			upkeepKey := e.MakeUpkeepKey(tc.blockKey, tc.upkeepIdentifier)
			if !reflect.DeepEqual(upkeepKey, tc.wantUpkeepKey) {
				t.Fatalf("unexpected upkeep key")
			}
		})
	}
}

func TestEncoder_SplitUpkeepKey(t *testing.T) {
	for _, tc := range []struct {
		name      string
		upkeepKey types.UpkeepKey

		wantBlockKey         types.BlockKey
		wantUpkeepIdentifier types.UpkeepIdentifier
		wantErr              error
	}{
		{
			name:                 "an upkeep key of 123|456 is split into a block key of 123 and an upkeep identifier of 456",
			upkeepKey:            chain.UpkeepKey("123|456"),
			wantBlockKey:         chain.BlockKey("123"),
			wantUpkeepIdentifier: types.UpkeepIdentifier("456"),
		},
		{
			name:                 "an upkeep key of 999|456 is split into a block key of 999 and an upkeep identifier of 456",
			upkeepKey:            chain.UpkeepKey("999|456"),
			wantBlockKey:         chain.BlockKey("999"),
			wantUpkeepIdentifier: types.UpkeepIdentifier("456"),
		},
		{
			name:                 "an upkeep key of a|456 is split into a block key of a and an upkeep identifier of 456, no validation is enforced",
			upkeepKey:            chain.UpkeepKey("a|456"),
			wantBlockKey:         chain.BlockKey("a"),
			wantUpkeepIdentifier: types.UpkeepIdentifier("456"),
		},
		{
			name:                 "an upkeep key of a|b is split into a block key of a and an upkeep identifier of b, no validation is enforced",
			upkeepKey:            chain.UpkeepKey("a|b"),
			wantBlockKey:         chain.BlockKey("a"),
			wantUpkeepIdentifier: types.UpkeepIdentifier("b"),
		},
		{
			name:    "an upkeep key of a|b is split into a block key of a and an upkeep identifier of b, no validation is enforced",
			wantErr: errors.New("upkeep key not parsable: missing data in upkeep key"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := NewEncoder()
			blockKey, upkeepIdentifier, err := e.SplitUpkeepKey(tc.upkeepKey)
			if !reflect.DeepEqual(blockKey, tc.wantBlockKey) {
				t.Fatalf("unexpected block key")
			}
			if !reflect.DeepEqual(upkeepIdentifier, tc.wantUpkeepIdentifier) {
				t.Fatalf("unexpected upkeep identifier")
			}
			if tc.wantErr != nil {
				if tc.wantErr.Error() != err.Error() {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			}
		})
	}
}

func TestEncoder_EncodeUpkeepIdentifier(t *testing.T) {
	for _, tc := range []struct {
		name         string
		upkeepResult types.UpkeepResult

		wantUpkeepIdentifier types.UpkeepIdentifier
		wantErr              error
	}{
		{
			name: "an upkeep result with an upkeep key of 123|456 returns an upkeep identifier of 456",
			upkeepResult: types.UpkeepResult{
				Key: chain.UpkeepKey("123|456"),
			},
			wantUpkeepIdentifier: types.UpkeepIdentifier("456"),
		},
		{
			name: "an upkeep result with an upkeep key of 123|999 returns an upkeep identifier of 999",
			upkeepResult: types.UpkeepResult{
				Key: chain.UpkeepKey("123|999"),
			},
			wantUpkeepIdentifier: types.UpkeepIdentifier("999"),
		},
		{
			name: "an upkeep result with an upkeep key of 555 returns an error",
			upkeepResult: types.UpkeepResult{
				Key: chain.UpkeepKey("555"),
			},
			wantErr: errors.New("upkeep key not parsable: missing data in upkeep key"),
		},
		{
			name: "an upkeep result with an upkeep key of 123|a returns an upkeep identifier of a, no validation is enforced on the identifier value",
			upkeepResult: types.UpkeepResult{
				Key: chain.UpkeepKey("123|a"),
			},
			wantUpkeepIdentifier: types.UpkeepIdentifier("a"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := NewEncoder()
			upkeepIdentifier, err := e.EncodeUpkeepIdentifier(tc.upkeepResult)
			if !reflect.DeepEqual(upkeepIdentifier, tc.wantUpkeepIdentifier) {
				t.Fatalf("unexpected upkeep identifier")
			}
			if tc.wantErr != nil {
				if tc.wantErr.Error() != err.Error() {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			}
		})
	}
}

func TestEncoder_Eligible(t *testing.T) {
	for _, tc := range []struct {
		name         string
		upkeepResult types.UpkeepResult

		wantEligible bool
		wantErr      error
	}{
		{
			name: "eligible state on upkeep result is deemed eligible",
			upkeepResult: types.UpkeepResult{
				State: types.Eligible,
			},
			wantEligible: true,
		},
		{
			name: "not eligible state on upkeep result is deemed ineligible",
			upkeepResult: types.UpkeepResult{
				State: types.NotEligible,
			},
			wantEligible: false,
		},
		{
			name: "unknown state on upkeep result is deemed ineligible",
			upkeepResult: types.UpkeepResult{
				State: 123,
			},
			wantEligible: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := NewEncoder()
			isEligible, err := e.Eligible(tc.upkeepResult)
			if isEligible != tc.wantEligible {
				t.Fatalf("unexpected eligibility, want %T, got %T ", tc.wantEligible, isEligible)
			}
			if tc.wantErr != nil && tc.wantErr.Error() != err.Error() {
				t.Fatalf("unexpected error: %s", err.Error())
			}
		})
	}
}

func TestEncoder_EncodeReport(t *testing.T) {
	t.Run("encoding an empty list of upkeep results returns a nil byte array and a nil error", func(t *testing.T) {
		e := NewEncoder()
		bytes, err := e.EncodeReport(nil)
		if err != nil {
			t.Fatalf("unexpected error: %s", err.Error())
		}
		if bytes != nil {
			t.Fatalf("unexpected bytes")
		}
	})

	t.Run("encoding a single upkeep result with missing FastGasGwei returns an error", func(t *testing.T) {
		e := NewEncoder()
		_, err := e.EncodeReport([]types.UpkeepResult{
			{
				Key:        chain.UpkeepKey("123|456"),
				LinkNative: big.NewInt(1),
			},
		})
		if err.Error() != "missing FastGasWei" {
			t.Fatalf("unexpected error: %s", err.Error())
		}
	})

	t.Run("encoding a single upkeep result with missing LinkNative returns an error", func(t *testing.T) {
		e := NewEncoder()
		_, err := e.EncodeReport([]types.UpkeepResult{
			{
				Key:        chain.UpkeepKey("123|456"),
				FastGasWei: big.NewInt(1),
			},
		})
		if err.Error() != "missing LinkNative" {
			t.Fatalf("unexpected error: %s", err.Error())
		}
	})

	t.Run("encoding a single upkeep result with a malformed upkeep key returns an error", func(t *testing.T) {
		e := NewEncoder()
		_, err := e.EncodeReport([]types.UpkeepResult{
			{
				Key:        chain.UpkeepKey("123"),
				FastGasWei: big.NewInt(1),
				LinkNative: big.NewInt(2),
			},
		})
		if err.Error() != "upkeep key not parsable: missing data in upkeep key: report encoding error" {
			t.Fatalf("unexpected error: %s", err.Error())
		}
	})

	t.Run("encoding a single upkeep result with an upkeep key containing a malformed upkeep identifier returns an error", func(t *testing.T) {
		e := NewEncoder()
		_, err := e.EncodeReport([]types.UpkeepResult{
			{
				Key:        chain.UpkeepKey("123|a"),
				FastGasWei: big.NewInt(1),
				LinkNative: big.NewInt(2),
			},
		})
		if err.Error() != "upkeep key not parsable" {
			t.Fatalf("unexpected error: %s", err.Error())
		}
	})

	t.Run("encoding a single upkeep result fails on report packing and returns an error", func(t *testing.T) {
		e := NewEncoder()
		e.packer = &mockPacker{
			PackFn: func(args ...interface{}) ([]byte, error) {
				return nil, errors.New("unable to pack")
			},
		}
		_, err := e.EncodeReport([]types.UpkeepResult{
			{
				Key:        chain.UpkeepKey("123|456"),
				FastGasWei: big.NewInt(1),
				LinkNative: big.NewInt(2),
			},
		})
		if err.Error() != "unable to pack: failed to pack report data" {
			t.Fatalf("unexpected error: %s", err.Error())
		}
	})

	t.Run("successfully encodes multiple upkeep results", func(t *testing.T) {
		e := NewEncoder()
		reportBytes, err := e.EncodeReport([]types.UpkeepResult{
			{
				Key:              chain.UpkeepKey("123|456"),
				FastGasWei:       big.NewInt(1),
				LinkNative:       big.NewInt(1),
				CheckBlockNumber: uint32(123),
			},
			{
				Key:              chain.UpkeepKey("124|456"),
				FastGasWei:       big.NewInt(2),
				LinkNative:       big.NewInt(3),
				CheckBlockNumber: uint32(124),
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %s", err.Error())
		}
		if len(reportBytes) == 0 {
			t.Fatalf("unexpected bytes")
		}
	})
}

func TestEncoder_KeysFromReport(t *testing.T) {
	t.Run("fails to unpack and returns an error", func(t *testing.T) {
		e := NewEncoder()
		key := chain.UpkeepKey("123|456")

		reportBytes, err := encodeReport(e, key)
		if err != nil {
			t.Fatalf("error encoding report: %s", err.Error())
		}

		e.packer = &mockPacker{
			UnpackIntoMapFn: func(v map[string]interface{}, data []byte) error {
				return errors.New("unable to unpack into map")
			},
		}
		keys, err := e.KeysFromReport(reportBytes)
		if err.Error() != "unable to unpack into map" {
			t.Fatalf("unexpected error: %s", err.Error())
		}
		if len(keys) != 0 {
			t.Fatalf("unexpected key count: %d", len(keys))
		}
	})

	t.Run("missing report key returns an error", func(t *testing.T) {
		e := NewEncoder()
		key := chain.UpkeepKey("123|456")

		reportBytes, err := encodeReport(e, key)
		if err != nil {
			t.Fatalf("error encoding report: %s", err.Error())
		}

		e.reportKeys = []string{"bulbasaur", "charmander", "squirtle", "pikachu"}
		keys, err := e.KeysFromReport(reportBytes)
		if err.Error() != "decoding error: bulbasaur missing from struct" {
			t.Fatalf("unexpected error: %s", err.Error())
		}
		if len(keys) != 0 {
			t.Fatalf("unexpected key count: %d", len(keys))
		}
	})

	t.Run("upkeep ids of incorrect type in report returns an error", func(t *testing.T) {
		e := NewEncoder()
		key := chain.UpkeepKey("123|456")

		reportBytes, err := encodeReport(e, key)
		if err != nil {
			t.Fatalf("error encoding report: %s", err.Error())
		}

		e.packer = &mockPacker{
			UnpackIntoMapFn: func(v map[string]interface{}, data []byte) error {
				v["upkeepIds"] = "dog"
				v["fastGasWei"] = big.NewInt(1)
				v["linkNative"] = big.NewInt(1)
				v["wrappedPerformDatas"] = []byte{}
				return nil
			},
		}
		keys, err := e.KeysFromReport(reportBytes)
		if err.Error() != "upkeep ids of incorrect type in report" {
			t.Fatalf("unexpected error: %s", err.Error())
		}
		if len(keys) != 0 {
			t.Fatalf("unexpected key count: %d", len(keys))
		}
	})

	t.Run("unable to read wrappedPerformedDatas returns an error", func(t *testing.T) {
		e := NewEncoder()
		key := chain.UpkeepKey("123|456")

		reportBytes, err := encodeReport(e, key)
		if err != nil {
			t.Fatalf("error encoding report: %s", err.Error())
		}

		e.packer = &mockPacker{
			UnpackIntoMapFn: func(v map[string]interface{}, data []byte) error {
				v["upkeepIds"] = []*big.Int{big.NewInt(1)}
				v["fastGasWei"] = big.NewInt(1)
				v["linkNative"] = big.NewInt(1)
				v["wrappedPerformDatas"] = []byte{}
				return nil
			},
		}
		keys, err := e.KeysFromReport(reportBytes)
		if err.Error() != "unable to read wrappedPerformDatas" {
			t.Fatalf("unexpected error: %s", err.Error())
		}
		if len(keys) != 0 {
			t.Fatalf("unexpected key count: %d", len(keys))
		}
	})

	t.Run("upkeep ids and performs of mismatched length returns an error", func(t *testing.T) {
		e := NewEncoder()
		key := chain.UpkeepKey("123|456")

		reportBytes, err := encodeReport(e, key)
		if err != nil {
			t.Fatalf("error encoding report: %s", err.Error())
		}

		e.packer = &mockPacker{
			UnpackIntoMapFn: func(v map[string]interface{}, data []byte) error {
				v["upkeepIds"] = []*big.Int{big.NewInt(1)}
				v["fastGasWei"] = big.NewInt(1)
				v["linkNative"] = big.NewInt(1)
				v["wrappedPerformDatas"] = []struct {
					CheckBlockNumber uint32   `json:"checkBlockNumber"`
					CheckBlockhash   [32]byte `json:"checkBlockhash"`
					PerformData      []byte   `json:"performData"`
				}{}
				return nil
			},
		}
		keys, err := e.KeysFromReport(reportBytes)
		if err.Error() != "upkeep ids and performs should have matching length" {
			t.Fatalf("unexpected error: %s", err.Error())
		}
		if len(keys) != 0 {
			t.Fatalf("unexpected key count: %d", len(keys))
		}
	})

	t.Run("successfully encodes a report and then reads the keys back", func(t *testing.T) {
		e := NewEncoder()
		key := chain.UpkeepKey("123|456")

		reportBytes, err := encodeReport(e, key)
		if err != nil {
			t.Fatalf("error encoding report: %s", err.Error())
		}

		keys, err := e.KeysFromReport(reportBytes)
		if err != nil {
			t.Fatalf("unexpected error: %s", err.Error())
		}
		if len(keys) != 1 {
			t.Fatalf("unexpected key count: %d", len(keys))
		}
		if !reflect.DeepEqual(keys[0], key) {
			t.Fatalf("unexpected key: %+s", keys[0])
		}
	})
}

func encodeReport(e *encoder, key chain.UpkeepKey) ([]byte, error) {
	reportBytes, err := e.EncodeReport([]types.UpkeepResult{
		{
			Key:              key,
			FastGasWei:       big.NewInt(1),
			LinkNative:       big.NewInt(1),
			CheckBlockNumber: uint32(123),
		},
	})
	if err != nil {
		return nil, err
	}
	if len(reportBytes) == 0 {
		return nil, fmt.Errorf("unexpected bytes")
	}
	return reportBytes, nil
}

type mockPacker struct {
	PackFn          func(args ...interface{}) ([]byte, error)
	UnpackIntoMapFn func(v map[string]interface{}, data []byte) error
}

func (p *mockPacker) Pack(args ...interface{}) ([]byte, error) {
	return p.PackFn(args...)
}

func (p *mockPacker) UnpackIntoMap(v map[string]interface{}, data []byte) error {
	return p.UnpackIntoMapFn(v, data)
}
