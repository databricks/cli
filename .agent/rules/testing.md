---
description: Rules for the testing strategy of this repo
---

# Rules for the testing strategy of this repo

## Test Types

- **Unit tests**: Standard Go tests alongside source files
- **Integration tests**: `integration/` directory, requires live Databricks workspace
- **Acceptance tests**: `acceptance/` directory, uses mock HTTP server

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

**RULE: Extract values used in 2 or more tests into package-level `const` or `var` at the top of the test file.** Applies to URIs, IDs, names, error messages, expected results, and any complex test fixtures. Avoids drift when the value changes and makes the intent of each test clearer.

When writing tests, don't include an explanation in each test case in your responses. Only the tests are needed.

## Acceptance Tests

**RULE: Never edit generated acceptance output files directly.** Files named `output.txt`, `out.test.toml`, `out.requests.txt`, or anything starting with `out` are regenerated. Use the `-update` flag to regenerate them.

Exception: mass string replacement when the change is predictable and much cheaper than re-running the test suite.

**RULE: All `EnvMatrix` variants MUST produce identical output files.** Filenames containing `$DATABRICKS_BUNDLE_ENGINE` (e.g. `output.direct.txt`) are the only per-engine exception.

**RULE: Do not run `-update` while a divergent variant exists.** It is destructive: it overwrites with the last variant and breaks the others. To debug: run a single variant you consider correct with `-update`, then debug the other variant to find why it diverges.

**RULE: Put common `test.toml` options in a parent directory.** Config is inherited from parents.

**RULE: Add test artifacts (e.g. `.databricks`) to `Ignore` in `test.toml`.**

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
