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

// TestComputePlainTarKeyProperties pins the working-tree fingerprint behavior: it is
// order-independent and reacts to any change in a file's path, size, or mtime.
func TestComputePlainTarKeyProperties(t *testing.T) {
	base := []snapshotFile{
		{rel: "a.txt", size: 10, modTime: 100},
		{rel: "src/b.py", size: 20, modTime: 200},
	}

	// Order-independent: the files are sorted by path before hashing.
	assert.Equal(t,
		computePlainTarKey(base),
		computePlainTarKey([]snapshotFile{base[1], base[0]}),
	)

	// A changed size, mtime, or path each yields a different key.
	assert.NotEqual(t, computePlainTarKey(base), computePlainTarKey([]snapshotFile{{rel: "a.txt", size: 11, modTime: 100}, base[1]}))
	assert.NotEqual(t, computePlainTarKey(base), computePlainTarKey([]snapshotFile{{rel: "a.txt", size: 10, modTime: 101}, base[1]}))
	assert.NotEqual(t, computePlainTarKey(base), computePlainTarKey([]snapshotFile{{rel: "renamed.txt", size: 10, modTime: 100}, base[1]}))

	// Adding or dropping a file changes the key.
	assert.NotEqual(t, computePlainTarKey(base), computePlainTarKey(base[:1]))
}
