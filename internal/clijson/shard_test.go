package clijson

import (
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// codegenDir holds the committed shards, relative to this package.
const codegenDir = "../../.codegen"

// Services are deliberately not in alphabetical order: ReadDir must restore the
// order from index.json, not from the file layout.
const testDoc = `{
  "schemas": {
    "jobs.Job": {"description": "A job.", "fields": {"settings": {"ref": "jobs.JobSettings"}}},
    "jobs.JobSettings": {"fields": {"cluster": {"ref": "compute.ClusterSpec"}}},
    "compute.ClusterSpec": {"description": "A <cluster> & more."}
  },
  "commands": {
    "services": [
      {"id": "jobs.Jobs", "name": "Jobs", "methods": [{"name": "Get"}]},
      {"id": "compute.Clusters", "name": "Clusters", "parent_service_id": "jobs.Jobs"}
    ],
    "workspace_docs_groups": [{"key": "jobs", "display_name": "Jobs"}],
    "account_docs_groups": []
  }
}`

func TestSplitReadDirRoundTrip(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Split([]byte(testDoc), dir))

	for _, f := range []string{
		"index.json",
		"schemas/jobs.json",
		"schemas/compute.json",
		"commands/jobs/Jobs.json",
		"commands/compute/Clusters.json",
	} {
		assert.FileExists(t, filepath.Join(dir, f))
	}

	got, err := ReadDir(dir)
	require.NoError(t, err)
	assert.JSONEq(t, testDoc, string(got))
}

func TestSplitReplacesStaleShards(t *testing.T) {
	dir := t.TempDir()
	stale := []string{
		filepath.Join(dir, "schemas", "removed.json"),
		filepath.Join(dir, "commands", "removed", "Gone.json"),
	}
	for _, path := range stale {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte("{}\n"), 0o644))
	}

	require.NoError(t, Split([]byte(testDoc), dir))

	for _, path := range stale {
		assert.NoFileExists(t, path)
	}
}

func TestSplitErrors(t *testing.T) {
	tests := []struct {
		name string
		doc  string
	}{
		{"unknown top-level block", `{"schemas": {}, "commands": {"services": []}, "extra": {}}`},
		{"missing services", `{"schemas": {}, "commands": {}}`},
		{"service id without package", `{"schemas": {}, "commands": {"services": [{"id": "Jobs"}]}}`},
		{"schema key without package", `{"schemas": {"Job": {}}, "commands": {"services": []}}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Error(t, Split([]byte(tc.doc), t.TempDir()))
		})
	}
}

// TestCommittedShardsAreCanonical re-splits the committed shards and requires
// the result to match them byte-for-byte, so a hand edit or a stale shard left
// behind fails here rather than surfacing as a silent drop in a generator.
func TestCommittedShardsAreCanonical(t *testing.T) {
	doc, err := ReadDir(codegenDir)
	require.NoError(t, err)

	dir := t.TempDir()
	require.NoError(t, Split(doc, dir))

	want := readShards(t, dir)
	got := readShards(t, codegenDir)
	require.ElementsMatch(t, slices.Collect(maps.Keys(want)), slices.Collect(maps.Keys(got)))
	for path, content := range want {
		assert.Equal(t, content, got[path], path)
	}
}

// readShards returns the contents of every shard under dir, keyed by its
// slash-separated path relative to dir.
func readShards(t *testing.T, dir string) map[string]string {
	shards := map[string]string{}
	index, err := os.ReadFile(filepath.Join(dir, indexFile))
	require.NoError(t, err)
	shards[indexFile] = string(index)

	for _, sub := range []string{schemasDir, commandsDir} {
		err := filepath.WalkDir(filepath.Join(dir, sub), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			shards[filepath.ToSlash(rel)] = string(b)
			return nil
		})
		require.NoError(t, err)
	}
	return shards
}
