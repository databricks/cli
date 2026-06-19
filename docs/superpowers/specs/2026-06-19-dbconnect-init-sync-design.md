# `databricks dbconnect init` / `sync` — Design

**Date:** 2026-06-19
**Status:** Approved for planning
**Branch context:** Databricks CLI (Go)

## Summary

Promote the proven `dbconnect-init.sh` demo into a real CLI subcommand namespace,
`databricks dbconnect`, with two commands: `init` and `sync`. Starting from the
compute target the user already selected (cluster / serverless / job), the
command derives and provisions a matching local Python environment: the right
Python version, the right `databricks-connect` version, and dependency
constraints so local resolution matches the Databricks runtime — no version
guessing.

The behavior is already implemented and verified as a 367-line bash script
(`dbconnect-init.sh` in the `databricks-vscode` repo). This design ports the
same phase pipeline to Go, with real API calls, a robust TOML merge, a
package-manager seam, and structured output.

## Reference implementation (the spec)

- **Script (source of truth for the pipeline):**
  `/Users/grigory.panov/work/databricks-vscode/packages/databricks-vscode/resources/python/dbconnect-init.sh`
- **VS Code consumer (context on how it's invoked + the `--json` consumer):**
  `/Users/grigory.panov/work/databricks-vscode/packages/databricks-vscode/src/language/VpexEnvironmentSetup.ts`

The script logs each `=== Phase N ===` header; the Go port matches those
outcomes. We can diff Go behavior against a live run of the script.

## Design decisions (resolved during brainstorming)

1. **Constraint source of truth:** configurable base URL, defaulting to the
   existing `databricks-environments` GitHub raw repo. Swap the default when an
   official endpoint exists. (Overridable via flag + env var.)
2. **Default target resolution:** when no `--cluster`/`--serverless`/`--job`
   flag is given, resolve from the **bundle's configured target** (the way
   bundle commands do), NOT from the VS Code `vscode.overrides.json` artifact.
   The standalone CLI does not read VS Code files.
3. **Package managers:** **uv only** in this PR, at full parity with the script.
   A `PackageManager` interface is the seam; pip and conda land in later PRs as
   additional files in the same package (no subpackages, no speculative stubs).
4. **`--json`:** a clean, documented, stable schema. VS Code adapts to it; we
   are not bound by the current TypeScript interface (which today parses phase
   headers from stdout, not JSON).
5. **TOML merge:** **surgical line edits** that preserve the user's formatting
   and comments. There is no format-preserving TOML editor in Go (`go-toml/v2`
   reformats just like the already-vendored `BurntSushi/toml`), so we use
   BurntSushi only to READ the fetched values and validate structure, and apply
   targeted line edits to write. No new dependency.
6. **Target resolution scope:** serverless is the working happy path; **cluster
   and job compute resolution are also real** in this PR (SDK
   `GetByClusterId` → `SparkVersion` → DBR → envKey, with nearest-supported
   fallback). Unsupported runtimes return a clear error, never a crash or a
   hard stub.

## Architecture

### Package layout

```
cmd/dbconnect/
  dbconnect.go      New() *cobra.Command — "dbconnect" group, registers init + sync
  init.go           init subcommand: flag wiring + RunE -> pipeline.Run(Init)
  sync.go           sync subcommand: flag wiring + RunE -> pipeline.Run(Sync)
  output.go         text + --json rendering of the result/plan/errors

libs/dbconnect/
  pipeline.go       the shared phase pipeline (Mode = Init|Sync); orchestrates phases
  target.go         target resolution: flags + bundle target -> ResolvedTarget (envKey)
  envkey.go         DBR/serverless version -> envKey mapping (+ nearest-supported fallback)
  constraints.go    fetch constraint pyproject.toml (configurable base URL) + offline cache
  merge.go          surgical TOML merge of the 3 managed regions
  pkgmanager.go     PackageManager interface; uvManager implementation (uv.go)
  result.go         structured Result/Plan/Phase types (the --json schema)
```

**Rationale:** `cmd/dbconnect/` stays thin (Cobra wiring + rendering), mirroring
`cmd/psql/psql.go`. All logic lives in `libs/dbconnect/` so it is unit-testable
without a Cobra command. The `PackageManager` **interface** — not a directory
split — is what lets pip/conda land cleanly later; subpackages would create
import-cycle pressure (pipeline → pkgmanager → shared types) and would be
speculative scaffolding for deferred code.

**Registration:** one line in `cmd/cmd.go`:
`cli.AddCommand(dbconnect.New())`, in the `development` ("Developer Tools")
group alongside `psql`. Hand-written workflow command — does NOT touch
`.codegen/` or run `generate-cligen`.

### Control flow

`init` and `sync` build the same `Pipeline` and call `Run(ctx)`. They differ in
exactly one phase behavior (Phase 3/4: write-fresh vs merge-into-existing),
selected by `Mode`. Every other phase is shared and runs once.

## The phase pipeline

`Pipeline.Run(ctx)` executes the script's phases in order. Each phase is a
method that returns an error and appends a `PhaseResult` to the accumulating
`Result`. `Mode` (Init|Sync) only changes Phase 3/4.

| # | Phase | Go behavior | Δ from script |
|---|-------|-------------|---------------|
| 0 | Preflight | We *are* the CLI; auth comes from the resolved workspace client (`root.MustWorkspaceClient`). Discover uv from PATH + standard install locations (`~/.local/bin`, `$XDG_BIN_HOME`, Homebrew bins); bootstrap via the official installer if missing. Honor `UV_INDEX_URL` from `~/.config/pip/pip.conf` if unset. | No `databricks` binary probe. Auth via SDK, not `current-user me` shell-out. |
| 1 | Resolve target → envKey | Flags first (`--cluster`/`--serverless`/`--job`); else the bundle's configured target. Produce `ResolvedTarget{envKey, pythonVersion?}`. Preserve three-state messaging. | API calls, not file read. |
| 2 | Fetch constraints | GET `{baseURL}/{envKey}/pyproject.toml` via the CLI's HTTP client. Offline cache under the user cache dir; on network failure fall back to cache with a warning, else a clear error. | Configurable base URL + cache. |
| 3 | Baseline / idempotency | **Init:** write a fresh managed `pyproject.toml` (back up any existing to `.bak`). **Sync:** restore from `.bak` if present, else back it up, then merge. | Same idempotency model. |
| 4 | Merge managed regions | Surgical line edits to the 3 managed regions (see Merge section). | Robust merge, not regex. |
| 5 | Ensure Python | `PackageManager.EnsurePython(version)` — version from the resolved target, not hardcoded. | Version from target. |
| 6 | Provision | `PackageManager.Provision()` → `.venv` (uv: `uv sync`). | Interface seam. |
| 7 | Post-provision (pip seed) | `PackageManager.PostProvision()` — uv seeds pip into `.venv`; carries the script's full rationale comment (VS Code's `ms-python.vscode-python-envs` falls back to `python -m pip list` when its `uv --version` probe fails on the GUI PATH; uv venvs have no pip; `uv sync` strips pip, so seed runs after every sync). | uv-specific, behind the interface. |
| 8 | Validate | Assert `.venv` Python minor == target; `databricks-connect` matches the pin read from the fetched file. Populate `Result`. | Same asserts, structured output. |

**`--check` (dry-run):** runs phases 0–2 (read-only: discover, resolve, fetch),
then computes and prints the plan + the unified diff that phase 4 would write,
and stops before any mutation. Mutating phases (3–8) are gated on `!check`.

**Errors:** each phase wraps with `%w` and context. Structured errors carry a
stable `code` (e.g. `no_target_selected`, `cluster_unsupported`,
`constraint_fetch_failed`) so consumers branch on the code, never on message
text (repo rule: compare errors with sentinels, never `err.Error()` strings).

**Cancellation:** phases respect `ctx`; long shell-outs (uv) run via
`libs/process` with the context so Ctrl-C / VS Code cancel terminates them.

## Target resolution → envKey

### Stage A — pick the target (ordered precedence, early-return style)

1. `--cluster <id>` → SDK `w.Clusters.GetByClusterId(id)` → `SparkVersion` (the
   DBR string, e.g. `15.4.x-scala2.12`).
2. `--serverless <vN>` → serverless target, version `N`.
3. `--job <id>` → `w.Jobs.Get(id)`, read the job's compute (job cluster
   `SparkVersion`, or serverless if the task is serverless).
4. No flag → the **bundle's configured target** (loaded the same way bundle
   commands load it), read the selected target's compute.

Flags are mutually exclusive (`cmd.MarkFlagsMutuallyExclusive`), rejected at
parse time (repo rule: reject incompatible inputs early with an actionable
error).

### Stage B — three-state messaging (preserved from script lines 179–192)

- **serverless selected** → proceed.
- **cluster selected** → resolve its DBR → envKey (implemented, not a stub). If
  the DBR maps to no supported envKey, a clear "runtime X not yet supported"
  error.
- **nothing selected** (bundle has no compute target) → actionable error: "No
  compute target is selected. Select a cluster or serverless target, or pass
  --cluster/--serverless/--job."

### Stage C — version → envKey mapping (`envkey.go`)

- **Serverless:** `vN` → `serverless/serverless-vN`.
- **Cluster/job DBR:** parse major.minor from `SparkVersion`, map to an envKey
  via a small in-repo table, with **nearest-supported fallback** — if the exact
  DBR isn't in the table, pick the closest supported one and warn, naming both.

The table maps version → envKey *path* only. The constraint *pins* always come
from the fetched file, never from the table.

## Surgical TOML merge (`merge.go`)

**Goal:** touch only the 3 managed regions; preserve every byte the user owns
(comments, ordering, whitespace, their dependencies).

**Read side (BurntSushi, already vendored):** parse the *fetched* env file into
a struct to extract the managed values authoritatively:
- `project.requires-python` (string)
- the `databricks-connect` pin from `dependency-groups.dev`
- `tool.uv.constraint-dependencies` ([]string)

Also parse the *target* file with BurntSushi purely to validate it is
well-formed and to locate which regions exist before editing. We never write via
BurntSushi.

**Write side (structured line edits)** — three idempotent transforms:

1. **`requires-python`** — replace the value of the existing `requires-python =`
   line under `[project]`, preserving indentation; if `[project]` exists but the
   key doesn't, insert it.
2. **`databricks-connect` pin** — within `[dependency-groups].dev`, replace the
   existing `"databricks-connect..."` element in place (preserve indentation and
   trailing-comma style).
3. **`[tool.uv].constraint-dependencies`** — replace the whole managed block:
   drop any existing block we previously wrote, append a freshly rendered one.
   Bracketed with a discreet `# managed by databricks dbconnect` marker so
   re-merges replace exactly our block without clobbering a user's own
   `[tool.uv]` settings.

**Edge cases the tests must cover** (where the script's regex breaks):
- multiline vs single-line arrays for the dev group and constraints
- single vs double quotes, trailing commas, comment lines inside arrays
- `[project]` present but no `requires-python`
- no `[tool.uv]` yet vs a pre-existing one (ours or the user's)
- CRLF files (Windows) — normalize on read, restore on write

**Idempotency:** merging twice produces byte-identical output.

**`--check` diff:** the merge produces the new content in memory; `--check`
renders a unified diff (old vs new) and writes nothing.

## Output, flags & the `--json` schema

### Flags (shared by both subcommands)

| Flag | Type | Meaning |
|------|------|---------|
| `--cluster` | string | target a cluster (mutually exclusive) |
| `--serverless` | string | target serverless `vN` (mutually exclusive) |
| `--job` | string | target a job's compute (mutually exclusive) |
| `--check` | bool | dry-run: print plan + diff, mutate nothing |
| `--json` | bool | machine-readable output (wired via existing `cmdio` output plumbing) |
| `--constraint-source` | string | override the constraints base URL; default = `databricks-environments` repo. Also via env var. Advanced/hidden. |

### `--json` schema (the documented contract)

```jsonc
{
  "mode": "init" | "sync",
  "check": false,
  "target": {
    "kind": "serverless" | "cluster" | "job",
    "cluster_id": "…",
    "spark_version": "15.4.x-…",
    "env_key": "serverless/serverless-v4",
    "python_version": "3.12",
    "fallback": { "requested": "…", "resolved": "…" }
  },
  "constraints": {
    "source_url": "https://…/serverless-v4/pyproject.toml",
    "from_cache": false,
    "requires_python": ">=3.12",
    "databricks_connect": "databricks-connect~=17.2.0",
    "constraint_count": 42
  },
  "plan": {
    "pyproject_path": "/abs/pyproject.toml",
    "backup_path": "/abs/pyproject.toml.bak",
    "diff": "--- …\n+++ …\n@@ …",
    "changed_regions": ["requires-python", "databricks-connect", "tool.uv.constraint-dependencies"]
  },
  "phases": [
    {"name": "preflight", "status": "ok", "detail": "uv 0.5.1"},
    {"name": "provision", "status": "ok"}
  ],
  "result": {
    "status": "success" | "failed",
    "venv_path": "/abs/.venv",
    "python_version": "3.12",
    "databricks_connect_installed": "17.2.0"
  },
  "error": { "code": "no_target_selected", "message": "…" }
}
```

- Under `--check`, `plan` is computed and emitted; `phases` and `result` are
  empty/omitted.
- `error` is present only on failure; `error.code` is an enumerated, documented,
  stable set.

**Text output** mirrors the script's `=== Phase N ===` headers and final success
summary (so VS Code's phase-regex narration keeps working). `--json` emits the
struct above and suppresses decorative phase logging.

## Testing

### Unit tests (`libs/dbconnect/`, table-driven)

- **`merge_test.go`** — golden input pyproject + fetched constraints → expected
  merged output, covering every edge case above. Idempotency test (merge twice →
  identical). Diff test for `--check`.
- **`envkey_test.go`** — version→envKey incl. nearest-supported fallback and the
  unsupported-runtime error.
- **`target_test.go`** — precedence (flag > bundle), mutual exclusivity, the
  three states with their exact messages; SDK calls behind a small stubbed
  interface.
- **`constraints_test.go`** — fetch success, cache hit on network failure, hard
  failure with clear error; uses `httptest`.

### Acceptance tests (`acceptance/dbconnect/<case>/`)

Golden `output.txt` per the repo pattern (`acceptance/quickstart/`). uv and
network are unavailable in the sandbox, so these cover the deterministic,
mockable surface (resolution, messaging, merge, `--check`, `--json` shape) using
`libs/testserver` for the constraint fetch and stubbed compute:

- `serverless-check` — `--serverless v4 --check`: plan + diff, no mutation.
- `serverless-json` — `--json` shape on the resolve+plan path.
- `no-target` — the "nothing selected" error + message.
- `cluster-unsupported` — a DBR with no envKey → clear error.
- `flag-conflict` — `--cluster x --serverless y` rejected at parse.

Phases needing a live uv/`.venv` (5–8) are exercised by unit tests with the
package-manager interface stubbed. A full end-to-end uv run is validated
manually against the script ("diff against a live script run") and noted as a
manual check, not an acceptance test.

### Build / quality gate

`./task build`, `./task test`, `./task lint-q`, `./task fmt-q` all green.
`NEXT_CHANGELOG.md` entry under **CLI**: new `databricks dbconnect init` /
`sync` commands.

## Out of scope (this PR)

- pip & conda managers (interface only).
- Flipping the `--constraint-source` default to an official endpoint.
- Any new third-party dependency.

## Risks to verify during planning

1. **Cluster-DBR envKey data:** the `databricks-environments` repo currently
   publishes `serverless/serverless-vN` paths. Full cluster/job resolution needs
   real DBR→envKey paths (e.g. `cluster/dbr-15.4`). If the repo doesn't publish
   them yet, the envKey table is the gap — surface it explicitly and decide
   whether to seed the table from another source or narrow to the runtimes the
   repo actually publishes. The nearest-supported fallback + "runtime X not yet
   supported" error covers the rest gracefully.
2. **`--json` / `cmdio` wiring:** confirm the exact mechanism the CLI uses for
   JSON output (global `--output json` vs a local `--json` flag) and follow the
   existing convention rather than inventing a parallel switch.
