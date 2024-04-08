package random

import (
	"math/rand"

	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

type Shuffler struct {
	Source rand.Source
}

func (s Shuffler) Shuffle(a []common.UpkeepPayload) []common.UpkeepPayload {
	r := rand.New(s.Source)
	r.Shuffle(len(a), func(i, j int) {
		a[i], a[j] = a[j], a[i]
	})
	return a
}

func ShuffleString(s string, rSrc [16]byte) string {
	shuffled := []rune(s)
	rand.New(NewKeyedCryptoRandSource(rSrc)).Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return string(shuffled)
}
