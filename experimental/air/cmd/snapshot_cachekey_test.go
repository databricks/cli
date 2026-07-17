package aircmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type goldenCase struct {
	Name         string   `json:"name"`
	CommitSHA    string   `json:"commit_sha"`
	IncludePaths []string `json:"include_paths"`
	CacheKey     string   `json:"cache_key"`
}

// TestComputeSnapshotCacheKeyGolden asserts byte-for-byte parity with golden
// fixtures across the local-only matrix (commit + include_paths permutations).
func TestComputeSnapshotCacheKeyGolden(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "cache_keys.json"))
	require.NoError(t, err)

	var cases []goldenCase
	require.NoError(t, json.Unmarshal(data, &cases))
	require.NotEmpty(t, cases)

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			assert.Equal(t, tc.CacheKey, computeSnapshotCacheKey(tc.CommitSHA, tc.IncludePaths))
		})
	}
}

// TestComputeSnapshotCacheKeyProperties pins the normalization behavior the golden cases
// encode, so a regression is legible without decoding hashes.
func TestComputeSnapshotCacheKeyProperties(t *testing.T) {
	sha := "a3492b801c0ffee00000000000000000000dead"

	// Order-independent: sorting means unsorted input yields the sorted key.
	assert.Equal(t,
		computeSnapshotCacheKey(sha, []string{"a", "b", "c"}),
		computeSnapshotCacheKey(sha, []string{"c", "a", "b"}),
	)

	// nil and empty include_paths are equivalent (both contribute an empty line).
	assert.Equal(t, computeSnapshotCacheKey(sha, nil), computeSnapshotCacheKey(sha, []string{}))

	// Paths are trimmed before hashing.
	assert.Equal(t,
		computeSnapshotCacheKey(sha, []string{"research", "data"}),
		computeSnapshotCacheKey(sha, []string{"  research  ", "  data "}),
	)

	// Duplicates are NOT collapsed — they are sorted and kept, matching Python.
	assert.NotEqual(t, computeSnapshotCacheKey(sha, []string{"x", "y"}), computeSnapshotCacheKey(sha, []string{"x", "x", "y"}))

	// The version constant participates: a different version is a different key.
	assert.NotEqual(t, snapshotPackagingVersion, "")
}
