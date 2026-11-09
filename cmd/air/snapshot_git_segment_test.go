package aircmd

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustCreateGitArchiveSnapshot packages commitSHA with createGitArchiveSnapshot and fails the
// test on error.
func mustCreateGitArchiveSnapshot(t *testing.T, ctx context.Context, git gitRepo, commitSHA, outputTarball, directoryName string, includePaths []string, subtreePrefix string) {
	t.Helper()
	_, err := createGitArchiveSnapshot(ctx, git, commitSHA, outputTarball, directoryName, includePaths, subtreePrefix)
	require.NoError(t, err)
}

// commitFiles commits n files of size bytes each, spread over three directories.
func commitFiles(t *testing.T, n, size int) (string, string) {
	t.Helper()
	repo := newTestRepo(t)
	for i := range n {
		writeRepoFile(t, repo, fmt.Sprintf("pkg%d/file%d.py", i%3, i), strings.Repeat("x", size+i))
	}
	return repo, commitAll(t, repo, "init")
}

// A multi-member git_archive snapshot reads as one stream with pax_global_header first. Its
// members are contiguous, decode independently to the same entries, and cover the file.
func TestCreateGitArchiveSnapshotSegmented(t *testing.T) {
	repo, sha := commitFiles(t, 30, 200)
	dirName := filepath.Base(repo)

	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	index, err := createGitArchiveSegments(t.Context(), newGitRepo(repo), sha, out, dirName, nil, "", 4096)
	require.NoError(t, err)
	require.Greater(t, len(index.Members), 1)
	assert.Equal(t, dirName, index.Component)

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, int64(len(data)), index.Size)

	gz, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	single := tarNames(t, gz)
	require.NotEmpty(t, single)
	assert.Equal(t, "pax_global_header", single[0])

	var union []string
	var next int64
	for i, m := range index.Members {
		assert.Equal(t, next, m.Offset, "member %d not contiguous", i)
		next += m.Length
		member, err := gzip.NewReader(bytes.NewReader(data[m.Offset : m.Offset+m.Length]))
		require.NoError(t, err)
		names := tarNames(t, member)
		if i == 0 {
			require.NotEmpty(t, names)
			assert.Equal(t, "pax_global_header", names[0])
		}
		union = append(union, names...)
	}
	assert.Equal(t, index.Size, next)
	assert.Equal(t, single, union)
}

// An archive larger than maxMembers times the minimum target still splits into balanced
// members, instead of the last member absorbing the rest. Balance is checked on uncompressed
// tar bytes, which is what the launcher parallelizes over.
func TestCreateGitArchiveSnapshotBalancesLargeArchive(t *testing.T) {
	const minTarget = 4096
	// 64 files of about 4 KiB: about 256 KiB of blobs, well over 16 times the 4 KiB minimum.
	repo, sha := commitFiles(t, 64, 4000)
	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	index, err := createGitArchiveSegments(t.Context(), newGitRepo(repo), sha, out, filepath.Base(repo), nil, "", minTarget)
	require.NoError(t, err)
	require.LessOrEqual(t, len(index.Members), segmentMaxMembers)
	require.Greater(t, len(index.Members), 1)

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	var sizes []int64
	var total int64
	for _, m := range index.Members {
		gz, err := gzip.NewReader(bytes.NewReader(data[m.Offset : m.Offset+m.Length]))
		require.NoError(t, err)
		n, err := io.Copy(io.Discard, gz)
		require.NoError(t, err)
		sizes = append(sizes, n)
		total += n
	}
	average := total / int64(len(sizes))
	for i, n := range sizes {
		assert.LessOrEqual(t, n, 2*average, "member %d holds %d of %d uncompressed bytes", i, n, total)
	}
}
