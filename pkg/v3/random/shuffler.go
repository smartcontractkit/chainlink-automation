package random

import "math/rand"

type Shuffler[T any] struct {
	Source rand.Source
}

func (s Shuffler[T]) Shuffle(a []T) []T {
	r := rand.New(s.Source)
	r.Shuffle(len(a), func(i, j int) {
		a[i], a[j] = a[j], a[i]
	})
	return a
}

// an optimized version of ShuffleString, where the shuffler is created once and reused
func NewStringShuffler(rSrc [16]byte) func(string) string {
	r := rand.New(NewKeyedCryptoRandSource(rSrc))
	return func(s string) string {
		shuffled := []rune(s)
		r.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})
		return string(shuffled)
	}
}

func ShuffleString(s string, rSrc [16]byte) string {
	shuffled := []rune(s)
	rand.New(NewKeyedCryptoRandSource(rSrc)).Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return string(shuffled)
}
