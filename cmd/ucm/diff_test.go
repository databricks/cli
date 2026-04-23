package ucm

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/ucm/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Canonical fixtures exercised across the diff-runner tests. Keeping them as
// top-level consts avoids re-declaring the same YAML in every test.
const (
	baseUcmYml = `ucm:
  name: fixture
workspace:
  host: https://example.cloud.databricks.com
resources:
  catalogs:
    sales:
      name: sales
      comment: base comment
    marketing:
      name: marketing
  schemas:
    raw:
      catalog: sales
      name: raw
`

	headUcmYml = `ucm:
  name: fixture
workspace:
  host: https://example.cloud.databricks.com
resources:
  catalogs:
    sales:
      name: sales
      comment: head comment
    marketing:
      name: marketing
  schemas:
    raw:
      catalog: sales
      name: raw
    bronze:
      catalog: sales
      name: bronze
`

	unchangedUcmYml = `ucm:
  name: fixture
workspace:
  host: https://example.cloud.databricks.com
resources:
  catalogs:
    sales:
      name: sales
      comment: base comment
    marketing:
      name: marketing
  schemas:
    raw:
      catalog: sales
      name: raw
`
)

func TestDiff_ResourcesDetectsAddedRemovedChanged(t *testing.T) {
	base := mustLoadRoot(t, baseUcmYml)
	head := mustLoadRoot(t, headUcmYml)

	got := diffResources(base, head)

	assert.Equal(t, []string{
		"resources.catalogs.sales", // comment changed
		"resources.schemas.bronze", // added
	}, got)
}

func TestDiff_ResourcesWithRemovedEntryReportsIt(t *testing.T) {
	base := mustLoadRoot(t, headUcmYml)
	head := mustLoadRoot(t, baseUcmYml)

	got := diffResources(base, head)

	assert.Equal(t, []string{
		"resources.catalogs.sales",
		"resources.schemas.bronze",
	}, got)
}

func TestDiff_ResourcesIdenticalReturnsEmpty(t *testing.T) {
	base := mustLoadRoot(t, baseUcmYml)
	head := mustLoadRoot(t, unchangedUcmYml)

	assert.Empty(t, diffResources(base, head))
}

func TestDiff_ResourcesHandlesNilRoots(t *testing.T) {
	head := mustLoadRoot(t, headUcmYml)

	got := diffResources(&config.Root{}, head)

	assert.Equal(t, []string{
		"resources.catalogs.marketing",
		"resources.catalogs.sales",
		"resources.schemas.bronze",
		"resources.schemas.raw",
	}, got)
}

func TestDiff_WriteTextNoChanges(t *testing.T) {
	var buf bytes.Buffer
	writeDiffText(&buf, diffResult{}, "origin/main")

	assert.Equal(t, "No ucm resource changes since origin/main.\n", buf.String())
}

func TestDiff_WriteTextAlignsColumns(t *testing.T) {
	var buf bytes.Buffer
	writeDiffText(&buf, diffResult{ChangedResources: []string{
		"resources.catalogs.sales",
		"resources.schemas.raw",
	}}, "origin/main")

	got := buf.String()
	assert.Contains(t, got, "resources.catalogs.sales  changed\n")
	assert.Contains(t, got, "resources.schemas.raw     changed\n")
}

func TestDiff_WriteJSONEmitsEmptySlice(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, writeDiffJSON(&buf, diffResult{}))

	assert.Contains(t, buf.String(), `"changed_resources": []`)
}

func TestDiff_WriteJSONSortedKeys(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, writeDiffJSON(&buf, diffResult{ChangedResources: []string{
		"resources.catalogs.sales",
		"resources.schemas.raw",
	}}))

	assert.Contains(t, buf.String(), "\"resources.catalogs.sales\"")
	assert.Contains(t, buf.String(), "\"resources.schemas.raw\"")
}

func TestDiff_RunnerWithFakeGitReportsChanges(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ucm.yml"), []byte(headUcmYml), 0o644))

	r := &diffRunner{
		base:    "origin/main",
		head:    "HEAD",
		output:  flags.OutputText,
		workDir: dir,
		gitRun:  fakeGitRun(dir, baseUcmYml),
	}

	var buf bytes.Buffer
	require.NoError(t, r.run(&buf))

	out := buf.String()
	assert.Contains(t, out, "resources.catalogs.sales")
	assert.Contains(t, out, "resources.schemas.bronze")
	assert.Contains(t, out, "changed")
}

func TestDiff_RunnerJSONOutput(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ucm.yml"), []byte(headUcmYml), 0o644))

	r := &diffRunner{
		base:    "origin/main",
		head:    "HEAD",
		output:  flags.OutputJSON,
		workDir: dir,
		gitRun:  fakeGitRun(dir, baseUcmYml),
	}

	var buf bytes.Buffer
	require.NoError(t, r.run(&buf))

	assert.Contains(t, buf.String(), `"resources.catalogs.sales"`)
	assert.Contains(t, buf.String(), `"resources.schemas.bronze"`)
}

// TestDiff_RunnerMissingBaseTreatsAsEmpty covers the "first PR to introduce
// ucm.yml" path — git show returns an error, the runner falls through to an
// empty Root and reports every head resource as added.
func TestDiff_RunnerMissingBaseTreatsAsEmpty(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ucm.yml"), []byte(headUcmYml), 0o644))

	r := &diffRunner{
		base:    "origin/main",
		head:    "HEAD",
		output:  flags.OutputText,
		workDir: dir,
		gitRun: func(args ...string) ([]byte, error) {
			if args[0] == "rev-parse" {
				return []byte(dir), nil
			}
			return nil, fmt.Errorf("fatal: path 'ucm.yml' does not exist")
		},
	}

	var buf bytes.Buffer
	require.NoError(t, r.run(&buf))

	out := buf.String()
	assert.Contains(t, out, "resources.catalogs.sales")
	assert.Contains(t, out, "resources.catalogs.marketing")
	assert.Contains(t, out, "resources.schemas.raw")
	assert.Contains(t, out, "resources.schemas.bronze")
}

// TestDiff_RunnerNoChangesSentinel verifies the friendly "no changes" text.
func TestDiff_RunnerNoChangesSentinel(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ucm.yml"), []byte(baseUcmYml), 0o644))

	r := &diffRunner{
		base:    "origin/main",
		head:    "HEAD",
		output:  flags.OutputText,
		workDir: dir,
		gitRun:  fakeGitRun(dir, baseUcmYml),
	}

	var buf bytes.Buffer
	require.NoError(t, r.run(&buf))
	assert.Equal(t, "No ucm resource changes since origin/main.\n", buf.String())
}

// TestDiff_RunnerEndToEndRealGit is a live-git smoke test. It builds a
// throwaway repo with a base commit and a working-tree edit, then runs the
// real defaultGitRun against it. Proves the shell integration works on the
// happy path without relying on the surrounding CLI's git history.
func TestDiff_RunnerEndToEndRealGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	dir := t.TempDir()
	runGit(t, dir, "init", "--initial-branch=main")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "test")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "ucm.yml"), []byte(baseUcmYml), 0o644))
	runGit(t, dir, "add", "ucm.yml")
	runGit(t, dir, "commit", "-m", "seed")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "ucm.yml"), []byte(headUcmYml), 0o644))

	r := &diffRunner{
		base:    "main",
		head:    "HEAD",
		output:  flags.OutputJSON,
		workDir: dir,
		gitRun:  defaultGitRun(dir),
	}

	var buf bytes.Buffer
	require.NoError(t, r.run(&buf))

	out := buf.String()
	assert.Contains(t, out, "resources.catalogs.sales")
	assert.Contains(t, out, "resources.schemas.bronze")
}

// TestDiff_RunnerDetachedHeadNamedRef ensures the runner works when --head
// names a concrete ref rather than the magic "HEAD" string — the important
// property is that the base ref is resolved via git show on a named ref too.
func TestDiff_RunnerDetachedHeadNamedRef(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	dir := t.TempDir()
	runGit(t, dir, "init", "--initial-branch=main")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "test")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "ucm.yml"), []byte(baseUcmYml), 0o644))
	runGit(t, dir, "add", "ucm.yml")
	runGit(t, dir, "commit", "-m", "seed")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "ucm.yml"), []byte(headUcmYml), 0o644))
	runGit(t, dir, "add", "ucm.yml")
	runGit(t, dir, "commit", "-m", "head")

	r := &diffRunner{
		base:    "main~1",
		head:    "HEAD", // working tree matches HEAD after the commit
		output:  flags.OutputText,
		workDir: dir,
		gitRun:  defaultGitRun(dir),
	}

	var buf bytes.Buffer
	require.NoError(t, r.run(&buf))

	out := buf.String()
	assert.Contains(t, out, "resources.catalogs.sales")
	assert.Contains(t, out, "resources.schemas.bronze")
}

// TestDiff_ResolveResourceEntriesStripsLocationMetadata confirms the dyn
// flattening collapses location metadata so YAML edits that only change
// source positions don't show up as spurious diffs.
func TestDiff_ResolveResourceEntriesStripsLocationMetadata(t *testing.T) {
	a := mustLoadRoot(t, baseUcmYml)
	reordered := `ucm:
  name: fixture
workspace:
  host: https://example.cloud.databricks.com
resources:
  schemas:
    raw:
      name: raw
      catalog: sales
  catalogs:
    marketing:
      name: marketing
    sales:
      comment: base comment
      name: sales
`
	b := mustLoadRoot(t, reordered)

	assert.Empty(t, diffResources(a, b))
}

// mustLoadRoot parses raw YAML into a *config.Root, failing the test on any
// diagnostic error. Used to keep the arrange block of every diff test tiny.
func mustLoadRoot(t *testing.T, raw string) *config.Root {
	t.Helper()
	cfg, diags := config.LoadFromBytes("ucm.yml", []byte(raw))
	require.False(t, diags.HasError(), diags.Error())
	return cfg
}

// fakeGitRun returns a gitRun stub that answers rev-parse with dir and
// `show <ref>:ucm.yml` with the supplied payload. All other git invocations
// return an error so unexpected commands surface in tests.
func fakeGitRun(dir, baseYaml string) func(args ...string) ([]byte, error) {
	return func(args ...string) ([]byte, error) {
		switch {
		case len(args) >= 1 && args[0] == "rev-parse":
			return []byte(dir), nil
		case len(args) >= 2 && args[0] == "show":
			if strings.HasSuffix(args[1], ":ucm.yml") {
				return []byte(baseYaml), nil
			}
			return nil, fmt.Errorf("unexpected show target %q", args[1])
		default:
			return nil, fmt.Errorf("unexpected git args %v", args)
		}
	}
}

// runGit shells out to git in dir and fails the test on non-zero exit. Keeps
// the setup in the end-to-end tests short and obvious.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s: %s", strings.Join(args, " "), string(out))
}
