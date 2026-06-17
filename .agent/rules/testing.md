---
description: Rules for the testing strategy of this repo
paths:
  - "**/*_test.go"
  - "acceptance/**"
  - "integration/**"
---

# Rules for the testing strategy of this repo

## Test Types

- **Unit tests**: Standard Go tests alongside source files
- **Integration tests**: `integration/` directory, requires live Databricks workspace
- **Acceptance tests**: `acceptance/` directory, uses mock HTTP server

## Choosing a test level

**RULE: For user-visible CLI output and changes to the bundle mutator pipeline, reach for acceptance tests first.** They exercise the full pipeline and capture the exact output the user sees. `cmd/...` commands and anything under the mutator pipeline (`bundle/config/mutator/...` and `bundle/mutator/...`) are the strongest candidates. When the coverage overlaps with an existing test, extend the existing acceptance directory instead of creating a new one.

**Unit tests are still the right tool** for pure functions, utility code, parsing/formatting helpers, and anything you can meaningfully test without mocking the whole world. Don't force a unit into an acceptance test just because the code lives under `cmd/`, and don't add a mutator unit test that only duplicates what an acceptance test already covers.

When in doubt: would the test fail in a useful way if a mutator earlier in the pipeline changed? If yes, the test wants to be an acceptance test.

## Unit tests

**RULE: Each source file should have a corresponding test file.** If you add new functionality to a file, extend the test file to cover it.

**RULE: Place tests in the same package but with a `_test` suffix.** Test names start with `Test` and describe the function or module under test.

**RULE: Use `require` for preconditions that would make the rest of the test meaningless on failure.** Use `assert` for expected values where the test can keep running after a failure.

```go
package mutator_test

func TestApplySomeChangeReturnsDiagnostics(t *testing.T) {
	...
}

func TestApplySomeChangeFixesThings(t *testing.T) {
	ctx := t.Context()
	b, err := ...some operation...
	require.NoError(t, err)
	...
	assert.Equal(t, ...)
}
```

**RULE: Use table-driven tests for multiple similar cases.** Reviewers prefer this pattern over repeating near-identical test functions when the inputs differ but the logic is the same.

**RULE: If a value is shared across tests and they must change together, extract it to a package-level `const` or `var`.** Think shared fixtures, identifiers that must stay in sync across tests, and expected error messages that the tests verify as a set. Repeated literals that happen to be the same (e.g. a header name like `"Authorization"` appearing inline in many tests) are fine to leave inline; forcing extraction there hurts readability without buying anything.

When writing tests, don't include an explanation in each test case in your responses. Only the tests are needed.

## Acceptance Tests

**RULE: Never edit generated acceptance output files directly.** Files named `output.txt`, `out.test.toml`, `out.requests.txt`, or anything starting with `out` are regenerated. Use the `-update` flag to regenerate them.

Exception: mass string replacement when the change is predictable and much cheaper than re-running the test suite.

**RULE: All `EnvMatrix` variants MUST produce identical output files.** Filenames containing `$DATABRICKS_BUNDLE_ENGINE` (e.g. `output.direct.txt`) are the only per-engine exception.

**RULE: Do not run `-update` while a divergent variant exists.** It is destructive: it overwrites with the last variant and breaks the others. To debug: run a single variant you consider correct with `-update`, then debug the other variant to find why it diverges.

**RULE: Put common `test.toml` options in a parent directory.** Config is inherited from parents.

**RULE: Add test artifacts (e.g. `.databricks`) to `Ignore` in `test.toml`.**

**RULE: Commit static test inputs into the acceptance test directory; do not create them in `script` at test time.** If a file's content is dynamic, generating it in `script` is fine. For everything else, check it in and let the test read it directly; you won't need an `Ignore` entry because there's nothing to clean up.

GOOD:

```
acceptance/cmd/fs/cp/file-to-dir/
  script        # $CLI fs cp local.txt dbfs:/path/
  test.toml
  local.txt     # committed input
  output.txt
```

BAD:

```
acceptance/cmd/fs/cp/file-to-dir/
  script        # echo "contents" > local.txt; $CLI fs cp local.txt dbfs:/path/; rm local.txt
  test.toml     # Ignore = ["local.txt"]
  output.txt
```

**RULE: When output genuinely diverges between engines (terraform vs direct), split only the diverging file into per-engine variants.** Keep the rest of the output unified. Files named `output.$DATABRICKS_BUNDLE_ENGINE.txt` or `out.requests.$DATABRICKS_BUNDLE_ENGINE.json` are the allowed per-engine form.

If the only reason for divergence is a server-side default that one engine sets and the other doesn't, set the field explicitly in `databricks.yml` so both engines produce identical output. Don't paper over it with per-engine files.

**RULE: On Windows, Git Bash auto-converts a leading-`/` path argument (e.g. `/api/2.0/...`) into a Windows path, so `$CLI` sees the wrong path and the testserver 404s.** Set `MSYS_NO_PATHCONV = "1"` in the test directory's `test.toml` under `[Env]`. Quoting the argument in bash does NOT help — the conversion is done by the Windows binary's argument processing. Precedent: `acceptance/cmd/workspace/export-dir-*/test.toml`.

**RULE: `EnvMatrix.<VAR> = []` removes that variable from the inherited matrix** (see `ExpandEnvMatrix` in `acceptance/internal/config.go`). The root `test.toml` matrixes `DATABRICKS_BUNDLE_ENGINE = [terraform, direct]`, so a non-bundle test opts out of both engine runs with `EnvMatrix.DATABRICKS_BUNDLE_ENGINE = []`. The `out.test.toml` snapshot of inherited values is generated and committed by design.

**RULE: If a test's `out.test.toml` is still in the older `[EnvMatrix]` block format, a regen rewrites it to the inline form and the post-test `git diff --exit-code` check fails** ("out.test.toml files that are out of date"). Regenerate just those files with `go test ./acceptance -run "^TestAccept$" -only-out-test-toml`, then commit.

### Reference

- Tests live in `acceptance/` with a nested directory structure.
- Each test directory contains `databricks.yml`, `script`, and `output.txt`.
- Source files: `test.toml`, `script`, `script.prepare`, `databricks.yml`, etc.
- Tests are configured via `test.toml`. Config schema and explanation is in `acceptance/internal/config.go`. Certain options are also dumped to `out.test.toml` so that inherited values are visible on PRs.
- Run a single test: `go test ./acceptance -run TestAccept/bundle/<path>/<to>/<folder>`
- Run a specific variant by appending `EnvMatrix` values to the test name: `go test ./acceptance -run 'TestAccept/.../DATABRICKS_BUNDLE_ENGINE=direct'`. When there are multiple `EnvMatrix` variables, they appear in alphabetical order.
- Useful flags: `-v` for verbose output, `-tail` to follow test output (requires `-v`), `-logrequests` to log all HTTP requests/responses (requires `-v`).
- Run tests on cloud: `deco env run -i -n aws-prod-ucws -- <go test command>` (requires `deco` tool and access to test env).
- `script.prepare` files from parent directories are concatenated into the test script. Use them for shared bash helpers.

### Built-in shell helpers

`acceptance/script.prepare` defines shell helpers that are in scope for every `script`. Prefer them over hand-rolled equivalents — they keep `output.txt` consistent across tests and make intent obvious to the next reader.

- `trace CMD...`: print `>>> CMD` to stderr, then run CMD. Wrap commands whose invocation should appear in `output.txt` so the captured output explains what produced it. Leading `KEY=value` arguments are exported for the command (e.g. `trace FOO=bar $CLI ...`).
- `title "TEXT"`: print a `=== TEXT` section header. Use `\n` escapes for spacing. Use it to label the phases of a multi-step script.
- `errcode CMD...`: run CMD; if it exits non-zero, append `Exit code: N` to the output but let the script continue (scripts run under `bash -e`, which would otherwise abort). Use when a command is *allowed* to fail and later steps must still run.
- `musterr CMD...`: run CMD and fail the whole test if it *succeeds*; on the expected failure the script continues. Use to assert a command must error.
- `withdir DIR CMD...`: run CMD with the working directory set to DIR, restoring it afterwards.
- `git-repo-init`: initialize a deterministic git repo (fixed user/email, no hooks) and commit `databricks.yml`.
- `uuid`, `sethome DIR`, `venv_activate`, `as-test-sp CMD...`, `readplanarg FILE`, `envsubst`: see `acceptance/script.prepare` for the full list and exact semantics.

**RULE: Wrap commands whose invocation should be visible in `output.txt` with `trace`.** The `>>> ...` line ties each block of output to the command that produced it; without it the output is an unlabeled wall of text.

**RULE: Assert an expected failure with `musterr`, not `! cmd` or a bare `errcode`.** Only `musterr` fails the test when the command unexpectedly *succeeds*. `! cmd` is exempt from `set -e` and silently passes on success; `errcode` is for *tolerated* failures, not required ones.

**RULE: Capture a tolerated non-zero exit with `errcode`, not `cmd || true`.** `errcode` records `Exit code: N` in the output so the failure stays visible and asserted; `|| true` hides it entirely.

### Helper scripts

**RULE: Use the `acceptance/bin/` helpers before reaching for inline `jq` or `grep` pipelines.** When a test needs to filter recorded requests, assert a substring is or isn't present, or register a dynamic replacement, the helpers handle sorting, URL query normalization, redaction hooks, and cross-platform path issues. Inline `jq` in an acceptance script is brittle and hard to read.

GOOD:

```bash
trace $CLI bundle plan | contains.py "Plan: 0 to add, 0 to delete, 1 to update"
trace print_requests.py //api/2.0/apps
echo "$deployment_id:DEPLOYMENT_ID" >> ACC_REPLS
```

BAD:

```bash
{ trace jq 'select(.method == "POST" and .path == "/api/2.0/apps")' out.requests.txt; } || true
```

Available on `PATH` during test execution (from `acceptance/bin/`):

- `contains.py SUBSTR [!SUBSTR_NOT]`: passthrough filter (stdin→stdout) that checks substrings are present (or absent with `!` prefix). Errors are reported on stderr.
- `print_requests.py //path [^//exclude] [--get] [--sort] [--unique] [--oneline] [--keep]`: print recorded HTTP requests matching path filters. Requires `RecordRequests = true` in `test.toml`. Excludes GET by default (`--get` includes them); clears `out.requests.txt` afterwards (`--keep` retains it). `^` prefix excludes a path; multiple positive filters are OR'd together. `--sort` orders output deterministically (use when the request set is order-independent), `--unique` collapses consecutive duplicates (e.g. repeated polls), `--oneline` prints one request per line.
- `replace_ids.py [-t TARGET]`: read deployment state and add `[NAME_ID]` replacements for all resource IDs.
- `read_id.py [-t TARGET] NAME`: read ID of a single resource from state, print it, and add a `[NAME_ID]` replacement.
- `add_repl.py VALUE REPLACEMENT`: add a custom replacement (VALUE will be replaced with `[REPLACEMENT]` in output).
- `update_file.py FILENAME OLD NEW`: replace all occurrences of OLD with NEW in FILENAME. Errors if OLD is not found. Cannot be used on `output.txt`.
- `find.py REGEX [--expect N]`: find files matching regex in current directory. `--expect N` asserts an exact count.
- `diff.py DIR1 DIR2` or `diff.py FILE1 FILE2`: recursive diff with test replacements applied.
- `print_state.py [-t TARGET] [--backup]`: print deployment state (terraform or direct).
- `edit_resource.py TYPE ID < script.py`: fetch resource by ID, execute Python on it (resource in `r`), then update it. TYPE is `jobs` or `pipelines`.
- `gron.py`: flatten JSON into greppable discrete assignments (simpler than `jq` for searching JSON).
- `jq` is also available for JSON processing.

**RULE: Prefer `gron.py | grep <field>` over inline `jq` paths for single-value lookups.** The gron output prints the JSON path inline, so the test log explains itself.

**RULE: Don't pass `--keep` to `print_requests.py` if a later `print_requests.py` call follows.** The buffer accumulates, so the second call double-prints the earlier requests.

**RULE: Filter recorded requests with `print_requests.py`, never with a hand-written `jq 'select(...)' out.requests.txt` pipeline** — inline, or hidden inside a local `print_requests()` shell function. The helper already excludes GET, normalizes query strings, optionally sorts, and deletes `out.requests.txt` afterwards. A copy-pasted `jq` wrapper reimplements all of that, drifts from the canonical output format, and is the single most common acceptance-test anti-pattern in this repo. Wrapping `print_requests.py` *itself* in a local function is fine — e.g. to send each variant's output to its own `out.requests.<name>.json`. Reach for `jq` on `out.requests.txt` only for what `print_requests.py` genuinely can't express: filtering on request *body* content, or deleting noisy body fields (prefer a `Repls` entry in `test.toml` even then).

GOOD:

```bash
trace print_requests.py //pipelines --sort
```

BAD:

```bash
print_requests() {
    jq --sort-keys 'select(.method != "GET" and (.path | contains("/pipelines")))' < out.requests.txt
    rm out.requests.txt
}
trace print_requests
```

**RULE: Route noisy or non-deterministic command output to `LOG.<name>` instead of `output.txt` or `/dev/null`.** `LOG.*` files are visible under `go test -v` but excluded from the diff — see `acceptance/selftest/log/`. Use `&> LOG.<name>` to capture both streams (then `contains.py` to assert invariants like `'!panic' '!internal error'`), or `2>>LOG.<name>` for cleanup-step stderr you'd otherwise drop to `/dev/null`.

### Test server

Acceptance tests run against an in-process fake of the Databricks API in `libs/testserver/` (`FakeWorkspace` and the per-service handler files). The fake keeps real in-memory state and returns the same errors the backend does: 404 on a missing parent, 409 on a duplicate create, 400 on a missing required field, and so on. `test.toml` can also stub a single route with a canned response:

```toml
[[Server]]
Pattern = "POST /api/2.2/jobs/create"
Response.StatusCode = 400
Response.Body = '''{"error_code": "INVALID_PARAMETER_VALUE", "message": "..."}'''
```

**RULE: Model API behavior in `libs/testserver/`, not in per-test `[[Server]]` response stubs.** When a test needs the fake server to validate input or return an error, add or extend the handler in `libs/testserver/` so the behavior is stateful and shared by every test. A `[[Server]]` stub hijacks the route with a static response that ignores request state, diverges from the real API, and only helps the one test that declares it — so the next test re-stubs the same error and the fake never converges on the real contract.

GOOD: teach the create handler in `libs/testserver/postgres.go` to return 404 when the referenced role does not exist, so every test that creates a database against a missing role observes the real error.

BAD: add `[[Server]]` with `Pattern = "POST .../databases"` and `Response.StatusCode = 404` to a single test's `test.toml` to fake that same error.

Reserve `[[Server]]` for routes the testserver does not model at all (a one-off endpoint exercised by a single test) and for injecting a response a stateful handler genuinely can't express (for transient faults and forced disconnects, prefer the `fault.py` / kill helpers instead).

### Update workflow

**RULE: Run `./task test-update` to regenerate outputs, then `./task fmt` and `./task lint`.** If fmt or lint modify files in `acceptance/`, there's an issue in the source files. Fix the source, regenerate, and verify fmt/lint pass cleanly.

### Template tests

Tests in `acceptance/bundle/templates` include materialized templates in output directories. These directories follow the same `out` convention: everything starting with `out` is generated output. Sources are in `libs/template/templates/`.

**RULE: Use `./task test-update-templates` to regenerate materialized templates.** If linters or formatters find issues in materialized templates, do not fix the output files; fix the source in `libs/template/templates/` and regenerate.
