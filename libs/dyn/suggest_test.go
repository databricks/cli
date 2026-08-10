package dyn

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a    string
		b    string
		want int
	}{
		{"", "", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"kitten", "sitting", 3},
		{"host", "hosts", 1},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, levenshteinDistance(tt.a, tt.b), "levenshteinDistance(%q, %q)", tt.a, tt.b)
	}
}

func newSuggestMapping(keys ...string) Mapping {
	var m Mapping
	for _, k := range keys {
		m.SetLoc(k, nil, V(k))
	}
	return m
}

func TestSuggestKeys(t *testing.T) {
	// Keys within distance 2 are returned ordered by increasing distance;
	// ties keep the map's insertion order.
	m := newSuggestMapping("host", "hosts", "token", "auth_type")
	assert.Equal(t, []string{"host", "hosts"}, suggestKeys(m, "host"))

	// No key is close enough.
	assert.Empty(t, suggestKeys(m, "completely_different"))

	// Distance-2 substitutions and insertions are both included.
	m = newSuggestMapping("profile", "prfile", "prof")
	assert.Equal(t, []string{"prfile", "profile"}, suggestKeys(m, "prfil"))

	// Empty map yields no suggestions.
	assert.Empty(t, suggestKeys(NewMapping(), "anything"))
}

func TestDidYouMean(t *testing.T) {
	assert.Empty(t, didYouMean(nil))
	assert.Empty(t, didYouMean([]string{}))
	assert.Equal(t, `, did you mean "host"?`, didYouMean([]string{"host"}))
	assert.Equal(t, `, did you mean one of: "host", "hosts"?`, didYouMean([]string{"host", "hosts"}))
}

func TestDidYouMeanSuffix(t *testing.T) {
	// A noSuchKeyError with suggestions produces the clause.
	err := noSuchKeyError{p: NewPath(Key("hst")), suggestions: []string{"host"}}
	assert.Equal(t, `, did you mean "host"?`, DidYouMeanSuffix(err))

	// A noSuchKeyError without suggestions produces nothing.
	err = noSuchKeyError{p: NewPath(Key("xyz"))}
	assert.Empty(t, DidYouMeanSuffix(err))

	// Any other error type produces nothing.
	assert.Empty(t, DidYouMeanSuffix(errors.New("some other error")))
}
