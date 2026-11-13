package aircmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlainTarNoticeLines(t *testing.T) {
	assert.Equal(t,
		[]string{"No git repository at /tmp/air-quickstart; packaging 3 file(s) (64.0 KiB uncompressed) from the working tree."},
		plainTarNoticeLines("/tmp/air-quickstart", 3, 65576, false))
	assert.Equal(t,
		[]string{"No git repository at /tmp/air-quickstart; reusing the cached snapshot (1 file(s), 15 B uncompressed)."},
		plainTarNoticeLines("/tmp/air-quickstart", 1, 15, true))
}

func TestPlainTarNoticeLargeSourceTip(t *testing.T) {
	const tip = "Tip: limit the snapshot with code_source.include_paths or a .gitignore in the directory."
	assert.Len(t, plainTarNoticeLines("/x", 1, snapshotLargeSourceBytes-1, false), 1)

	lines := plainTarNoticeLines("/x", 2, snapshotLargeSourceBytes, false)
	assert.Equal(t, []string{"No git repository at /x; packaging 2 file(s) (64.0 MiB uncompressed) from the working tree.", tip}, lines)

	// A cache hit packages nothing, so there is nothing to narrow.
	assert.Len(t, plainTarNoticeLines("/x", 2, snapshotLargeSourceBytes, true), 1)
}

func TestShippedSnapshotFilesMatchesTarpack(t *testing.T) {
	files := []snapshotFile{
		{rel: "main.py", size: 15, mode: 0o644},
		{rel: "link.py", size: 7, mode: uint32(os.ModeSymlink | 0o777), linkTarget: "main.py"},
		{rel: "lib/", size: 4096, mode: uint32(os.ModeDir | 0o755)},
	}
	count, sourceBytes, skipped := shippedSnapshotFiles(files)
	assert.Equal(t, 2, count)
	// A symlink is stored as a link entry with no content.
	assert.Equal(t, int64(15), sourceBytes)
	assert.Equal(t, []string{"lib/"}, skipped)
}

func TestSkippedSnapshotWarningCapsNames(t *testing.T) {
	assert.Equal(t,
		"Warning: the snapshot leaves out 1 listed path(s) that are not regular files or symlinks, such as nested git repositories: lib/.",
		skippedSnapshotWarning([]string{"lib/"}))
	assert.Equal(t,
		"Warning: the snapshot leaves out 7 listed path(s) that are not regular files or symlinks, such as nested git repositories: a/, b/, c/, d/, e/ (and 2 more).",
		skippedSnapshotWarning([]string{"a/", "b/", "c/", "d/", "e/", "f/", "g/"}))
}

// syncBuffer is a goroutine-safe stderr: the spinner's renderer and the submit
// path write to it concurrently.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// nonGitSubmitServer stubs a submit and keeps the uploaded snapshot tarballs by
// path (the filer may retry a write after creating parent directories), so a test
// can compare the notice against what actually shipped.
func nonGitSubmitServer(t *testing.T) (*databricks.WorkspaceClient, func() [][]byte) {
	t.Helper()
	server := testserver.New(t)
	t.Cleanup(server.Close)

	var mu sync.Mutex
	tarballs := map[string][]byte{}
	server.Handle("POST", "/api/2.0/workspace-files/import-file/{path...}", func(req testserver.Request) any {
		p := req.Vars["path"]
		if strings.Contains(p, "/.air/repo_snapshots/") {
			mu.Lock()
			tarballs[p] = bytes.Clone(req.Body)
			mu.Unlock()
		}
		return req.Workspace.WorkspaceFilesImportFile(p, req.Body, req.URL.Query().Get("overwrite") == "true")
	})
	server.Handle("POST", "/api/2.2/jobs/runs/submit", func(req testserver.Request) any {
		return jobs.SubmitRunResponse{RunId: 555}
	})
	stubValidateConfig(server)
	testserver.AddDefaultHandlers(server)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	return w, func() [][]byte {
		mu.Lock()
		defer mu.Unlock()
		return slices.Collect(maps.Values(tarballs))
	}
}

func nonGitSnapshotConfig(t *testing.T, root string) (*runConfig, string) {
	t.Helper()
	cfgPath := writeConfigFile(t, "run.yaml", minimalConfig+`
code_source:
  type: snapshot
  snapshot:
    root_path: `+root+`
`)
	cfg, err := loadRunConfig(cfgPath)
	require.NoError(t, err)
	return cfg, cfgPath
}

func tarEntryNames(t *testing.T, tarball []byte) []string {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(tarball))
	require.NoError(t, err)
	tr := tar.NewReader(gz)
	var names []string
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return names
		}
		require.NoError(t, err)
		names = append(names, hdr.Name)
	}
}

// A nested git repository inside a non-git root is listed by ls-files as one
// directory entry, which the tarball skips together with its files. The notice
// must count what ships, and a warning must name the skipped directory.
func TestSubmitWorkloadNonGitNestedRepo(t *testing.T) {
	w, tarballs := nonGitSubmitServer(t)

	root := filepath.Join(t.TempDir(), "proj")
	writeRepoFile(t, root, "main.py", "print('hello')\n")
	writeRepoFile(t, root, "run.log", "log\n")
	writeRepoFile(t, root, "lib/x.py", strings.Repeat("x", 100))
	runGit(t, filepath.Join(root, "lib"), "init", "-q")
	cfg, cfgPath := nonGitSnapshotConfig(t, root)

	var stderr syncBuffer
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, strings.NewReader(""), io.Discard, &stderr, "", ""))
	_, _, err := submitWorkload(ctx, w, cfg, cfgPath, "idem", true)
	require.NoError(t, err)

	uploaded := tarballs()
	require.Len(t, uploaded, 1)
	assert.ElementsMatch(t, []string{"proj/main.py", "proj/run.log"}, tarEntryNames(t, uploaded[0]))

	out := stderr.String()
	assert.Contains(t, out, "Warning: the snapshot leaves out 1 listed path(s) that are not regular files or symlinks, such as nested git repositories: lib/.\n")
	assert.Contains(t, out, "No git repository at "+filepath.ToSlash(root)+"; packaging 2 file(s) (19 B uncompressed) from the working tree.\n")

	// --dry-run reports the same count.
	count, err := countSnapshotFiles(t.Context(), cfg.CodeSource.Snapshot, cfgPath)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

// The notice prints before the staging spinner starts, so the spinner never
// draws over it. cmdio.NewTestIO forces an interactive stderr, so the spinner
// really renders: its first output (bubbletea's cursor-hiding escape sequence)
// must come after the notice and the warning.
func TestSubmitWorkloadNonGitNoticeBeforeSpinner(t *testing.T) {
	w, _ := nonGitSubmitServer(t)

	root := filepath.Join(t.TempDir(), "proj")
	writeRepoFile(t, root, "main.py", "print('hello')\n")
	writeRepoFile(t, root, "lib/x.py", "x")
	runGit(t, filepath.Join(root, "lib"), "init", "-q")
	cfg, cfgPath := nonGitSnapshotConfig(t, root)

	var stderr syncBuffer
	ctx := cmdio.InContext(t.Context(), cmdio.NewTestIO(strings.NewReader(""), io.Discard, &stderr))
	_, _, err := submitWorkload(ctx, w, cfg, cfgPath, "idem", true)
	require.NoError(t, err)

	out := stderr.String()
	warning := strings.Index(out, "Warning: the snapshot leaves out")
	notice := strings.Index(out, "No git repository at ")
	spinner := strings.Index(out, "\x1b[")
	uploading := strings.Index(out, "Uploading ")
	require.NotEqual(t, -1, warning, out)
	require.NotEqual(t, -1, notice, out)
	require.NotEqual(t, -1, spinner, "the spinner rendered nothing: %q", out)
	require.NotEqual(t, -1, uploading, out)
	assert.Less(t, warning, notice)
	assert.Less(t, notice, spinner, "notice must precede the spinner: %q", out)
	assert.Less(t, notice, uploading)
}

// An unchanged resubmit reuses the uploaded tarball and says so; under -o json
// (showProgress=false) the notice is skipped like "Submitting experiment:", but
// the skipped-path warning still prints.
func TestSubmitWorkloadNonGitReuseAndJSON(t *testing.T) {
	w, tarballs := nonGitSubmitServer(t)

	root := filepath.Join(t.TempDir(), "proj")
	writeRepoFile(t, root, "train.py", "print()")
	cfg, cfgPath := nonGitSnapshotConfig(t, root)

	submit := func(showProgress bool) string {
		var stderr syncBuffer
		ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, strings.NewReader(""), io.Discard, &stderr, "", ""))
		_, _, err := submitWorkload(ctx, w, cfg, cfgPath, "idem", showProgress)
		require.NoError(t, err)
		return stderr.String()
	}

	assert.Contains(t, submit(true), "; packaging 1 file(s) (7 B uncompressed) from the working tree.")
	reuse := submit(true)
	assert.Contains(t, reuse, "No git repository at "+filepath.ToSlash(root)+"; reusing the cached snapshot (1 file(s), 7 B uncompressed).\n")
	assert.NotContains(t, reuse, "packaging")
	assert.Len(t, tarballs(), 1, "the resubmit must not upload again")

	assert.NotContains(t, submit(false), "No git repository")

	require.NoError(t, os.MkdirAll(filepath.Join(root, "lib"), 0o755))
	runGit(t, filepath.Join(root, "lib"), "init", "-q")
	writeRepoFile(t, root, "lib/x.py", "x")
	jsonOut := submit(false)
	assert.Contains(t, jsonOut, "Warning: the snapshot leaves out 1 listed path(s)")
	assert.NotContains(t, jsonOut, "No git repository")
}

// A git repository gets no notice: git scopes its snapshot.
func TestSubmitWorkloadGitRepoNoNotice(t *testing.T) {
	w, _ := nonGitSubmitServer(t)
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "train.py", "print()")
	commitAll(t, repo, "init")
	cfg, cfgPath := nonGitSnapshotConfig(t, repo)

	var stderr syncBuffer
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, strings.NewReader(""), io.Discard, &stderr, "", ""))
	_, _, err := submitWorkload(ctx, w, cfg, cfgPath, "idem", true)
	require.NoError(t, err)
	assert.NotContains(t, stderr.String(), "No git repository")
}
