package v1

import (
	"errors"
	"reflect"
	"testing"

	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func TestUpkeepProvider_MakeUpkeepKey(t *testing.T) {
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
			e := upkeepProvider{}
			upkeepKey := e.MakeUpkeepKey(tc.blockKey, tc.upkeepIdentifier)
			if !reflect.DeepEqual(upkeepKey, tc.wantUpkeepKey) {
				t.Fatalf("unexpected upkeep key")
			}
		})
	}
}

func TestUpkeepProvider_SplitUpkeepKey(t *testing.T) {
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
			e := upkeepProvider{}
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
