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

func TestDidYouMeanReferences(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		reference string
		want      string
	}{
		{
			name:      "single suggestion",
			err:       noSuchKeyError{p: NewPath(Key("variables"), Key("hst")), suggestions: []string{"host"}},
			reference: "var.hst",
			want:      "\n\ndid you mean:\n  ${var.host}",
		},
		{
			name:      "multiple suggestions",
			err:       noSuchKeyError{p: NewPath(Key("variables"), Key("hst")), suggestions: []string{"host", "hosts"}},
			reference: "var.hst",
			want:      "\n\ndid you mean:\n  ${var.host}\n  ${var.hosts}",
		},
		{
			name:      "nested outer-key typo keeps suffix",
			err:       noSuchKeyError{p: NewPath(Key("variables"), Key("clustr")), suggestions: []string{"cluster"}},
			reference: "var.clustr.spark_version",
			want:      "\n\ndid you mean:\n  ${var.cluster.spark_version}",
		},
		{
			name:      "deep leaf typo keeps prefix",
			err:       noSuchKeyError{p: NewPath(Key("variables"), Key("cluster"), Key("value"), Key("spark_versio")), suggestions: []string{"spark_version"}},
			reference: "var.cluster.spark_versio",
			want:      "\n\ndid you mean:\n  ${var.cluster.spark_version}",
		},
		{
			name:      "index component preserved",
			err:       noSuchKeyError{p: NewPath(Key("variables"), Key("librariez")), suggestions: []string{"libraries"}},
			reference: "var.librariez[0].jar",
			want:      "\n\ndid you mean:\n  ${var.libraries[0].jar}",
		},
		{
			name:      "non-var prefix",
			err:       noSuchKeyError{p: NewPath(Key("workspace"), Key("stot_path")), suggestions: []string{"root_path", "state_path"}},
			reference: "workspace.stot_path",
			want:      "\n\ndid you mean:\n  ${workspace.root_path}\n  ${workspace.state_path}",
		},
		{
			name:      "no suggestions",
			err:       noSuchKeyError{p: NewPath(Key("xyz"))},
			reference: "var.xyz",
			want:      "",
		},
		{
			name:      "other error type",
			err:       errors.New("some other error"),
			reference: "var.xyz",
			want:      "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, DidYouMeanReferences(tt.err, tt.reference))
		})
	}
}

func TestReplaceKeyFallsBackWhenKeyAbsent(t *testing.T) {
	// The failed key is not present in the reference, so only the replacement
	// itself is returned rather than a spliced reference.
	assert.Equal(t, "host", replaceKey("var.foo", "missing", "host"))
}
