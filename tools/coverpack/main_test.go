package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const program = `package main

import "os"

func main() {
	if len(os.Args) > 1 {
		hello()
	} else {
		world()
	}
	for i := range 2 {
		if i == 1 { os.Stdout.WriteString("one\n") }
	}
}

func hello() { os.Stdout.WriteString("hello\n") }

func world() {
	os.Stdout.WriteString("world\n")
}

func never() int {
	return 1
}
`

func run(t *testing.T, dir string, env []string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

// textfmt is what `go tool covdata textfmt` reads from dirs: its profile lines, sorted.
func textfmt(t *testing.T, dirs ...string) []string {
	out := filepath.Join(t.TempDir(), "profile.txt")
	run(t, ".", nil, "go", "tool", "covdata", "textfmt", "-i", strings.Join(dirs, ","), "-o", out)
	data, err := os.ReadFile(out)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")[1:] // mode: set
	slices.Sort(lines)
	return lines
}

func covered(lines []string) []string {
	return slices.DeleteFunc(slices.Clone(lines), func(l string) bool { return strings.HasSuffix(l, " 0") })
}

// sections is the profile lines of each section of a dump, sorted.
func sections(dump string) map[string][]string {
	result := map[string][]string{}
	var name string
	for line := range strings.Lines(dump) {
		line = strings.TrimSuffix(line, "\n")
		if h, ok := strings.CutPrefix(line, "# "); ok {
			name = h
			result[name] = []string{}
			continue
		}
		result[name] = append(result[name], line)
	}
	for _, lines := range result {
		slices.Sort(lines)
	}
	return result
}

func TestPackMatchesCovdata(t *testing.T) {
	src := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "go.mod"), []byte("module example.com/m\n\ngo 1.22\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "main.go"), []byte(program), 0o644))
	git := []string{"GIT_CONFIG_GLOBAL=/dev/null", "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t"}
	run(t, src, git, "git", "init", "-q")
	run(t, src, git, "git", "add", ".")
	run(t, src, git, "git", "commit", "-q", "-m", "program")
	head, err := exec.Command("git", "-C", src, "rev-parse", "HEAD").Output()
	require.NoError(t, err)
	bin := filepath.Join(t.TempDir(), "bin")
	if runtime.GOOS == "windows" {
		bin += ".exe" // exec only runs a file with an executable extension there
	}
	run(t, src, []string{"GOWORK=off", "GOFLAGS="}, "go", "build", "-cover", "-o", bin, ".")

	acc, unit := t.TempDir(), t.TempDir()
	for name, args := range map[string][]string{"TestAccept/hello": {"x"}, "TestAccept/world": nil} {
		dir := filepath.Join(acc, filepath.Base(name))
		require.NoError(t, os.Mkdir(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "name"), []byte(name), 0o644))
		run(t, src, []string{"GOCOVERDIR=" + dir}, bin, args...)
	}
	run(t, src, []string{"GOCOVERDIR=" + unit}, bin)
	// A directory without a name file is not a test, and one without counters ran nothing.
	require.NoError(t, os.Mkdir(filepath.Join(acc, "other"), 0o755))

	out := filepath.Join(t.TempDir(), "coverage.bin.zst")
	require.NoError(t, pack(options{unitDir: unit, accDir: acc, root: src, meta: `{"cell": "test"}`, output: out}))
	var buf bytes.Buffer
	require.NoError(t, dump(&buf, out))
	got := sections(buf.String())

	hello, world := filepath.Join(acc, "hello"), filepath.Join(acc, "world")
	assert.Equal(t, textfmt(t, hello, world), got["all"])
	assert.Equal(t, textfmt(t, unit), got["unit"])
	assert.Equal(t, covered(textfmt(t, hello)), got["test TestAccept/hello"])
	assert.Equal(t, covered(textfmt(t, world)), got["test TestAccept/world"])
	assert.Len(t, got, 5) // coverage (the meta), all, unit and the two tests
	assert.Contains(t, buf.String(), `"module":"example.com/m"`)
	assert.Contains(t, buf.String(), `"commit":"`+strings.TrimSpace(string(head))+`"`)
	assert.Contains(t, got["all"], "example.com/m/main.go:23.2,24.1 1 0") // never() did not run
}

func TestDecodeRejectsTruncated(t *testing.T) {
	data, _ := encode([]byte(`{}`), []*layout{{path: "a.go", blocks: []block{{file: "a.go", sl: 1, sc: 1, el: 2, ec: 2, stmts: 1}}}},
		[]profile{{kind: kindTest, name: "T", files: []int{0}, bits: bitset{1}}})
	d, err := decode(data)
	require.NoError(t, err)
	assert.Equal(t, [][]byte{{1}}, d.bitmaps[0])
	for n := range len(data) {
		_, err := decode(data[:n])
		assert.Error(t, err, "truncated to %d bytes", n)
	}
}
