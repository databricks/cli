package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFingerprintDeterministic(t *testing.T) {
	// Two independently constructed but equal inputs must hash identically, so
	// the fingerprint depends only on content, not on slice identity or run.
	first := []TreeEntry{
		{Path: "a.go", Blob: "aaa"},
		{Path: "b.go", Blob: "bbb"},
	}
	second := []TreeEntry{
		{Path: "a.go", Blob: "aaa"},
		{Path: "b.go", Blob: "bbb"},
	}
	assert.Equal(t, Fingerprint(first), Fingerprint(second))
}

func TestFingerprintOrderIndependent(t *testing.T) {
	forward := []TreeEntry{
		{Path: "a.go", Blob: "aaa"},
		{Path: "b.go", Blob: "bbb"},
	}
	reversed := []TreeEntry{
		{Path: "b.go", Blob: "bbb"},
		{Path: "a.go", Blob: "aaa"},
	}
	assert.Equal(t, Fingerprint(forward), Fingerprint(reversed),
		"listing order must not change the fingerprint")
}

func TestFingerprintContentSensitive(t *testing.T) {
	before := []TreeEntry{{Path: "a.go", Blob: "aaa"}}
	after := []TreeEntry{{Path: "a.go", Blob: "aab"}}
	assert.NotEqual(t, Fingerprint(before), Fingerprint(after),
		"a blob change must change the fingerprint")
}

func TestFingerprintRenameSensitive(t *testing.T) {
	before := []TreeEntry{{Path: "a.go", Blob: "aaa"}}
	after := []TreeEntry{{Path: "renamed.go", Blob: "aaa"}}
	assert.NotEqual(t, Fingerprint(before), Fingerprint(after),
		"a rename (same content, new path) must change the fingerprint")
}

func TestFingerprintAddedFileSensitive(t *testing.T) {
	before := []TreeEntry{{Path: "a.go", Blob: "aaa"}}
	after := []TreeEntry{
		{Path: "a.go", Blob: "aaa"},
		{Path: "b.go", Blob: "bbb"},
	}
	assert.NotEqual(t, Fingerprint(before), Fingerprint(after),
		"adding a file must change the fingerprint")
}

// TestFingerprintNoConcatenationCollision guards the length-prefixed encoding:
// path/blob boundaries must be unambiguous so no two distinct entry sets whose
// naive concatenation matches can collide.
func TestFingerprintNoConcatenationCollision(t *testing.T) {
	a := []TreeEntry{{Path: "ab", Blob: "c"}}
	b := []TreeEntry{{Path: "a", Blob: "bc"}}
	assert.NotEqual(t, Fingerprint(a), Fingerprint(b))
}

func TestFingerprintEmptyStable(t *testing.T) {
	// The empty set has a well-defined hash; callers (not this function) decide
	// whether "nothing matched" is meaningful. Just assert it is stable.
	assert.Equal(t, Fingerprint(nil), Fingerprint([]TreeEntry{}))
}

func TestParseLsTree(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want []TreeEntry
	}{
		{
			name: "basic",
			out:  "aaa a.go\nbbb dir/b.go\n",
			want: []TreeEntry{
				{Path: "a.go", Blob: "aaa"},
				{Path: "dir/b.go", Blob: "bbb"},
			},
		},
		{
			name: "path_with_spaces",
			out:  "aaa dir/file with spaces.txt\n",
			want: []TreeEntry{
				{Path: "dir/file with spaces.txt", Blob: "aaa"},
			},
		},
		{
			name: "trailing_and_blank_lines_ignored",
			out:  "aaa a.go\n\nbbb b.go\n\n",
			want: []TreeEntry{
				{Path: "a.go", Blob: "aaa"},
				{Path: "b.go", Blob: "bbb"},
			},
		},
		{
			name: "empty",
			out:  "",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseLsTree(tt.out))
		})
	}
}

// TestParseThenFingerprintMatchesDirect ties the two halves together: parsing
// ls-tree output and fingerprinting it yields the same hash as fingerprinting
// the entries directly.
func TestParseThenFingerprintMatchesDirect(t *testing.T) {
	entries := []TreeEntry{
		{Path: "a.go", Blob: "aaa"},
		{Path: "b.go", Blob: "bbb"},
	}
	parsed := ParseLsTree("aaa a.go\nbbb b.go\n")
	assert.Equal(t, Fingerprint(entries), Fingerprint(parsed))
}
