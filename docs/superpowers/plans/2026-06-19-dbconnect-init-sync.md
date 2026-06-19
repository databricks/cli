# `databricks dbconnect init` / `sync` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `databricks dbconnect` command namespace with `init` and `sync` subcommands that provision a local Python `.venv` matched to the user's Databricks compute target (Python version, `databricks-connect` pin, and dependency constraints).

**Architecture:** A thin Cobra layer in `cmd/dbconnect/` wires flags and rendering; all logic lives in a unit-testable `libs/dbconnect/` package built around a shared phase `Pipeline` (parameterized by `Mode = Init|Sync`) and a `PackageManager` interface (uv only in this PR). Target resolution uses the SDK Clusters/Jobs APIs and the bundle's configured target; constraints are fetched from a configurable base URL; the `pyproject.toml` merge is surgical (formatting/comment-preserving).

**Tech Stack:** Go, Cobra, `github.com/databricks/databricks-sdk-go` (compute/jobs APIs), `github.com/BurntSushi/toml` (read-only parsing — already vendored), `libs/cmdio` (output), `libs/process` (uv shell-outs), `libs/cmdctx`/`cmd/root` (workspace client + bundle).

## Global Constraints

- **No new third-party dependency.** Use the already-vendored `github.com/BurntSushi/toml` for reading TOML; never use it to write the user's file.
- **Hand-written command, not codegen.** Do NOT touch `.codegen/` or run `generate-cligen`.
- **`--json` is the global `--output json` flag**, accessed via `root.OutputType(cmd)` returning `flags.Output` (`flags.OutputText`/`flags.OutputJSON`); render with `cmdio.Render(ctx, v)`. Do NOT add a custom `--json` flag.
- **Errors:** wrap with `%w`; compare with `errors.Is`/`errors.As` against sentinels, never `err.Error()` string content. Structured errors carry a stable `code`.
- **Env vars** in library/product code via `github.com/databricks/cli/libs/env` (`env.Get(ctx, ...)`/`env.Lookup(ctx, ...)`), not `os.Getenv`.
- **Logging** via `github.com/databricks/cli/libs/log` (`log.Warnf`/`Debugf`), stdout via `cmdio.LogString`. Paths printed with `filepath.ToSlash`.
- **Context** passed as an argument, never stored in a struct; never `context.Background()` outside `main`; tests use `t.Context()`.
- **Modern Go idioms:** `for i := range N`, `min`/`max` builtins, `switch` for same-decision alternatives, early-return for ordered precedence, collapse `if err != nil { return err }; return nil` to `return err`.
- **Test fixture hosts** use a reserved TLD (`.test`/`.invalid`).
- **Reference URLs in comments** when integrating an external tool/endpoint (uv installer, constraint repo, pip.conf).
- One focused PR; `NEXT_CHANGELOG.md` entry under **CLI**.

## Constants (verbatim, used across tasks)

- Default constraint base URL: `https://raw.githubusercontent.com/pietern/databricks-environments/main`
- Constraint base URL override env var: `DATABRICKS_DBCONNECT_CONSTRAINT_SOURCE`
- envKey for serverless: `serverless/serverless-{vN}` (e.g. `serverless/serverless-v4`)
- envKey for clusters/jobs: `dbr/{spark_version}` where `{spark_version}` is the cluster's `SparkVersion` verbatim (e.g. `dbr/15.4.x-scala2.12`)
- Backup suffix: `.bak`
- Managed-block marker (start): `# managed by databricks dbconnect — do not edit`
- Managed-block marker (end): `# end managed by databricks dbconnect`
- uv installer URL (comment reference only): `https://astral.sh/uv/install.sh`

## File Structure

```
cmd/dbconnect/
  dbconnect.go      New() *cobra.Command; "dbconnect" group; registers init + sync
  init.go           newInitCommand(): flag wiring + RunE -> pipeline.Run(Init)
  sync.go           newSyncCommand(): flag wiring + RunE -> pipeline.Run(Sync)
  output.go         renderResult(ctx, cmd, *dbconnect.Result) — text vs JSON

libs/dbconnect/
  result.go         Mode, Result, Plan, TargetInfo, ConstraintInfo, PhaseResult, PipelineError, error codes
  envkey.go         EnvKeyForServerless, EnvKeyForSparkVersion, PythonMinorFromRequires
  constraints.go    Constraints struct; FetchConstraints(ctx, baseURL, envKey) (+ cache)
  merge.go          MergeManaged(target []byte, c Constraints) (merged []byte, regions []string, err error)
  target.go         TargetResolver, ResolveTarget(...) (*TargetInfo, error)
  pkgmanager.go     PackageManager interface
  uv.go             uvManager implements PackageManager
  pipeline.go       Pipeline struct + Run(ctx)

acceptance/dbconnect/
  serverless-check/ , serverless-json/ , no-target/ , cluster-unsupported/ , flag-conflict/
```

---

### Task 1: Scaffold the command namespace + registration

**Files:**
- Create: `cmd/dbconnect/dbconnect.go`
- Create: `cmd/dbconnect/init.go`
- Create: `cmd/dbconnect/sync.go`
- Modify: `cmd/cmd.go` (import + `cli.AddCommand(dbconnect.New())`)
- Test: `acceptance/dbconnect/help/` (golden `output.txt`)

**Interfaces:**
- Produces: `func New() *cobra.Command` (the `dbconnect` group); private `newInitCommand()`/`newSyncCommand() *cobra.Command`.

- [ ] **Step 1: Create the namespace command.** `cmd/dbconnect/dbconnect.go`:

```go
package dbconnect

import "github.com/spf13/cobra"

// New returns the `dbconnect` command group.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dbconnect",
		Short:   "Set up a local Python environment matched to your Databricks compute",
		GroupID: "development",
		Long: `Set up a local Python environment matched to your Databricks compute target.

Derives the Python version, databricks-connect version, and dependency
constraints from the selected compute (cluster, serverless, or job) so that
local resolution matches the Databricks runtime.`,
	}
	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newSyncCommand())
	return cmd
}
```

- [ ] **Step 2: Create stub subcommands.** `cmd/dbconnect/init.go`:

```go
package dbconnect

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a fresh pyproject.toml and provision a matched .venv",
	}
	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}
	return cmd
}
```

`cmd/dbconnect/sync.go` is identical except `Use: "sync"`, `Short: "Merge managed dependencies into an existing pyproject.toml and re-provision"`, and `newSyncCommand`.

- [ ] **Step 3: Register in `cmd/cmd.go`.** Add import `"github.com/databricks/cli/cmd/dbconnect"` (alphabetical, near the `psql` import) and, in the "other subcommands" block next to `cli.AddCommand(psql.New())`:

```go
	cli.AddCommand(dbconnect.New())
```

- [ ] **Step 4: Build.**

Run: `./task build`
Expected: builds clean.

- [ ] **Step 5: Verify the command appears.**

Run: `./bin/databricks dbconnect --help`
Expected: shows `init` and `sync` subcommands.

- [ ] **Step 6: Add a help acceptance test.** Create `acceptance/dbconnect/help/script` containing:

```
$CLI dbconnect --help
$CLI dbconnect init --help
```

Then generate the golden output:

Run: `go test ./acceptance -run 'TestAccept/dbconnect/help' -tail -test.v -update`
Expected: creates `acceptance/dbconnect/help/output.txt`; re-running without `-update` passes.

- [ ] **Step 7: Commit.**

```bash
git add cmd/dbconnect/ cmd/cmd.go acceptance/dbconnect/help/
git commit -m "Add dbconnect command namespace scaffold"
```

---

### Task 2: Result types + error codes (`result.go`)

**Files:**
- Create: `libs/dbconnect/result.go`
- Test: `libs/dbconnect/result_test.go`

**Interfaces:**
- Produces:
  - `type Mode int` with `const ( ModeInit Mode = iota; ModeSync )` and `func (m Mode) String() string` (`"init"`/`"sync"`).
  - `type ErrorCode string` with consts: `ErrNoTargetSelected="no_target_selected"`, `ErrClusterUnsupported="cluster_unsupported"`, `ErrConstraintFetchFailed="constraint_fetch_failed"`, `ErrMergeFailed="merge_failed"`, `ErrProvisionFailed="provision_failed"`, `ErrValidationFailed="validation_failed"`, `ErrUvUnavailable="uv_unavailable"`.
  - `type PipelineError struct { Code ErrorCode; Msg string; Err error }` with `func (e *PipelineError) Error() string` and `func (e *PipelineError) Unwrap() error`.
  - `func NewError(code ErrorCode, err error, format string, args ...any) *PipelineError`.
  - Structs (all with `json:"..."` tags matching the spec): `TargetInfo{Kind, ClusterID, SparkVersion, EnvKey, PythonVersion string; Fallback *FallbackInfo}`, `FallbackInfo{Requested, Resolved string}`, `ConstraintInfo{SourceURL string; FromCache bool; RequiresPython, DatabricksConnect string; ConstraintCount int}`, `Plan{PyprojectPath, BackupPath, Diff string; ChangedRegions []string}`, `PhaseResult{Name, Status, Detail string}`, `ResultDetail{Status, VenvPath, PythonVersion, DatabricksConnectInstalled string}`, `Result{Mode string; Check bool; Target *TargetInfo; Constraints *ConstraintInfo; Plan *Plan; Phases []PhaseResult; Result *ResultDetail; Error *PipelineError}`.

- [ ] **Step 1: Write the failing test.** `libs/dbconnect/result_test.go`:

```go
package dbconnect

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPipelineErrorWrapsAndExposesCode(t *testing.T) {
	base := errors.New("boom")
	err := NewError(ErrConstraintFetchFailed, base, "fetch %s", "x")
	assert.Equal(t, "fetch x: boom", err.Error())
	assert.Equal(t, ErrConstraintFetchFailed, err.Code)
	assert.True(t, errors.Is(err, base))
}

func TestModeString(t *testing.T) {
	assert.Equal(t, "init", ModeInit.String())
	assert.Equal(t, "sync", ModeSync.String())
}
```

- [ ] **Step 2: Run test to verify it fails.**

Run: `go test ./libs/dbconnect/ -run 'TestPipelineError|TestModeString' -v`
Expected: FAIL (undefined symbols).

- [ ] **Step 3: Implement `result.go`** with the types from the Interfaces block. `NewError` formats `Msg` via `fmt.Sprintf(format, args...)`; `Error()` returns `Msg` plus `": "+Err.Error()` when `Err != nil`; `Unwrap()` returns `Err`.

- [ ] **Step 4: Run tests to verify they pass.**

Run: `go test ./libs/dbconnect/ -run 'TestPipelineError|TestModeString' -v`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add libs/dbconnect/result.go libs/dbconnect/result_test.go
git commit -m "Add dbconnect result types and error codes"
```

---

### Task 3: envKey mapping + Python-version parsing (`envkey.go`)

**Files:**
- Create: `libs/dbconnect/envkey.go`
- Test: `libs/dbconnect/envkey_test.go`

**Interfaces:**
- Produces:
  - `func EnvKeyForServerless(version string) string` — normalizes `"4"`, `"v4"` → `"serverless/serverless-v4"`.
  - `func EnvKeyForSparkVersion(sparkVersion string) string` — returns `"dbr/"+sparkVersion`.
  - `func PythonMinorFromRequires(requiresPython string) (string, error)` — parses a PEP 440 `requires-python` (e.g. `"==3.12.*"`, `">=3.12"`, `"==3.12.3"`) and returns `"3.12"`. Error if no `MAJOR.MINOR` can be extracted.

- [ ] **Step 1: Write the failing test.** `libs/dbconnect/envkey_test.go`:

```go
package dbconnect

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvKeyForServerless(t *testing.T) {
	for _, in := range []string{"4", "v4", "V4"} {
		assert.Equal(t, "serverless/serverless-v4", EnvKeyForServerless(in))
	}
}

func TestEnvKeyForSparkVersion(t *testing.T) {
	assert.Equal(t, "dbr/15.4.x-scala2.12", EnvKeyForSparkVersion("15.4.x-scala2.12"))
}

func TestPythonMinorFromRequires(t *testing.T) {
	cases := map[string]string{
		"==3.12.*": "3.12",
		">=3.12":   "3.12",
		"==3.12.3": "3.12",
		"~=3.11":   "3.11",
	}
	for in, want := range cases {
		got, err := PythonMinorFromRequires(in)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	_, err := PythonMinorFromRequires("garbage")
	assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails.**

Run: `go test ./libs/dbconnect/ -run 'TestEnvKey|TestPythonMinor' -v`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement `envkey.go`.** `EnvKeyForServerless`: lowercase, strip leading `v`, format `serverless/serverless-v%s`. `EnvKeyForSparkVersion`: `"dbr/" + sparkVersion`. `PythonMinorFromRequires`: use `regexp.MustCompile(\`(\d+)\.(\d+)\`)`, `FindStringSubmatch`; on no match return `fmt.Errorf("cannot parse python version from %q", requiresPython)`.

- [ ] **Step 4: Run tests to verify they pass.**

Run: `go test ./libs/dbconnect/ -run 'TestEnvKey|TestPythonMinor' -v`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add libs/dbconnect/envkey.go libs/dbconnect/envkey_test.go
git commit -m "Add dbconnect envKey mapping and python-version parsing"
```

---

### Task 4: Constraint fetch + cache + parse (`constraints.go`)

**Files:**
- Create: `libs/dbconnect/constraints.go`
- Test: `libs/dbconnect/constraints_test.go`

**Interfaces:**
- Consumes: `ErrConstraintFetchFailed`, `NewError` (Task 2); `PythonMinorFromRequires` (Task 3).
- Produces:
  - `type Constraints struct { EnvKey, SourceURL string; FromCache bool; RequiresPython, DatabricksConnect string; ConstraintDeps []string }`.
  - `func FetchConstraints(ctx context.Context, baseURL, envKey, cacheDir string) (*Constraints, error)` — GET `baseURL+"/"+envKey+"/pyproject.toml"`; on HTTP success, parse and write a cache copy to `cacheDir/<sanitized-envKey>.toml`; on network/HTTP error, fall back to the cached file with a `log.Warnf` if present, else return `NewError(ErrConstraintFetchFailed, ...)`. `FromCache` reflects which path served the bytes.
  - `func parseConstraints(data []byte) (requiresPython, dbconnect string, deps []string, err error)` — uses `toml.Unmarshal` into a struct mirroring `project.requires-python`, `dependency-groups.dev`, `tool.uv.constraint-dependencies`; selects the `dev` element whose despaced value starts with `databricks-connect`.

- [ ] **Step 1: Write the failing test.** `libs/dbconnect/constraints_test.go`:

```go
package dbconnect

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleToml = `[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [
    "databricks-connect~=17.2.0",
    "pytest~=8.0",
]

[tool.uv]
constraint-dependencies = [
    "pydantic~=2.10.6",
    "anyio~=4.6.2",
]
`

func TestParseConstraints(t *testing.T) {
	rp, dbc, deps, err := parseConstraints([]byte(sampleToml))
	require.NoError(t, err)
	assert.Equal(t, "==3.12.*", rp)
	assert.Equal(t, "databricks-connect~=17.2.0", dbc)
	assert.Equal(t, []string{"pydantic~=2.10.6", "anyio~=4.6.2"}, deps)
}

func TestFetchConstraintsHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/serverless/serverless-v4/pyproject.toml", r.URL.Path)
		_, _ = w.Write([]byte(sampleToml))
	}))
	defer srv.Close()

	c, err := FetchConstraints(t.Context(), srv.URL, "serverless/serverless-v4", t.TempDir())
	require.NoError(t, err)
	assert.False(t, c.FromCache)
	assert.Equal(t, "databricks-connect~=17.2.0", c.DatabricksConnect)
	assert.Len(t, c.ConstraintDeps, 2)
}

func TestFetchConstraintsFallsBackToCache(t *testing.T) {
	cacheDir := t.TempDir()
	// First, a successful fetch populates the cache.
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleToml))
	}))
	_, err := FetchConstraints(t.Context(), good.URL, "serverless/serverless-v4", cacheDir)
	require.NoError(t, err)
	good.Close()

	// Now the server is down; fetch must serve the cache.
	c, err := FetchConstraints(t.Context(), good.URL, "serverless/serverless-v4", cacheDir)
	require.NoError(t, err)
	assert.True(t, c.FromCache)
}
```

- [ ] **Step 2: Run test to verify it fails.**

Run: `go test ./libs/dbconnect/ -run 'TestParseConstraints|TestFetchConstraints' -v`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement `constraints.go`.** Build the URL; use an `http.Client` with the request's context (`http.NewRequestWithContext`). On a 2xx, read the body, call `parseConstraints`, write bytes to `filepath.Join(cacheDir, sanitize(envKey)+".toml")` (sanitize replaces `/` with `__`), set `FromCache=false`. On any transport error or non-2xx, attempt to read the cache file: if present, parse it, set `FromCache=true`, `log.Warnf(ctx, "constraint fetch failed, using cached copy: %v", err)`; if absent, return `NewError(ErrConstraintFetchFailed, err, "fetch constraints for %s", envKey)`. `parseConstraints` despaces each dev entry with `strings.ReplaceAll(s, " ", "")` before the `HasPrefix("databricks-connect")` check, but stores the original string. Add a comment citing the constraint repo URL.

- [ ] **Step 4: Run tests to verify they pass.**

Run: `go test ./libs/dbconnect/ -run 'TestParseConstraints|TestFetchConstraints' -v`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add libs/dbconnect/constraints.go libs/dbconnect/constraints_test.go
git commit -m "Add dbconnect constraint fetch with offline cache"
```

---

### Task 5: Surgical TOML merge (`merge.go`)

**Files:**
- Create: `libs/dbconnect/merge.go`
- Test: `libs/dbconnect/merge_test.go`
- Test fixtures: `libs/dbconnect/testdata/merge/*.toml`

**Interfaces:**
- Consumes: `Constraints` (Task 4).
- Produces:
  - `func MergeManaged(target []byte, c Constraints) (merged []byte, regions []string, err error)` — applies the three managed transforms below, preserving all other bytes (comments/order/whitespace). `regions` lists which of `"requires-python"`, `"databricks-connect"`, `"tool.uv.constraint-dependencies"` were changed. Idempotent: `MergeManaged(MergeManaged(x)) == MergeManaged(x)`.
  - `func RenderFreshPyproject(projectName string, c Constraints) []byte` — produces a complete managed `pyproject.toml` for `init` on a project that has none (used by Task 8 only when no file exists; if a file exists, `init` overwrites via MergeManaged after backup).

The three transforms:
1. `[project].requires-python` — replace the value of an existing `requires-python = ...` line within the `[project]` table, preserving indentation. If `[project]` exists without the key, insert the line directly under the `[project]` header.
2. The `databricks-connect` element inside `[dependency-groups].dev` — replace the existing element matching `"databricks-connect..."` in place, preserving leading indentation and trailing comma.
3. `[tool.uv].constraint-dependencies` — replace the marker-bracketed managed block; if no managed block exists, drop any plain `[tool.uv]` table we own and append a freshly rendered, marker-bracketed `[tool.uv]` block at end of file.

- [ ] **Step 1: Write the failing tests.** `libs/dbconnect/merge_test.go`:

```go
package dbconnect

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConstraints() Constraints {
	return Constraints{
		RequiresPython:    "==3.12.*",
		DatabricksConnect: "databricks-connect~=17.2.0",
		ConstraintDeps:    []string{"pydantic~=2.10.6", "anyio~=4.6.2"},
	}
}

func TestMergeReplacesRequiresPythonPreservingComments(t *testing.T) {
	in := []byte(`[project]
name = "demo"
# keep this comment
requires-python = ">=3.10"

[dependency-groups]
dev = [
    "databricks-connect~=16.0.0",
    "pytest~=8.0",
]
`)
	out, regions, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	assert.Contains(t, string(out), `requires-python = "==3.12.*"`)
	assert.Contains(t, string(out), "# keep this comment")
	assert.Contains(t, string(out), `"databricks-connect~=17.2.0",`)
	assert.Contains(t, string(out), `"pytest~=8.0",`)
	assert.Contains(t, regions, "requires-python")
	assert.Contains(t, regions, "databricks-connect")
	assert.Contains(t, regions, "tool.uv.constraint-dependencies")
	assert.Contains(t, string(out), "pydantic~=2.10.6")
}

func TestMergeIsIdempotent(t *testing.T) {
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = [
    "databricks-connect~=16.0.0",
]
`)
	once, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	twice, _, err := MergeManaged(once, testConstraints())
	require.NoError(t, err)
	assert.Equal(t, string(once), string(twice))
}

func TestMergeInsertsRequiresPythonWhenMissing(t *testing.T) {
	in := []byte(`[project]
name = "demo"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	assert.Contains(t, string(out), `requires-python = "==3.12.*"`)
}

func TestMergeReplacesExistingManagedToolUvBlock(t *testing.T) {
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]

` + managedMarkerStart + `
[tool.uv]
constraint-dependencies = [
    "stale~=1.0.0",
]
` + managedMarkerEnd + `
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	assert.NotContains(t, string(out), "stale~=1.0.0")
	assert.Contains(t, string(out), "pydantic~=2.10.6")
	// Only one managed block remains.
	assert.Equal(t, 1, countOccurrences(string(out), managedMarkerStart))
}
```

Add a tiny `countOccurrences` helper at the bottom of the test file using `strings.Count`.

- [ ] **Step 2: Run tests to verify they fail.**

Run: `go test ./libs/dbconnect/ -run 'TestMerge' -v`
Expected: FAIL (undefined `MergeManaged`, `managedMarkerStart`).

- [ ] **Step 3: Implement `merge.go`.** Define `const managedMarkerStart = "# managed by databricks dbconnect — do not edit"` and `const managedMarkerEnd = "# end managed by databricks dbconnect"`. Normalize CRLF→LF on entry, restore the original line ending on exit (detect by presence of `\r\n` in input). Work on `strings.Split(s, "\n")`:
  - **requires-python:** scan for a line matching `^\s*requires-python\s*=` (regexp) after the `[project]` header and before the next `^\[`; replace its value preserving the leading whitespace capture group. If absent, insert `requires-python = "<value>"` right after the `[project]` header line.
  - **databricks-connect:** scan within `[dependency-groups]` for a line containing `"databricks-connect`; replace the quoted token, preserving indentation and a trailing comma if the original had one. Record region only if a replacement happened.
  - **tool.uv block:** if a marker-bracketed block exists, replace the lines between (and including) the markers; else remove any existing `[tool.uv]` table (header to next `^\[` or EOF) and append a freshly rendered block. Render:
    ```
    <markerStart>
    [tool.uv]
    constraint-dependencies = [
        "dep1",
        "dep2",
    ]
    <markerEnd>
    ```
    separated from prior content by exactly one blank line; file ends with a single trailing newline.
  - `RenderFreshPyproject` builds a minimal `[project]` + `[dependency-groups].dev` (with the dbconnect pin) + the marker-bracketed `[tool.uv]` block.

- [ ] **Step 4: Run tests to verify they pass.**

Run: `go test ./libs/dbconnect/ -run 'TestMerge' -v`
Expected: PASS.

- [ ] **Step 5: Add CRLF + quote-style edge-case tests** and make them pass (extend `merge_test.go`):

```go
func TestMergePreservesCRLF(t *testing.T) {
	in := []byte("[project]\r\nrequires-python = \">=3.10\"\r\n\r\n[dependency-groups]\r\ndev = [\"databricks-connect~=16.0.0\"]\r\n")
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	assert.Contains(t, string(out), "\r\n")
	assert.Contains(t, string(out), `requires-python = "==3.12.*"`)
}
```

Run: `go test ./libs/dbconnect/ -run 'TestMerge' -v`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add libs/dbconnect/merge.go libs/dbconnect/merge_test.go
git commit -m "Add surgical formatting-preserving pyproject.toml merge"
```

---

### Task 6: Target resolution (`target.go`)

**Files:**
- Create: `libs/dbconnect/target.go`
- Test: `libs/dbconnect/target_test.go`

**Interfaces:**
- Consumes: `TargetInfo`, `ErrNoTargetSelected`, `ErrClusterUnsupported`, `NewError` (Task 2); `EnvKeyForServerless`, `EnvKeyForSparkVersion` (Task 3).
- Produces:
  - `type ComputeClient interface { GetClusterSparkVersion(ctx context.Context, clusterID string) (string, error); GetJobSparkVersion(ctx context.Context, jobID string) (string, isServerless bool, version string, err error) }` — a narrow seam over the SDK so tests stub it. (Job returns either a spark version or serverless marker.)
  - `type TargetFlags struct { Cluster, Serverless, Job string }`.
  - `type BundleTarget struct { ClusterID string; Serverless bool; Selected bool }` — the three-state result of reading the bundle's configured target (`Selected=false` ⇒ nothing selected).
  - `func ResolveTarget(ctx context.Context, f TargetFlags, c ComputeClient, bt BundleTarget) (*TargetInfo, error)` — precedence: cluster flag → serverless flag → job flag → bundle target. Produces `TargetInfo` with `EnvKey` set; `PythonVersion` is filled later from the fetched constraints (left empty here). Three-state errors when falling back to the bundle.
  - `func ValidateTargetFlags(f TargetFlags) error` — at most one of the three set (the Cobra layer also marks them mutually exclusive; this guards the library path).

- [ ] **Step 1: Write the failing test.** `libs/dbconnect/target_test.go`:

```go
package dbconnect

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCompute struct {
	clusterVersion string
	clusterErr     error
}

func (s stubCompute) GetClusterSparkVersion(_ context.Context, _ string) (string, error) {
	return s.clusterVersion, s.clusterErr
}
func (s stubCompute) GetJobSparkVersion(_ context.Context, _ string) (string, bool, string, error) {
	return "", false, "", nil
}

func TestResolveServerlessFlag(t *testing.T) {
	ti, err := ResolveTarget(t.Context(), TargetFlags{Serverless: "v4"}, stubCompute{}, BundleTarget{})
	require.NoError(t, err)
	assert.Equal(t, "serverless", ti.Kind)
	assert.Equal(t, "serverless/serverless-v4", ti.EnvKey)
}

func TestResolveClusterFlag(t *testing.T) {
	c := stubCompute{clusterVersion: "15.4.x-scala2.12"}
	ti, err := ResolveTarget(t.Context(), TargetFlags{Cluster: "abc"}, c, BundleTarget{})
	require.NoError(t, err)
	assert.Equal(t, "cluster", ti.Kind)
	assert.Equal(t, "dbr/15.4.x-scala2.12", ti.EnvKey)
	assert.Equal(t, "abc", ti.ClusterID)
}

func TestResolveBundleNothingSelected(t *testing.T) {
	_, err := ResolveTarget(t.Context(), TargetFlags{}, stubCompute{}, BundleTarget{Selected: false})
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrNoTargetSelected, pe.Code)
}

func TestResolveBundleServerless(t *testing.T) {
	ti, err := ResolveTarget(t.Context(), TargetFlags{}, stubCompute{}, BundleTarget{Selected: true, Serverless: true})
	require.NoError(t, err)
	assert.Equal(t, "serverless/serverless-v4", ti.EnvKey)
}

func TestValidateTargetFlagsMutuallyExclusive(t *testing.T) {
	assert.Error(t, ValidateTargetFlags(TargetFlags{Cluster: "a", Serverless: "v4"}))
	assert.NoError(t, ValidateTargetFlags(TargetFlags{Cluster: "a"}))
}
```

Note: `TestResolveBundleServerless` encodes the spec rule that a bundle serverless target with no recorded version defaults to `v4` (the script's documented stand-in). Add a code comment to that effect.

- [ ] **Step 2: Run test to verify it fails.**

Run: `go test ./libs/dbconnect/ -run 'TestResolve|TestValidateTargetFlags' -v`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement `target.go`** with ordered-precedence early returns. Cluster flag → `GetClusterSparkVersion` → `Kind:"cluster"`, `EnvKey: EnvKeyForSparkVersion(v)`. Serverless flag → normalize, `EnvKey: EnvKeyForServerless(v)`. Job flag → `GetJobSparkVersion`; serverless job → serverless envKey (default `v4`), else cluster envKey. No flag → read `bt`: `!Selected` → `NewError(ErrNoTargetSelected, nil, "No compute target is selected. Select a cluster or serverless target, or pass --cluster/--serverless/--job.")`; `Serverless` → serverless `v4` (with the documented-default comment); `ClusterID != ""` → resolve via `GetClusterSparkVersion`. `ValidateTargetFlags` counts non-empty fields; >1 → error naming the conflicting flags.

- [ ] **Step 4: Run tests to verify they pass.**

Run: `go test ./libs/dbconnect/ -run 'TestResolve|TestValidateTargetFlags' -v`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add libs/dbconnect/target.go libs/dbconnect/target_test.go
git commit -m "Add dbconnect target resolution with three-state messaging"
```

---

### Task 7: PackageManager interface + uv implementation (`pkgmanager.go`, `uv.go`)

**Files:**
- Create: `libs/dbconnect/pkgmanager.go`
- Create: `libs/dbconnect/uv.go`
- Test: `libs/dbconnect/uv_test.go`

**Interfaces:**
- Consumes: `libs/process` for shell-outs; `ErrUvUnavailable`, `ErrProvisionFailed`, `NewError` (Task 2).
- Produces:
  - `type PackageManager interface { Name() string; EnsureAvailable(ctx context.Context) (version string, err error); EnsurePython(ctx context.Context, minor string) error; Provision(ctx context.Context, projectDir string) error; PostProvision(ctx context.Context, projectDir string) error; Validate(ctx context.Context, projectDir string) (pythonVersion, dbconnectVersion string, err error) }`.
  - `type uvManager struct { bin string }` implementing it; `func newUvManager() *uvManager`.
  - `func discoverUv(ctx context.Context) (string, error)` — search `exec.LookPath`, then `~/.local/bin/uv`, `$XDG_BIN_HOME/uv`, `/opt/homebrew/bin/uv`, `/usr/local/bin/uv`. Returns the path or `NewError(ErrUvUnavailable, ...)`. (Bootstrapping via the installer is invoked by `EnsureAvailable` when discovery fails — guarded so tests can stub.)

Because real uv shell-outs can't run in unit tests, `uv_test.go` covers `discoverUv` path logic (with a fake bin dir on a temp `PATH`) and the argument construction of each command via a small indirection: `uvManager` builds `[]string` arg slices through unexported helpers (`syncArgs()`, `pipSeedArgs(py string)`, `pythonInstallArgs(minor string)`) that are unit-tested directly.

- [ ] **Step 1: Write the failing test.** `libs/dbconnect/uv_test.go`:

```go
package dbconnect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUvArgs(t *testing.T) {
	m := &uvManager{bin: "uv"}
	assert.Equal(t, []string{"sync"}, m.syncArgs())
	assert.Equal(t, []string{"python", "install", "3.12"}, m.pythonInstallArgs("3.12"))
	assert.Equal(t, []string{"pip", "install", "pip", "--python", "/p/.venv/bin/python"}, m.pipSeedArgs("/p/.venv/bin/python"))
}

func TestDiscoverUvFindsBinOnPath(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "uv")
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755))
	t.Setenv("PATH", dir)
	got, err := discoverUv(t.Context())
	require.NoError(t, err)
	assert.Equal(t, bin, got)
}
```

- [ ] **Step 2: Run test to verify it fails.**

Run: `go test ./libs/dbconnect/ -run 'TestUv|TestDiscoverUv' -v`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement `pkgmanager.go` (interface only) and `uv.go`.** `discoverUv` uses `exec.LookPath` first, then the candidate list (expand `~` via `os.UserHomeDir`, read `XDG_BIN_HOME` via `env.Lookup`). The arg helpers return the slices asserted above. `Provision` runs `uv sync` in `projectDir` via `process.Background` (or the repo's standard `process` runner) with `ctx`. `PostProvision` runs `uv pip install pip --python <venv python>` and carries the full Phase 7 rationale comment from the script (VS Code pip fallback; uv venvs lack pip; `uv sync` strips pip). `Validate` runs `uv run --no-project python -c` to read the Python minor and `importlib.metadata.version("databricks-connect")`. `EnsureAvailable` calls `discoverUv`; on failure, runs the installer (`curl ... | sh`) with a reference-URL comment, then re-discovers; on still-missing, returns `ErrUvUnavailable`.

- [ ] **Step 4: Run tests to verify they pass.**

Run: `go test ./libs/dbconnect/ -run 'TestUv|TestDiscoverUv' -v`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add libs/dbconnect/pkgmanager.go libs/dbconnect/uv.go libs/dbconnect/uv_test.go
git commit -m "Add PackageManager interface and uv implementation"
```

---

### Task 8: The pipeline (`pipeline.go`)

**Files:**
- Create: `libs/dbconnect/pipeline.go`
- Test: `libs/dbconnect/pipeline_test.go`

**Interfaces:**
- Consumes: every type above.
- Produces:
  - `type Pipeline struct { Mode Mode; Check bool; ProjectDir string; ConstraintBaseURL string; CacheDir string; Flags TargetFlags; Compute ComputeClient; Bundle BundleTarget; PM PackageManager }`.
  - `func (p *Pipeline) Run(ctx context.Context) (*Result, error)` — executes phases 1–8 (preflight folded into PM `EnsureAvailable`), honoring `Check` (stop after computing the plan/diff; no mutation). Returns a fully populated `*Result`; on a phase error, sets `Result.Error` and returns the error too.
  - Phase methods are unexported (`resolve`, `fetch`, `mergePlan`, `applyMerge`, `provision`, `validate`), each appending a `PhaseResult`.

Mode behavior: `ModeInit` — if `pyproject.toml` exists, back up to `.bak` then `MergeManaged`; if absent, `RenderFreshPyproject`. `ModeSync` — restore from `.bak` if present (else back up), then `MergeManaged`.

- [ ] **Step 1: Write the failing test** (drives the full pipeline with stubbed Compute + PM + httptest constraint server, against a temp project dir). `libs/dbconnect/pipeline_test.go`:

```go
package dbconnect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePM struct{ py, dbc string }

func (fakePM) Name() string                                       { return "fake" }
func (fakePM) EnsureAvailable(context.Context) (string, error)    { return "fake 1.0", nil }
func (fakePM) EnsurePython(context.Context, string) error         { return nil }
func (fakePM) Provision(context.Context, string) error            { return nil }
func (fakePM) PostProvision(context.Context, string) error        { return nil }
func (f fakePM) Validate(context.Context, string) (string, string, error) {
	return f.py, f.dbc, nil
}

func writeProject(t *testing.T) string {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(`[project]
name = "demo"
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]
`), 0o644))
	return dir
}

func newTestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleToml))
	}))
}

func TestPipelineCheckMutatesNothing(t *testing.T) {
	dir := writeProject(t)
	before, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeSync, Check: true, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.True(t, res.Check)
	require.NotNil(t, res.Plan)
	assert.Contains(t, res.Plan.Diff, "==3.12.*")
	after, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	assert.Equal(t, string(before), string(after)) // unchanged
}

func TestPipelineSyncProvisionsAndValidates(t *testing.T) {
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeSync, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	require.NotNil(t, res.Result)
	assert.Equal(t, "success", res.Result.Status)
	assert.Equal(t, "3.12", res.Result.PythonVersion)
	merged, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	assert.Contains(t, string(merged), `"databricks-connect~=17.2.0",`)
	assert.FileExists(t, filepath.Join(dir, "pyproject.toml.bak"))
}
```

(`sampleToml`, `stubCompute` come from earlier test files in the same package.)

- [ ] **Step 2: Run tests to verify they fail.**

Run: `go test ./libs/dbconnect/ -run 'TestPipeline' -v`
Expected: FAIL (undefined `Pipeline`).

- [ ] **Step 3: Implement `pipeline.go`.** `Run`: `EnsureAvailable` (record phase + `ErrUvUnavailable` on fail) → `resolve` (ResolveTarget) → `fetch` (FetchConstraints; fill `TargetInfo.PythonVersion` from `PythonMinorFromRequires(c.RequiresPython)`, build `ConstraintInfo`) → `mergePlan` (read existing file or empty; compute merged bytes via `MergeManaged`/`RenderFreshPyproject`; build `Plan` with a unified diff — use a small diff helper or `libs/textutil` if present, else a minimal line diff; set `ChangedRegions`). If `Check`, populate `Result` (Mode, Check, Target, Constraints, Plan) and return. Else `applyMerge` (Mode-specific backup/restore then write bytes) → `EnsurePython(py)` → `Provision` → `PostProvision` → `Validate` (assert minor==`py`; `databricks-connect` major matches the pin's major, else `ErrValidationFailed`) → populate `Result.Result`. Each phase appends a `PhaseResult{Name,Status,Detail}`.

- [ ] **Step 4: Run tests to verify they pass.**

Run: `go test ./libs/dbconnect/ -run 'TestPipeline' -v`
Expected: PASS.

- [ ] **Step 5: Run the whole package.**

Run: `go test ./libs/dbconnect/ -v`
Expected: all PASS.

- [ ] **Step 6: Commit.**

```bash
git add libs/dbconnect/pipeline.go libs/dbconnect/pipeline_test.go
git commit -m "Add dbconnect pipeline orchestrating all phases"
```

---

### Task 9: Wire the Cobra layer (flags, bundle/compute adapters, rendering)

**Files:**
- Modify: `cmd/dbconnect/init.go`, `cmd/dbconnect/sync.go`
- Create: `cmd/dbconnect/output.go`
- Create: `cmd/dbconnect/compute.go` (SDK adapter implementing `dbconnect.ComputeClient`)

**Interfaces:**
- Consumes: `dbconnect.Pipeline`, `dbconnect.ComputeClient`, `dbconnect.Result`, `root.OutputType`, `cmdctx.WorkspaceClient`, `root.TryConfigureBundle`.
- Produces: `func runPipeline(cmd *cobra.Command, mode dbconnect.Mode) error`; `type sdkCompute struct{ w *databricks.WorkspaceClient }` implementing `ComputeClient` via `w.Clusters.GetByClusterId` (→ `.SparkVersion`) and `w.Jobs.Get`.

- [ ] **Step 1: Implement the shared `runPipeline`** in `init.go` (sync.go calls it with `ModeSync`). Read flags (`--cluster/--serverless/--job/--check/--constraint-source`), build `TargetFlags`, `ValidateTargetFlags`, resolve `ProjectDir` (cwd), `CacheDir` (`os.UserCacheDir()/databricks/dbconnect`), `ConstraintBaseURL` (flag → `env.Lookup(ctx, "DATABRICKS_DBCONNECT_CONSTRAINT_SOURCE")` → default constant), `Compute: sdkCompute{w}`, `Bundle:` from `root.TryConfigureBundle` (map `ClusterId`/serverless mode → `BundleTarget`), `PM: newUvManager()`. Mark the three target flags mutually exclusive via `cmd.MarkFlagsMutuallyExclusive`. Call `p.Run(ctx)`, then `renderResult`.

- [ ] **Step 2: Implement `output.go`** `renderResult(cmd, res, err)`: when `root.OutputType(cmd) == flags.OutputJSON`, `cmdio.Render(ctx, res)`; else print the phase headers + a success/plan summary mirroring the script (`=== Phase N ===` style via `cmdio.LogString`). On error, JSON path still renders `res` (with `res.Error` set); text path returns the wrapped error.

- [ ] **Step 3: Implement `compute.go`** adapter. `GetClusterSparkVersion`: `d, err := w.Clusters.GetByClusterId(ctx, id)`; return `d.SparkVersion`. `GetJobSparkVersion`: `w.Jobs.Get`; inspect the job's first task/job-cluster for a `SparkVersion` or serverless. Add a comment if the job compute shape is non-obvious.

- [ ] **Step 4: Build + manual smoke.**

Run: `./task build && ./bin/databricks dbconnect init --serverless v4 --check --output json`
Expected: prints the JSON plan; no files changed. (Network to the constraint repo required; if offline, expect the `constraint_fetch_failed` code.)

- [ ] **Step 5: Commit.**

```bash
git add cmd/dbconnect/
git commit -m "Wire dbconnect Cobra layer: flags, compute adapter, rendering"
```

---

### Task 10: Acceptance tests

**Files:**
- Create: `acceptance/dbconnect/serverless-check/{script,output.txt}`
- Create: `acceptance/dbconnect/no-target/{script,output.txt}`
- Create: `acceptance/dbconnect/cluster-unsupported/{script,output.txt}`
- Create: `acceptance/dbconnect/flag-conflict/{script,output.txt}`
- Create: `acceptance/dbconnect/serverless-json/{script,output.txt}`
- Possibly: per-case `test.toml` to stub the constraint server + workspace (follow `acceptance/quickstart/` and any testserver-backed case).

**Interfaces:** Consumes the built CLI via the acceptance harness `$CLI`.

- [ ] **Step 1: Inspect an existing testserver-backed acceptance case** to copy the pattern for stubbing HTTP + the workspace client.

Run: `ls acceptance/cmd/ acceptance/auth/ && sed -n '1,40p' acceptance/quickstart/script 2>/dev/null`
Expected: see how `script`, `output.txt`, and `test.toml` cooperate (env, replacements, stubbed server).

- [ ] **Step 2: Write `flag-conflict`** (no network needed). `script`:

```
$CLI dbconnect init --cluster abc --serverless v4
```

Generate golden:

Run: `go test ./acceptance -run 'TestAccept/dbconnect/flag-conflict' -tail -test.v -update`
Expected: `output.txt` shows the mutual-exclusion error and non-zero exit.

- [ ] **Step 3: Write `no-target`** (bundle with no compute selected, no flags). Provide a minimal `databricks.yml` fixture in the case dir; `script`:

```
$CLI dbconnect init
```

Run: `go test ./acceptance -run 'TestAccept/dbconnect/no-target' -tail -test.v -update`
Expected: `output.txt` shows the "No compute target is selected…" message.

- [ ] **Step 4: Write `serverless-check`, `serverless-json`, `cluster-unsupported`** using the stubbed constraint server (point `DATABRICKS_DBCONNECT_CONSTRAINT_SOURCE` at the test server via `test.toml`/`script`). `serverless-check` runs `--serverless v4 --check`; `serverless-json` adds `--output json`; `cluster-unsupported` points `--cluster` at a stubbed cluster whose DBR has no constraint dir (server 404) → `cluster_unsupported`/`constraint_fetch_failed`.

Run: `go test ./acceptance -run 'TestAccept/dbconnect' -tail -test.v -update`
Expected: all goldens created.

- [ ] **Step 5: Verify without `-update`.**

Run: `go test ./acceptance -run 'TestAccept/dbconnect' -tail -test.v`
Expected: all PASS.

- [ ] **Step 6: Commit.**

```bash
git add acceptance/dbconnect/
git commit -m "Add dbconnect acceptance tests"
```

---

### Task 11: Changelog, lint, fmt, full suite

**Files:**
- Modify: `NEXT_CHANGELOG.md`

- [ ] **Step 1: Add the changelog entry** under `### CLI` in `NEXT_CHANGELOG.md`:

```markdown
* Add `databricks dbconnect init` and `databricks dbconnect sync` to provision a local Python environment (Python version, `databricks-connect` pin, and dependency constraints) matched to the selected Databricks compute target.
```

- [ ] **Step 2: Format changed files.**

Run: `./task fmt-q`
Expected: no diff or auto-applied formatting only.

- [ ] **Step 3: Lint changed files.**

Run: `./task lint-q`
Expected: clean (fix anything reported).

- [ ] **Step 4: Full test suite.**

Run: `./task test`
Expected: all PASS.

- [ ] **Step 5: Commit.**

```bash
git add NEXT_CHANGELOG.md
git commit -m "Add changelog entry for dbconnect init/sync"
```

---

## Self-Review

**Spec coverage:**
- Namespace + `init`/`sync` → Tasks 1, 9. ✓
- Phase pipeline (0–8) → Task 8 (preflight folded into PM.EnsureAvailable, Task 7). ✓
- Shared flags `--cluster/--serverless/--job/--check/--json` → Task 9; `--json` realized as global `--output json` per Global Constraints. ✓
- Target resolution via API + three-state messaging + full cluster/job → Tasks 6, 9. ✓
- Robust surgical TOML merge of the 3 managed regions → Task 5. ✓
- Constraint fetch (configurable URL) + offline cache → Task 4. ✓
- Structured `--json` schema + `--check` dry-run → Tasks 2, 8, 9. ✓
- uv branch incl. pip-seed (Phase 7) rationale → Task 7. ✓
- Acceptance cases (serverless happy/check, no-target, cluster-stub→unsupported, --check, --json) → Task 10. ✓
- Unit tests for merge/envkey/target/constraints → Tasks 3–8. ✓
- Changelog + lint + fmt → Task 11. ✓
- "uv only now, pip/conda later" → PackageManager interface (Task 7), no pip/conda files. ✓
- No new dependency → uses vendored BurntSushi (read-only) + stdlib. ✓

**Placeholder scan:** No "TBD"/"handle edge cases"/"similar to". Each code step shows code; each run step shows command + expected output. The one explicit investigation step (Task 10 Step 1) is a deliberate "inspect existing pattern" action, not a placeholder.

**Type consistency:** `MergeManaged`, `FetchConstraints`, `ResolveTarget`, `TargetFlags`, `BundleTarget`, `ComputeClient`, `PackageManager`, `Pipeline`, `Result`/`Plan`/`TargetInfo`/`ConstraintInfo` names are used identically across Tasks 2–9. `managedMarkerStart`/`managedMarkerEnd` consistent between Task 5 impl and tests. uv arg-helper names (`syncArgs`,`pythonInstallArgs`,`pipSeedArgs`) consistent between Task 7 impl and tests.

**Known follow-ups (out of scope, noted for the implementer):** confirm the exact `databricks.yml` shape used to derive `BundleTarget` from `TryConfigureBundle` (cluster_id vs serverless mode) during Task 9; the SDK `Jobs.Get` compute shape may need a small comment per the repo's "non-obvious backend quirk" rule.
