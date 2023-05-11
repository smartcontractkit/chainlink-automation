package ocr2keepers

import (
	"math/rand"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
)

func filterDedupeShuffleObservations(upkeepKeys [][]UpkeepKey, keyRandSource [16]byte, filters ...func(UpkeepKey) bool) ([]UpkeepKey, error) {
	uniqueKeys, err := filterAndDedupe(upkeepKeys, filters...)
	if err != nil {
		return nil, err
	}

	rand.New(util.NewKeyedCryptoRandSource(keyRandSource)).Shuffle(len(uniqueKeys), func(i, j int) {
		uniqueKeys[i], uniqueKeys[j] = uniqueKeys[j], uniqueKeys[i]
	})

	return uniqueKeys, nil
}

func filterAndDedupe(inputs [][]UpkeepKey, filters ...func(UpkeepKey) bool) ([]UpkeepKey, error) {
	var max int
	for _, input := range inputs {
		max += len(input)
	}

	output := make([]UpkeepKey, 0, max)
	matched := make(map[string]struct{})
	for _, input := range inputs {
		for _, val := range input {
			add := true
			for _, filter := range filters {
				if !filter(val) {
					add = false
					break
				}
			}

			if !add {
				continue
			}

			key := string(val)
			_, ok := matched[key]
			if !ok {
				matched[key] = struct{}{}
				output = append(output, val)
			}
		}
	}

	return output, nil
}

func shuffleObservations(upkeepIdentifiers []UpkeepIdentifier, source [16]byte) []UpkeepIdentifier {
	rand.New(util.NewKeyedCryptoRandSource(source)).Shuffle(len(upkeepIdentifiers), func(i, j int) {
		upkeepIdentifiers[i], upkeepIdentifiers[j] = upkeepIdentifiers[j], upkeepIdentifiers[i]
	})

	return upkeepIdentifiers
}
