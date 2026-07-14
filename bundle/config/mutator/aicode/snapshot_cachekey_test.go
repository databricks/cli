package aicode

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComputeSnapshotCacheKeyGolden checks parity with the Python air CLI: the
// keys in testdata/cache_keys.json are produced by the Python implementation, so
// matching them proves the Go port hashes identical material.
func TestComputeSnapshotCacheKeyGolden(t *testing.T) {
	data, err := os.ReadFile("testdata/cache_keys.json")
	require.NoError(t, err)

	var cases []struct {
		Name         string   `json:"name"`
		CommitSHA    string   `json:"commit_sha"`
		IncludePaths []string `json:"include_paths"`
		CacheKey     string   `json:"cache_key"`
	}
	require.NoError(t, json.Unmarshal(data, &cases))
	require.NotEmpty(t, cases)

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			assert.Equal(t, tc.CacheKey, computeSnapshotCacheKey(tc.CommitSHA, tc.IncludePaths))
		})
	}
}
