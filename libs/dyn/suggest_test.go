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
	tests := []struct {
		name string
		keys []string
		typo string
		want []string
	}{
		{
			// Keys within distance 2 are returned ordered by increasing
			// distance; ties keep the map's insertion order.
			name: "ordered by distance",
			keys: []string{"host", "hosts", "token", "auth_type"},
			typo: "host",
			want: []string{"host", "hosts"},
		},
		{
			name: "no key close enough",
			keys: []string{"host", "hosts", "token", "auth_type"},
			typo: "completely_different",
			want: []string{},
		},
		{
			// Distance-2 substitutions and insertions are both included.
			name: "distance two included",
			keys: []string{"profile", "prfile", "prof"},
			typo: "prfil",
			want: []string{"prfile", "profile"},
		},
		{
			name: "empty map",
			keys: nil,
			typo: "anything",
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, suggestKeys(newSuggestMapping(tt.keys...), tt.typo))
		})
	}
}

func TestDidYouMean(t *testing.T) {
	tests := []struct {
		name        string
		suggestions []string
		want        string
	}{
		{"nil", nil, ""},
		{"empty", []string{}, ""},
		{"single", []string{"host"}, `, did you mean "host"?`},
		{"multiple", []string{"host", "hosts"}, `, did you mean one of: "host", "hosts"?`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, didYouMean(tt.suggestions))
		})
	}
}

func TestDidYouMeanSuffix(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "no such key with suggestions",
			err:  noSuchKeyError{p: NewPath(Key("hst")), suggestions: []string{"host"}},
			want: `, did you mean "host"?`,
		},
		{
			name: "no such key without suggestions",
			err:  noSuchKeyError{p: NewPath(Key("xyz"))},
			want: "",
		},
		{
			name: "other error type",
			err:  errors.New("some other error"),
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, DidYouMeanSuffix(tt.err))
		})
	}
}
