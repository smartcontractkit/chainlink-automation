package malicious

import (
	"context"
	"encoding/json"

	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

// ObservationParseError produces an encoded json object that does not properly
// parse into a valid observation.
func ObservationParseError(_ context.Context, _ []byte, _ error) (string, []byte, error) {
	type internal struct {
		D string
		E uint
	}

	type external struct {
		A int
		B string
		C internal
	}

	b, err := json.Marshal(external{5, "string", internal{"", 8}})
	return "Invalid Observation Structure", b, err
}

// NilBytesObservation returns a nil bytes observation
func NilBytesObservation(_ context.Context, _ []byte, _ error) (string, []byte, error) {
	return "Nil Bytes Observation", nil, nil
}

// EmptyBytesObservation returns a nil bytes observation
func EmptyBytesObservation(_ context.Context, _ []byte, _ error) (string, []byte, error) {
	return "Empty Bytes Observation", nil, nil
}

type UpkeepQuerant func(context.Context) (types.BlockKey, types.UpkeepResults, error)

// ObservationExtraFields produces an encoded json object that does not properly
// parse into a valid observation.
func ObservationExtraFields(ctx context.Context, original []byte, _ error) (string, []byte, error) {
	name := "Observation Extra Fields"

	type bulkyObservation struct {
		BlockKey          chain.BlockKey           `json:"1"`
		UpkeepIdentifiers []types.UpkeepIdentifier `json:"2"`
		OtherData         []int                    `json:"3"`
	}

	var ob chain.UpkeepObservation
	if err := json.Unmarshal(original, &ob); err != nil {
		return name, nil, err
	}

	observation := bulkyObservation{
		BlockKey:          ob.BlockKey,
		UpkeepIdentifiers: ob.UpkeepIdentifiers,
		OtherData:         []int{1, 2, 3, 4, 5, 6, 7},
	}

	b, err := json.Marshal(observation)
	return name, b, err
}

func InvalidObservationBlockKeyError(ctx context.Context, original []byte, _ error) (string, []byte, error) {
	name := "Invalid Observation Block Key"

	var ob chain.UpkeepObservation
	if err := json.Unmarshal(original, &ob); err != nil {
		return name, nil, err
	}

	// The max uint64 value is 18446744073709551615.
	// Just incrementing the value here.
	ob.BlockKey = "18446744073709551616"

	badObservation, err := json.Marshal(ob)
	if err != nil {
		return name, nil, err
	}

	return name, badObservation, nil
}

func InvalidObservationUpkeepKeyError(ctx context.Context, original []byte, _ error) (string, []byte, error) {
	name := "Invalid Observation Upkeep Key"

	var ob chain.UpkeepObservation
	if err := json.Unmarshal(original, &ob); err != nil {
		return name, nil, err
	}

	for i := range ob.UpkeepIdentifiers {
		// The max uint64 value is 18446744073709551615.
		// Just incrementing the value here.
		ob.UpkeepIdentifiers[i] = types.UpkeepIdentifier("18446744073709551616")
	}

	badObservation, err := json.Marshal(ob)
	if err != nil {
		return name, nil, err
	}

	return name, badObservation, nil
}
