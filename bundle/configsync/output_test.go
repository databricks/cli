package configsync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveFiles_Success(t *testing.T) {
	ctx := t.Context()

	tmpDir := t.TempDir()

	yamlPath := filepath.Join(tmpDir, "subdir", "databricks.yml")
	modifiedContent := `resources:
  jobs:
    test_job:
      name: "Updated Job"
      timeout_seconds: 7200
`

	files := []FileChange{
		{
			Path:            yamlPath,
			OriginalContent: "original content",
			ModifiedContent: modifiedContent,
		},
	}

	err := SaveFiles(ctx, &bundle.Bundle{}, files)
	require.NoError(t, err)

	_, err = os.Stat(yamlPath)
	require.NoError(t, err)

	content, err := os.ReadFile(yamlPath)
	require.NoError(t, err)
	assert.Equal(t, modifiedContent, string(content))

	_, err = os.Stat(filepath.Dir(yamlPath))
	require.NoError(t, err)
}

func TestSaveFiles_MultipleFiles(t *testing.T) {
	ctx := t.Context()

	tmpDir := t.TempDir()

	file1Path := filepath.Join(tmpDir, "file1.yml")
	file2Path := filepath.Join(tmpDir, "subdir", "file2.yml")
	content1 := "content for file 1"
	content2 := "content for file 2"

	files := []FileChange{
		{
			Path:            file1Path,
			OriginalContent: "original 1",
			ModifiedContent: content1,
		},
		{
			Path:            file2Path,
			OriginalContent: "original 2",
			ModifiedContent: content2,
		},
	}

	err := SaveFiles(ctx, &bundle.Bundle{}, files)
	require.NoError(t, err)

	content, err := os.ReadFile(file1Path)
	require.NoError(t, err)
	assert.Equal(t, content1, string(content))

	content, err = os.ReadFile(file2Path)
	require.NoError(t, err)
	assert.Equal(t, content2, string(content))
}

func TestSaveFiles_EmptyList(t *testing.T) {
	ctx := t.Context()

	err := SaveFiles(ctx, &bundle.Bundle{}, []FileChange{})
	require.NoError(t, err)
}

func TestWriteResultTextReportsSkippedResources(t *testing.T) {
	skipped := []SkippedChange{
		{ResourceKey: "resources.jobs.a", Reason: SkipReasonClusterPolicyNotFound, PolicyID: "p1"},
		{ResourceKey: "resources.jobs.b", Reason: SkipReasonClusterPolicyPermissionDenied, PolicyID: "p2", Error: "denied"},
		{ResourceKey: "resources.jobs.c", Reason: SkipReasonClusterPolicyReadFailed, PolicyID: "p3", Error: "boom"},
	}
	var out strings.Builder
	require.NoError(t, WriteResult(&out, false, &Stats{}, nil, nil, skipped, nil))
	assert.Equal(t, `Sync is incomplete: some remote changes were skipped.
Left resources.jobs.a unchanged: cluster policy p1 was not found.
Left resources.jobs.b unchanged: no permission to read cluster policy p2.
Left resources.jobs.c unchanged: cluster policy p3 could not be read: boom
No changes could be synced.
`, out.String())

	out.Reset()
	require.NoError(t, WriteResult(&out, true, &Stats{}, nil, nil, skipped[:1], nil))
	assert.Contains(t, out.String(), `"status": "partial"`)
	assert.Contains(t, out.String(), `"reason": "cluster_policy_not_found"`)
	assert.NotContains(t, out.String(), `"error"`)
}
