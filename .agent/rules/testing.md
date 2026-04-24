---
description: Rules for the testing strategy of this repo
---

# Rules for the testing strategy of this repo

## Test Types

- **Unit tests**: Standard Go tests alongside source files
- **Integration tests**: `integration/` directory, requires live Databricks workspace
- **Acceptance tests**: `acceptance/` directory, uses mock HTTP server

## Choosing a test level

**RULE: For user-visible CLI output and mutator pipeline behavior, reach for acceptance tests first.** They exercise the full pipeline and capture the exact output the user sees. `cmd/...` commands and changes to `bundle/config/mutator/...` are the strongest candidates. Before adding a new acceptance test file, see if an existing nearby test can be extended.

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
- `print_requests.py //path [^//exclude] [--get] [--sort] [--keep]`: print recorded HTTP requests matching path filters. Requires `RecordRequests=true` in `test.toml`. Clears `out.requests.txt` afterwards unless `--keep`. Use `--get` to include GET requests (excluded by default). Use `^` prefix to exclude paths.
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

### Update workflow

**RULE: Run `make test-update` to regenerate outputs, then `make fmt` and `make lint`.** If fmt or lint modify files in `acceptance/`, there's an issue in the source files. Fix the source, regenerate, and verify fmt/lint pass cleanly.

### Template tests

Tests in `acceptance/bundle/templates` include materialized templates in output directories. These directories follow the same `out` convention: everything starting with `out` is generated output. Sources are in `libs/template/templates/`.

**RULE: Use `make test-update-templates` to regenerate materialized templates.** If linters or formatters find issues in materialized templates, do not fix the output files; fix the source in `libs/template/templates/` and regenerate.
