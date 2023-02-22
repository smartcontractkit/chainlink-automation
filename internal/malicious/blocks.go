package malicious

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"math/big"
	"strings"

	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

// SendVeryOldBlockNumber creates an observation with a block number that is
// far in the past
func SendVeryOldBlockNumber(_ context.Context, original []byte, err error) (string, []byte, error) {
	name := "Send Very Old Block Number"

	if err != nil {
		return name, nil, err
	}

	var ob chain.UpkeepObservation
	if err := json.Unmarshal(original, &ob); err != nil {
		return name, nil, err
	}

	bl, _ := ob.BlockKey.BigInt()
	ob.BlockKey = chain.BlockKey(new(big.Int).Sub(bl, big.NewInt(100)).String())

	b, err := json.Marshal(ob)
	return name, b, err
}

// SendVeryFutureBlockNumber creates an observation with a block number that is
// far in the future
func SendVeryFutureBlockNumber(_ context.Context, original []byte, err error) (string, []byte, error) {
	name := "Send Future Block Number"

	if err != nil {
		return name, nil, err
	}

	var ob chain.UpkeepObservation
	if err := json.Unmarshal(original, &ob); err != nil {
		return name, nil, err
	}

	bl, _ := ob.BlockKey.BigInt()
	ob.BlockKey = chain.BlockKey(new(big.Int).Add(bl, big.NewInt(100)).String())

	b, err := json.Marshal(ob)
	return name, b, err
}

// SendNegativeBlockNumber creates an observation with a block number less than
// zero
func SendNegativeBlockNumber(_ context.Context, original []byte, err error) (string, []byte, error) {
	name := "Send Negative Block Number"

	if err != nil {
		return name, nil, err
	}

	var ob chain.UpkeepObservation
	if err := json.Unmarshal(original, &ob); err != nil {
		return name, nil, err
	}

	ob.BlockKey = chain.BlockKey(big.NewInt(-1000).String())

	b, err := json.Marshal(ob)
	return name, b, err
}

// SendZeroBlockNumber creates an observation with a zero block number
func SendZeroBlockNumber(_ context.Context, original []byte, err error) (string, []byte, error) {
	name := "Send Zero Block Number"

	if err != nil {
		return name, nil, err
	}

	var ob chain.UpkeepObservation
	if err := json.Unmarshal(original, &ob); err != nil {
		return name, nil, err
	}

	ob.BlockKey = chain.BlockKey(big.NewInt(0).String())

	b, err := json.Marshal(ob)
	return name, b, err
}

// SendEmptyBlockValue produces an encoded json object that does not properly
// parse into a valid observation.
func SendEmptyBlockValue(ctx context.Context, original []byte, _ error) (string, []byte, error) {
	name := "Send Empty Block Value"

	type modifiedObservation struct {
		BlockKey          string                   `json:"1"`
		UpkeepIdentifiers []types.UpkeepIdentifier `json:"2"`
	}

	var ob chain.UpkeepObservation
	if err := json.Unmarshal(original, &ob); err != nil {
		return name, nil, err
	}

	observation := modifiedObservation{
		BlockKey:          "",
		UpkeepIdentifiers: ob.UpkeepIdentifiers,
	}

	b, err := json.Marshal(observation)
	return name, b, err
}

// SendVeryLargeBlockValue produces an encoded json object that does not properly
// parse into a valid observation.
func SendVeryLargeBlockValue(ctx context.Context, original []byte, _ error) (string, []byte, error) {
	name := "Send Very Large Block Value"

	type modifiedObservation struct {
		BlockKey          string                   `json:"1"`
		UpkeepIdentifiers []types.UpkeepIdentifier `json:"2"`
	}

	var ob chain.UpkeepObservation
	if err := json.Unmarshal(original, &ob); err != nil {
		return name, nil, err
	}

	keyStr, err := GenerateRandomASCIIString(1000)
	if err != nil {
		return name, nil, err
	}

	observation := modifiedObservation{
		BlockKey:          keyStr,
		UpkeepIdentifiers: ob.UpkeepIdentifiers,
	}

	b, err := json.Marshal(observation)
	return name, b, err
}

func GenerateRandomASCIIString(length int) (string, error) {
	result := strings.Builder{}
	for {
		if result.Len() >= length {
			return result.String(), nil
		}
		num, err := rand.Int(rand.Reader, big.NewInt(int64(127)))
		if err != nil {
			return "", err
		}
		n := num.Int64()
		// Make sure that the number/byte/letter is inside
		// the range of printable ASCII characters (excluding space and DEL)
		if n > 32 && n < 127 {
			result.WriteRune(rune(n))
		}
	}
}
