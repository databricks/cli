package sandbox

import (
	"fmt"
	"math/rand/v2"
)

// Keep generated IDs easy to type while satisfying the public SDK's resource
// ID constraints. Display labels supplied with --name remain independent.
func generateSandboxID() string {
	adjectives := [...]string{
		"agile", "amber", "ample", "bold", "brave", "bright", "calm", "careful",
		"cheerful", "clear", "clever", "cool", "cosmic", "crisp", "curious", "daring",
		"eager", "easy", "fair", "fancy", "fast", "firm", "fluffy", "fresh",
		"friendly", "gentle", "glad", "golden", "grand", "green", "happy", "helpful",
		"honest", "humble", "jolly", "keen", "kind", "lively", "lucky", "mellow",
		"merry", "mighty", "neat", "nimble", "noble", "patient", "peaceful", "playful",
		"polite", "proud", "quick", "quiet", "ready", "rosy", "shiny", "sleepy",
		"smart", "smooth", "steady", "sunny", "swift", "tidy", "warm", "wise",
	}
	nouns := [...]string{
		"alpaca", "badger", "bear", "beaver", "bison", "bobcat", "buffalo", "camel",
		"canary", "cheetah", "chicken", "cobra", "cougar", "coyote", "crane", "deer",
		"dolphin", "dove", "dragon", "duck", "eagle", "elephant", "falcon", "ferret",
		"finch", "flamingo", "fox", "frog", "gazelle", "gecko", "giraffe", "goose",
		"hamster", "hawk", "heron", "horse", "ibis", "jaguar", "koala", "lemur",
		"leopard", "lion", "llama", "lynx", "magpie", "meerkat", "moose", "otter",
		"owl", "panda", "parrot", "pelican", "penguin", "puma", "quail", "rabbit",
		"raven", "robin", "seal", "sparrow", "tiger", "turtle", "whale", "zebra",
	}
	return fmt.Sprintf("%s-%s-%d", adjectives[rand.IntN(len(adjectives))], nouns[rand.IntN(len(nouns))], 1000+rand.IntN(9000))
}
