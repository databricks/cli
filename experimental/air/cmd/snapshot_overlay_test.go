package aircmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func overlayTestFile(t *testing.T, repo, rel string) snapshotFile {
	t.Helper()
	info, err := os.Lstat(filepath.Join(repo, filepath.FromSlash(rel)))
	require.NoError(t, err)
	linkTarget := ""
	if info.Mode()&os.ModeSymlink != 0 {
		linkTarget, err = os.Readlink(filepath.Join(repo, filepath.FromSlash(rel)))
		require.NoError(t, err)
	}
	return snapshotFile{
		rel:        filepath.FromSlash(rel),
		size:       info.Size(),
		modTime:    info.ModTime().UnixNano(),
		mode:       uint32(info.Mode()),
		linkTarget: linkTarget,
	}
}

func TestComputeSnapshotOverlayDiff(t *testing.T) {
	anchor := map[string]snapshotOverlayEntry{
		"same":    {Size: 1, ModTime: 1, Mode: 0o644},
		"mode":    {Size: 2, ModTime: 2, Mode: 0o644},
		"link":    {Size: 3, ModTime: 3, Mode: uint32(os.ModeSymlink | 0o777), LinkTarget: "old"},
		"deleted": {Size: 10, ModTime: 4, Mode: 0o644},
	}
	files := []snapshotFile{
		{rel: "same", size: 1, modTime: 1, mode: 0o644},
		{rel: "mode", size: 2, modTime: 2, mode: 0o755},
		{rel: "link", size: 3, modTime: 3, mode: uint32(os.ModeSymlink | 0o777), linkTarget: "new"},
		{rel: "added", size: 4, modTime: 5, mode: 0o600},
	}

	diff := computeSnapshotOverlayDiff(files, anchor)

	require.Len(t, diff.Changed, 3)
	assert.Equal(t, []string{"added", "link", "mode"}, []string{
		filepath.ToSlash(diff.Changed[0].rel),
		filepath.ToSlash(diff.Changed[1].rel),
		filepath.ToSlash(diff.Changed[2].rel),
	})
	assert.Equal(t, []string{"deleted"}, diff.Deleted)
	assert.Equal(t, int64(9), diff.ChangedBytes)
	assert.Equal(t, int64(10), diff.TotalBytes)
}

func TestSnapshotOverlayAnchorReason(t *testing.T) {
	assert.Empty(t, snapshotOverlayAnchorReason(snapshotOverlayDiff{ChangedBytes: 1, TotalBytes: 100}))
	assert.Equal(t, "changed-path-count", snapshotOverlayAnchorReason(snapshotOverlayDiff{
		Changed: make([]snapshotFile, snapshotOverlayMaxChanged+1),
	}))
	assert.Equal(t, "changed-bytes", snapshotOverlayAnchorReason(snapshotOverlayDiff{
		ChangedBytes: snapshotOverlayMaxBytes + 1,
		TotalBytes:   snapshotOverlayMaxBytes * 10,
	}))
	assert.Equal(t, "changed-byte-ratio", snapshotOverlayAnchorReason(snapshotOverlayDiff{
		ChangedBytes: 26,
		TotalBytes:   100,
	}))
}

func TestCreateSnapshotOverlay(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "changed.sh", "#!/bin/sh\necho changed\n")
	require.NoError(t, os.Chmod(filepath.Join(repo, "changed.sh"), 0o755))
	writeRepoFile(t, repo, "added.txt", "added")
	require.NoError(t, os.Symlink("added.txt", filepath.Join(repo, "current")))
	files := []snapshotFile{
		overlayTestFile(t, repo, "changed.sh"),
		overlayTestFile(t, repo, "added.txt"),
		overlayTestFile(t, repo, "current"),
	}
	state := &snapshotAnchorState{
		AnchorName:        "air_anchor_v1_aabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccdd.tar.gz",
		AnchorFingerprint: "aabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccdd",
	}
	out := filepath.Join(t.TempDir(), "overlay.tar.gz")
	require.NoError(t, createSnapshotOverlay(repo, filepath.Base(repo), state, "b"+state.AnchorFingerprint[1:], snapshotOverlayDiff{
		Changed: files,
		Deleted: []string{"deleted.txt"},
	}, out))

	archive, err := os.Open(out)
	require.NoError(t, err)
	defer archive.Close()
	gz, err := gzip.NewReader(archive)
	require.NoError(t, err)
	defer gz.Close()
	reader := tar.NewReader(gz)

	first, err := reader.Next()
	require.NoError(t, err)
	assert.Equal(t, snapshotOverlayDescriptorName, first.Name)
	descriptorBytes, err := io.ReadAll(reader)
	require.NoError(t, err)
	var descriptor snapshotOverlayDescriptor
	require.NoError(t, json.Unmarshal(descriptorBytes, &descriptor))
	assert.Equal(t, state.AnchorName, descriptor.Anchor)
	assert.Equal(t, filepath.Base(repo), descriptor.Component)

	entries := map[string]*tar.Header{}
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		copy := *header
		entries[header.Name] = &copy
	}
	assert.Contains(t, entries, snapshotOverlayDeletionsRoot+"/deleted.txt")
	assert.Equal(t, int64(0), entries[snapshotOverlayDeletionsRoot+"/deleted.txt"].Size)
	assert.Equal(t, int64(0o755), entries[filepath.Base(repo)+"/changed.sh"].Mode)
	assert.Equal(t, byte(tar.TypeSymlink), entries[filepath.Base(repo)+"/current"].Typeflag)
	assert.Equal(t, "added.txt", entries[filepath.Base(repo)+"/current"].Linkname)
}

func TestSnapshotAnchorStateIsBranchLocalAndStrict(t *testing.T) {
	repo := t.TempDir()
	config := filepath.Join(repo, "run.yaml")
	mainPath, err := snapshotOverlayStatePath(repo, config, []string{"src"}, "main")
	require.NoError(t, err)
	featurePath, err := snapshotOverlayStatePath(repo, config, []string{"src"}, "feature")
	require.NoError(t, err)
	assert.NotEqual(t, mainPath, featurePath)

	entries := map[string]snapshotOverlayEntry{"src/train.py": {Size: 1}}
	anchorFingerprint := snapshotOverlayFingerprint([]snapshotFile{{rel: filepath.FromSlash("src/train.py"), size: 1}})
	anchorName := "air_anchor_v1_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.tar.gz"
	state := snapshotAnchorState{
		Version:           snapshotOverlayVersion,
		Branch:            "main",
		CreatedAtUnix:     time.Now().Unix(),
		AnchorName:        anchorName,
		AnchorFingerprint: anchorFingerprint,
		Component:         filepath.Base(repo),
		Entries:           entries,
		Results:           map[string]string{anchorFingerprint: anchorName},
	}
	require.NoError(t, saveSnapshotAnchorState(mainPath, state))
	assert.NotNil(t, loadSnapshotAnchorState(mainPath, "main", filepath.Base(repo)))
	assert.Nil(t, loadSnapshotAnchorState(mainPath, "other", filepath.Base(repo)))

	require.NoError(t, os.WriteFile(mainPath, append([]byte(`{"unknown":true}`), '\n'), 0o600))
	assert.Nil(t, loadSnapshotAnchorState(mainPath, "main", filepath.Base(repo)))
}

func TestMaybeUploadSnapshotOverlayLifecycle(t *testing.T) {
	repo := t.TempDir()
	largePath := filepath.Join(repo, "large.bin")
	large, err := os.Create(largePath)
	require.NoError(t, err)
	require.NoError(t, large.Truncate(snapshotCacheMinBytes+1))
	require.NoError(t, large.Close())
	writeRepoFile(t, repo, "train.py", "print('one')")
	configPath := filepath.Join(repo, "run.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("test"), 0o600))

	remoteDir := t.TempDir()
	store, err := filer.NewLocalClient(remoteDir)
	require.NoError(t, err)
	plan := snapshotPlan{mode: modePlainTar, isGitRepo: false}
	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	statePath, err := snapshotOverlayStatePath(repo, configPath, nil, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(statePath)) })

	anchor, handled, err := maybeUploadSnapshotOverlay(
		t.Context(), store, "/Workspace/test/.internal", repo, configPath, plan, files,
		t.TempDir(), "", false,
	)
	require.NoError(t, err)
	require.True(t, handled)
	assert.Contains(t, filepath.Base(anchor.CodeSourcePath), "air_anchor_v1_")

	oldInfo, err := os.Stat(filepath.Join(repo, "train.py"))
	require.NoError(t, err)
	writeRepoFile(t, repo, "train.py", "print('two')")
	require.NoError(t, os.Chtimes(
		filepath.Join(repo, "train.py"), oldInfo.ModTime().Add(time.Second), oldInfo.ModTime().Add(time.Second),
	))
	files, err = snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	overlay, handled, err := maybeUploadSnapshotOverlay(
		t.Context(), store, "/Workspace/test/.internal", repo, configPath, plan, files,
		t.TempDir(), "", false,
	)
	require.NoError(t, err)
	require.True(t, handled)
	assert.Contains(t, filepath.Base(overlay.CodeSourcePath), "air_overlay_v1_")
	assert.NotEqual(t, anchor.CodeSourcePath, overlay.CodeSourcePath)

	reused, handled, err := maybeUploadSnapshotOverlay(
		t.Context(), store, "/Workspace/test/.internal", repo, configPath, plan, files,
		t.TempDir(), "", false,
	)
	require.NoError(t, err)
	require.True(t, handled)
	assert.Equal(t, overlay.CodeSourcePath, reused.CodeSourcePath)
}
