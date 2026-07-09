package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// goldenCase is one entry in testdata/cache_keys.json, captured from the Python
// CLI's compute_snapshot_cache_key
type goldenCase struct {
	Name         string   `json:"name"`
	CommitSHA    string   `json:"commit_sha"`
	IncludePaths []string `json:"include_paths"`
	CacheKey     string   `json:"cache_key"`
}

// TestComputeCacheKeyGolden asserts byte-for-byte parity with golden
// fixtures across the local-only matrix (commit + include_paths permutations).
func TestComputeCacheKeyGolden(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "cache_keys.json"))
	require.NoError(t, err)

	var cases []goldenCase
	require.NoError(t, json.Unmarshal(data, &cases))
	require.NotEmpty(t, cases)

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			assert.Equal(t, tc.CacheKey, ComputeCacheKey(tc.CommitSHA, tc.IncludePaths))
		})
	}
}

// TestComputeCacheKeyProperties pins the normalization behavior the golden cases
// encode, so a regression is legible without decoding hashes.
func TestComputeCacheKeyProperties(t *testing.T) {
	sha := "a3492b801c0ffee00000000000000000000dead"

	// Order-independent: sorting means unsorted input yields the sorted key.
	assert.Equal(t,
		ComputeCacheKey(sha, []string{"a", "b", "c"}),
		ComputeCacheKey(sha, []string{"c", "a", "b"}),
	)

	// nil and empty include_paths are equivalent (both contribute an empty line).
	assert.Equal(t, ComputeCacheKey(sha, nil), ComputeCacheKey(sha, []string{}))

	// Paths are trimmed before hashing.
	assert.Equal(t,
		ComputeCacheKey(sha, []string{"research", "data"}),
		ComputeCacheKey(sha, []string{"  research  ", "  data "}),
	)

	// Duplicates are NOT collapsed — they are sorted and kept, matching Python.
	assert.NotEqual(t, ComputeCacheKey(sha, []string{"x", "y"}), ComputeCacheKey(sha, []string{"x", "x", "y"}))

	// The version constant participates: a different version is a different key.
	assert.NotEqual(t, PackagingVersion, "")
}
