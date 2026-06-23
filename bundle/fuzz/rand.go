package fuzz

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

var words = []string{
	"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel",
	"india", "juliet", "kilo", "lima", "mike", "november", "oscar", "papa",
}

// newRNG returns a deterministic RNG for the given seed, so any job the fuzzer
// flags can be regenerated from the printed seed alone.
func newRNG(seed int64) *rand.Rand {
	return rand.New(rand.NewPCG(uint64(seed), 0))
}

// chance returns true with probability p (0..1).
func chance(rng *rand.Rand, p float64) bool {
	return rng.Float64() < p
}

// oneOf returns a random element of s. s must be non-empty.
func oneOf[T any](rng *rand.Rand, s []T) T {
	return s[rng.IntN(len(s))]
}

func randWord(rng *rand.Rand) string {
	return oneOf(rng, words)
}

// randName returns a deterministic-but-varied identifier with the given prefix,
// e.g. "job_alpha_4271".
func randName(rng *rand.Rand, prefix string) string {
	return fmt.Sprintf("%s_%s_%d", prefix, randWord(rng), rng.IntN(10000))
}

func randSentence(rng *rand.Rand) string {
	n := rng.IntN(4) + 2
	parts := make([]string, 0, n)
	for range n {
		parts = append(parts, randWord(rng))
	}
	return strings.Join(parts, " ")
}
