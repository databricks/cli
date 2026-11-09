package aircmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writePlainTarball packages files with createPlainTarball and fails the test on error.
func writePlainTarball(t *testing.T, ctx context.Context, repo, out string, files []snapshotFile) {
	t.Helper()
	_, err := createPlainTarball(ctx, repo, out, files)
	require.NoError(t, err)
}

// readSingleStream reads the .tar.gz as one gzip stream and returns the entry names,
// as an old launcher's `tar -xzf` does. It fails on any error other than the clean
// end of the archive.
func readSingleStream(t *testing.T, b []byte) []string {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(b))
	require.NoError(t, err)
	var names []string
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return names
		}
		require.NoError(t, err)
		names = append(names, hdr.Name)
		_, err = io.Copy(io.Discard, tr)
		require.NoError(t, err)
	}
}

// readMemberFragment decodes one member on its own and returns its entry names.
// Members other than the last end without the end-of-archive marker, so a clean EOF
// on a header boundary is the expected end.
func readMemberFragment(t *testing.T, member []byte) []string {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(member))
	require.NoError(t, err)
	var names []string
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return names
		}
		require.NoError(t, err)
		names = append(names, hdr.Name)
		_, err = io.Copy(io.Discard, tr)
		require.NoError(t, err)
	}
}

// makeSegmentRepo creates n files of growing size under a repo directory and returns
// the repo path and its snapshot file listing.
func makeSegmentRepo(t *testing.T, n int) (string, []snapshotFile) {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "code")
	for i := range n {
		writeRepoFile(t, repo, fmt.Sprintf("pkg%d/file%d.py", i%3, i), strings.Repeat(fmt.Sprintf("x = %d\n", i), 50*i+1))
	}
	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	return repo, files
}

func TestPlanSegments(t *testing.T) {
	tests := []struct {
		name       string
		sizes      []int64
		target     int64
		maxMembers int
		want       []int
	}{
		{name: "no entries is one member", sizes: nil, target: 100, maxMembers: 16, want: []int{0}},
		{name: "all zero-size entries is one member", sizes: []int64{0, 0}, target: 100, maxMembers: 16, want: []int{0}},
		{name: "under target is one member", sizes: []int64{40, 40}, target: 100, maxMembers: 16, want: []int{0}},
		{name: "balanced by bytes", sizes: []int64{50, 50, 50, 50}, target: 100, maxMembers: 16, want: []int{0, 2}},
		{name: "big entry is not split", sizes: []int64{300, 10}, target: 100, maxMembers: 16, want: []int{0, 1}},
		{name: "capped at max members", sizes: []int64{10, 10, 10, 10, 10, 10}, target: 10, maxMembers: 2, want: []int{0, 3}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, planSegments(tc.sizes, tc.target, tc.maxMembers))
		})
	}
}

// Old launchers read the whole .tar.gz as one gzip stream: it must yield exactly the
// expected entries, even with several members.
func TestSegmentedTarballReadsAsSingleStream(t *testing.T) {
	ctx := t.Context()
	repo, files := makeSegmentRepo(t, 30)
	entries, sizes := plainTarEntries(repo, files)
	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	index, err := writeSegmentedTarball(ctx, filepath.Base(repo), out, entries, sizes, 4096, 16)
	require.NoError(t, err)
	require.Greater(t, len(index.Members), 1)

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	var want []string
	for _, e := range entries {
		want = append(want, e.Name)
	}
	assert.ElementsMatch(t, want, readSingleStream(t, data))
}

// Each member decodes independently to a valid tar fragment, and the union of the
// members is the full set of entries in order.
func TestSegmentedTarballMembersDecodeIndependently(t *testing.T) {
	ctx := t.Context()
	repo, files := makeSegmentRepo(t, 30)
	entries, sizes := plainTarEntries(repo, files)
	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	index, err := writeSegmentedTarball(ctx, filepath.Base(repo), out, entries, sizes, 4096, 16)
	require.NoError(t, err)
	require.Greater(t, len(index.Members), 1)

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	var union []string
	for _, m := range index.Members {
		union = append(union, readMemberFragment(t, data[m.Offset:m.Offset+m.Length])...)
	}
	var want []string
	for _, e := range entries {
		want = append(want, e.Name)
	}
	assert.Equal(t, want, union)
}

// The index matches the tarball: members are contiguous, cover [0, size), size equals
// the file size, and every entry sits under the component directory.
func TestSegmentedTarballIndexContract(t *testing.T) {
	ctx := t.Context()
	repo, files := makeSegmentRepo(t, 30)
	entries, sizes := plainTarEntries(repo, files)
	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	index, err := writeSegmentedTarball(ctx, filepath.Base(repo), out, entries, sizes, 4096, 16)
	require.NoError(t, err)

	info, err := os.Stat(out)
	require.NoError(t, err)
	assert.Equal(t, segmentFormat, index.Format)
	assert.Equal(t, segmentFormatVersion, index.Version)
	assert.Equal(t, "code", index.Component)
	assert.Equal(t, info.Size(), index.Size)

	var next int64
	for _, m := range index.Members {
		assert.Equal(t, next, m.Offset)
		assert.Positive(t, m.Length)
		next += m.Length
	}
	assert.Equal(t, index.Size, next)

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	for _, name := range readSingleStream(t, data) {
		assert.True(t, strings.HasPrefix(name, "code/"), name)
	}
}

// Only the last member carries the end-of-archive marker.
func TestSegmentedTarballEndMarkerOnlyInLastMember(t *testing.T) {
	ctx := t.Context()
	repo, files := makeSegmentRepo(t, 30)
	entries, sizes := plainTarEntries(repo, files)
	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	index, err := writeSegmentedTarball(ctx, filepath.Base(repo), out, entries, sizes, 4096, 16)
	require.NoError(t, err)
	require.Greater(t, len(index.Members), 1)

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	for i, m := range index.Members {
		gz, err := gzip.NewReader(bytes.NewReader(data[m.Offset : m.Offset+m.Length]))
		require.NoError(t, err)
		payload, err := io.ReadAll(gz)
		require.NoError(t, err)
		trailer := make([]byte, 1024)
		endsWithMarker := bytes.Equal(payload[len(payload)-1024:], trailer)
		assert.Equal(t, i == len(index.Members)-1, endsWithMarker, "member %d", i)
	}
}

// A small repo packages as one member, and the tarball is a plain single-member archive.
func TestCreatePlainTarballSmallRepoIsOneMember(t *testing.T) {
	repo, files := makeSegmentRepo(t, 3)
	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	index, err := createPlainTarball(t.Context(), repo, out, files)
	require.NoError(t, err)
	assert.Len(t, index.Members, 1)
	assert.Equal(t, []segmentMember{{Offset: 0, Length: index.Size}}, index.Members)
}

// Same input yields byte-identical tarballs and indexes, which the content-addressed
// cache key depends on.
func TestCreatePlainTarballIsDeterministic(t *testing.T) {
	repo, files := makeSegmentRepo(t, 30)
	dir := t.TempDir()
	first := filepath.Join(dir, "first.tar.gz")
	second := filepath.Join(dir, "second.tar.gz")
	entries, sizes := plainTarEntries(repo, files)
	indexA, err := writeSegmentedTarball(t.Context(), filepath.Base(repo), first, entries, sizes, 4096, 16)
	require.NoError(t, err)
	indexB, err := writeSegmentedTarball(t.Context(), filepath.Base(repo), second, entries, sizes, 4096, 16)
	require.NoError(t, err)

	a, err := os.ReadFile(first)
	require.NoError(t, err)
	b, err := os.ReadFile(second)
	require.NoError(t, err)
	assert.Equal(t, a, b)
	assert.Equal(t, indexA, indexB)
}
