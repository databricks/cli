package main

import (
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const before = `package m

func f(x int) int {
	return x
}
`

// The PR adds g: its first branch ran in the acceptance tests, the second ran in nothing.
const after = `package m

func f(x int) int {
	return x
}

func g(x int) int {
	if x > 0 {
		return 1
	}
	return 2
}
`

func commitFile(t *testing.T, repo, content string) string {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(repo, "m", "a.go"), []byte(content), 0o644))
	git := []string{"GIT_CONFIG_GLOBAL=/dev/null", "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t"}
	run(t, repo, git, "git", "add", ".")
	run(t, repo, git, "git", "commit", "-q", "-m", "change")
	out, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	require.NoError(t, err)
	return strings.TrimSpace(string(out))
}

// packFile writes a build that covered blocks (catalog ids of the layout) in its acceptance
// and its unit tests.
func packFile(t *testing.T, l *layout, acc, unit []int32) string {
	t.Helper()
	bits := func(ids []int32) bitset {
		b := newBitset(len(l.blocks))
		for _, id := range ids {
			b.set(id)
		}
		return b
	}
	data, _ := encode([]byte(`{"cell":"direct"}`), []*layout{l}, []profile{
		{kind: kindAll, files: []int{0}, bits: bits(acc)},
		{kind: kindUnit, files: []int{0}, bits: bits(unit)},
	})
	p := filepath.Join(t.TempDir(), "coverage.bin.zst")
	_, err := writeCompressed(p, data)
	require.NoError(t, err)
	return p
}

func TestCompare(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repo, "m"), 0o755))
	run(t, repo, nil, "git", "init", "-q")
	from := commitFile(t, repo, before)
	to := commitFile(t, repo, after)
	blob, err := exec.Command("git", "-C", repo, "rev-parse", to+":m/a.go").Output()
	require.NoError(t, err)

	f := block{file: "m/a.go", sl: 3, sc: 19, el: 5, ec: 2, stmts: 1}
	baseLayout := &layout{path: "m/a.go", lines: 5, blocks: []block{f}}
	headLayout := &layout{path: "m/a.go", lines: 12, blocks: []block{
		f,
		{file: "m/a.go", sl: 7, sc: 19, el: 8, ec: 11, stmts: 1},
		{file: "m/a.go", sl: 8, sc: 11, el: 10, ec: 3, stmts: 1},
		{file: "m/a.go", sl: 11, sc: 2, el: 11, ec: 10, stmts: 1},
	}}
	_, err = hex.Decode(headLayout.sha[:], []byte(strings.TrimSpace(string(blob))))
	require.NoError(t, err)

	out := filepath.Join(t.TempDir(), "comment.md")
	require.NoError(t, compare(compareOptions{
		base:     []string{packFile(t, baseLayout, []int32{0}, []int32{0})},
		head:     []string{packFile(t, headLayout, []int32{0, 1, 2}, []int32{0})},
		repo:     repo,
		from:     from,
		to:       to,
		baseDesc: "BASE",
		headDesc: "HEAD",
		output:   out,
	}))
	data, err := os.ReadFile(out)
	require.NoError(t, err)
	comment := string(data)

	assert.True(t, strings.HasPrefix(comment, commentMarker+"\n"))
	assert.Contains(t, comment, "Linux (`direct`), statements covered by the unit and the acceptance tests. PR: HEAD. Base: BASE.")
	// Lines 7-11 are the added lines with code: 7-10 ran in the acceptance tests, 11 in nothing.
	assert.Contains(t, comment, "| Combined | 80.0% (4 of 5) | 75.0% | -25.00 pp |")
	assert.Contains(t, comment, "| Acceptance | 80.0% (4 of 5) | 75.0% | -25.00 pp |")
	assert.Contains(t, comment, "| Unit | 0.0% (0 of 5) | 25.0% | -75.00 pp |")
	assert.Contains(t, comment, "- `m/a.go`: 11\n")
	assert.Contains(t, comment, "| `m` | 4 (+3) | 75.0% | -25.00 pp | +1 |")

	// Without a base, the totals stand alone.
	require.NoError(t, compare(compareOptions{
		head: []string{packFile(t, headLayout, []int32{0, 1, 2}, nil)}, repo: repo, from: from, to: to, output: out,
	}))
	data, err = os.ReadFile(out)
	require.NoError(t, err)
	assert.Contains(t, string(data), "| Combined | 80.0% (4 of 5) | 75.0% |\n")
	assert.Contains(t, string(data), "No coverage of main was found")
	assert.NotContains(t, string(data), "changed packages")

	// A build of another version of the file is left out of the changed lines.
	headLayout.sha = [20]byte{}
	require.NoError(t, compare(compareOptions{
		head: []string{packFile(t, headLayout, []int32{0}, nil)}, repo: repo, from: from, to: to, output: out,
	}))
	data, err = os.ReadFile(out)
	require.NoError(t, err)
	assert.Contains(t, string(data), "| Combined | – | 25.0% |")
	assert.Contains(t, string(data), "1 changed file is left out")
}

func TestAddedLines(t *testing.T) {
	// The hunk of x.go adds a line that looks like a header; old.go is deleted, new.go added.
	diff := `diff --git a/x.go b/x.go
--- a/x.go
+++ b/x.go
@@ -3 +3,3 @@ func f() {
-a
+b
++++ b/fake.go
+c
@@ -10,2 +12,0 @@
-gone
-gone too
--- a/old.go
+++ /dev/null
@@ -1 +0,0 @@
-x
--- /dev/null
+++ b/new.go
@@ -0,0 +1 @@
+package m
\ No newline at end of file
`
	added, touched, err := addedLines(strings.NewReader(diff))
	require.NoError(t, err)
	assert.Equal(t, map[string][]int{"x.go": {3, 4, 5}, "new.go": {1}}, added)
	assert.Equal(t, map[string]bool{"x.go": true, "old.go": true, "new.go": true}, touched)
}

func TestDecodeRejectsHostileBlocks(t *testing.T) {
	// A block past the end of its file (compare would walk its lines), and a file without
	// blocks that claims many bitmaps (they would take no bytes).
	for _, l := range []*layout{
		{path: "a.go", lines: 3, blocks: []block{{file: "a.go", sl: 1, sc: 1, el: 1 << 31, ec: 2, stmts: 1}}},
	} {
		data, _ := encode([]byte(`{}`), []*layout{l}, []profile{{kind: kindAll, files: []int{0}, bits: bitset{1}}})
		_, err := decode(data)
		assert.ErrorIs(t, err, errMalformed)
	}
	data := []byte(magic)
	data = append(data, 0, 1, 1, 'a') // no meta, one layout "a"
	data = append(data, make([]byte, 20)...)
	data = append(data, 0, 0, 100) // no lines, no blocks, 100 bitmaps
	data = append(data, make([]byte, 200)...)
	_, err := decode(data)
	assert.ErrorIs(t, err, errMalformed)
}

func TestRangesAndNames(t *testing.T) {
	assert.Equal(t, "1-3, 5, 9-10", ranges([]int{1, 2, 3, 5, 9, 10}, 8))
	assert.Equal(t, "1, 3, ...", ranges([]int{1, 3, 5}, 2))
	assert.Equal(t, "a/b_c.go", safeName("a/b_c.go"))
	assert.Equal(t, "a?b?.go", safeName("a`b|.go"))
	assert.Equal(t, "-1,234", thousands(-1234))
}

func TestRenderPackagesNamesTheModuleRoot(t *testing.T) {
	var out strings.Builder
	base := map[string]counts{".": {stmts: 2, covered: [measures]int{1, 1, 0}}}
	head := map[string]counts{".": {stmts: 3, covered: [measures]int{1, 1, 0}}}
	renderPackages(&out, base, head, map[string]bool{".": true})
	assert.Contains(t, out.String(), "| (module root) | 3 (+1) | 33.3% |")
}
