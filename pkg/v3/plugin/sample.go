package plugin

import (
	"log"
	"math/rand"
	"strings"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
)

type samples struct {
	limit        int
	allValues    []ocr2keepers.UpkeepIdentifier
	randomSource [16]byte
	logger       *log.Logger
}

func newSamples(limit int, rSrc [16]byte, logger *log.Logger) *samples {
	return &samples{
		limit:        limit,
		allValues:    []ocr2keepers.UpkeepIdentifier{},
		randomSource: rSrc,
		logger:       logger,
	}
}

func (s *samples) add(observation ocr2keepersv3.AutomationObservation) {
	rawValues, ok := observation.Metadata[ocr2keepersv3.SampleProposalObservationKey]
	if !ok {
		return
	}

	samples, ok := rawValues.([]ocr2keepers.UpkeepIdentifier)
	if !ok {
		return
	}

	s.allValues = append(s.allValues, samples...)
}

func (s *samples) set(outcome *ocr2keepersv3.AutomationOutcome) {
	final := dedupeShuffleObservations(s.allValues, s.randomSource)

	if len(final) > s.limit {
		final = final[:s.limit]
	}

	printed := []string{}
	for _, id := range final {
		printed = append(printed, string(id))
	}

	s.logger.Printf("%d samples agreed on: '%s'", len(final), strings.Join(printed, "','"))

	outcome.Metadata[ocr2keepersv3.CoordinatedSamplesProposalKey] = final
}

func dedupeShuffleObservations(upkeepIds []ocr2keepers.UpkeepIdentifier, keyRandSource [16]byte) []ocr2keepers.UpkeepIdentifier {
	uniqueKeys := dedupe(upkeepIds)

	rand.New(util.NewKeyedCryptoRandSource(keyRandSource)).Shuffle(len(uniqueKeys), func(i, j int) {
		uniqueKeys[i], uniqueKeys[j] = uniqueKeys[j], uniqueKeys[i]
	})

	return uniqueKeys
}

func dedupe(inputs []ocr2keepers.UpkeepIdentifier) []ocr2keepers.UpkeepIdentifier {
	output := make([]ocr2keepers.UpkeepIdentifier, 0, len(inputs))
	matched := make(map[string]struct{})

	for _, input := range inputs {
		key := string(input)
		_, ok := matched[key]
		if !ok {
			matched[key] = struct{}{}
			output = append(output, input)
		}
	}

	return output
}
