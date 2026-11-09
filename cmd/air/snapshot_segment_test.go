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

	"github.com/databricks/cli/libs/tarpack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writePlainTarball packages files with createPlainTarball and fails the test on error.
func writePlainTarball(t *testing.T, ctx context.Context, repo, out string, files []snapshotFile) {
	t.Helper()
	_, err := createPlainTarball(ctx, repo, out, files)
	require.NoError(t, err)
}

// tarNames reads a tar stream to its end and returns the entry names. A clean EOF is
// accepted at an entry boundary, so it also reads a member fragment with no end marker.
func tarNames(t *testing.T, r io.Reader) []string {
	t.Helper()
	var names []string
	tr := tar.NewReader(r)
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

// segmentFixture is a repo packaged as a multi-member segmented tarball. The small target
// forces several members.
type segmentFixture struct {
	entries []tarpack.Entry
	index   segmentIndex
	data    []byte
}

func newSegmentFixture(t *testing.T) segmentFixture {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "code")
	for i := range 30 {
		writeRepoFile(t, repo, fmt.Sprintf("pkg%d/file%d.py", i%3, i), strings.Repeat(fmt.Sprintf("x = %d\n", i), 50*i+1))
	}
	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	entries, sizes := plainTarEntries(repo, files)

	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	index, err := writeSegmentedTarball(t.Context(), "code", out, entries, sizes, 4096, 16)
	require.NoError(t, err)
	require.Greater(t, len(index.Members), 1)
	data, err := os.ReadFile(out)
	require.NoError(t, err)
	return segmentFixture{entries: entries, index: index, data: data}
}

func (f segmentFixture) names() []string {
	var names []string
	for _, e := range f.entries {
		names = append(names, e.Name)
	}
	return names
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
	f := newSegmentFixture(t)
	gz, err := gzip.NewReader(bytes.NewReader(f.data))
	require.NoError(t, err)
	assert.Equal(t, f.names(), tarNames(t, gz))
}

// Each member decodes on its own to a tar fragment, and the members in order are the full
// entry set. Only the last member carries the end-of-archive marker.
func TestSegmentedTarballMembersDecodeIndependently(t *testing.T) {
	f := newSegmentFixture(t)
	var union []string
	for i, m := range f.index.Members {
		member := f.data[m.Offset : m.Offset+m.Length]
		gz, err := gzip.NewReader(bytes.NewReader(member))
		require.NoError(t, err)
		payload, err := io.ReadAll(gz)
		require.NoError(t, err)
		endsWithMarker := bytes.HasSuffix(payload, make([]byte, 1024))
		assert.Equal(t, i == len(f.index.Members)-1, endsWithMarker, "member %d", i)

		gz, err = gzip.NewReader(bytes.NewReader(member))
		require.NoError(t, err)
		union = append(union, tarNames(t, gz)...)
	}
	assert.Equal(t, f.names(), union)
}

// The index matches the tarball: members are contiguous, cover [0, size) with size equal
// to the file size, and every entry sits under the component directory.
func TestSegmentedTarballIndexContract(t *testing.T) {
	f := newSegmentFixture(t)
	assert.Equal(t, segmentFormat, f.index.Format)
	assert.Equal(t, segmentFormatVersion, f.index.Version)
	assert.Equal(t, "code", f.index.Component)
	assert.Equal(t, int64(len(f.data)), f.index.Size)

	var next int64
	for _, m := range f.index.Members {
		assert.Equal(t, next, m.Offset)
		assert.Positive(t, m.Length)
		next += m.Length
	}
	assert.Equal(t, f.index.Size, next)

	for _, name := range f.names() {
		assert.True(t, strings.HasPrefix(name, "code/"), name)
	}
}

// A small repo packages as one member, and the index covers the whole file.
func TestCreatePlainTarballSmallRepoIsOneMember(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "code")
	writeRepoFile(t, repo, "a.py", "print()")
	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	out := filepath.Join(t.TempDir(), "snap.tar.gz")
	index, err := createPlainTarball(t.Context(), repo, out, files)
	require.NoError(t, err)

	info, err := os.Stat(out)
	require.NoError(t, err)
	assert.Equal(t, []segmentMember{{Offset: 0, Length: info.Size()}}, index.Members)
	assert.Equal(t, info.Size(), index.Size)
}

// Same input yields byte-identical tarballs and indexes, which the content-addressed cache
// key depends on.
func TestCreatePlainTarballIsDeterministic(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "code")
	for i := range 30 {
		writeRepoFile(t, repo, fmt.Sprintf("f%d.py", i), strings.Repeat("x", 300*i+1))
	}
	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	entries, sizes := plainTarEntries(repo, files)

	dir := t.TempDir()
	write := func(name string) (segmentIndex, []byte) {
		out := filepath.Join(dir, name)
		index, err := writeSegmentedTarball(t.Context(), "code", out, entries, sizes, 4096, 16)
		require.NoError(t, err)
		data, err := os.ReadFile(out)
		require.NoError(t, err)
		return index, data
	}
	indexA, dataA := write("a.tar.gz")
	indexB, dataB := write("b.tar.gz")
	require.Greater(t, len(indexA.Members), 1)
	assert.Equal(t, dataA, dataB)
	assert.Equal(t, indexA, indexB)
}
