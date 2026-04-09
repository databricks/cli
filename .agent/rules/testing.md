---
description: Rules for the testing strategy of this repo
---

# Rules for the testing strategy of this repo

## Test Types

- **Unit tests**: Standard Go tests alongside source files
- **Integration tests**: `integration/` directory, requires live Databricks workspace
- **Acceptance tests**: `acceptance/` directory, uses mock HTTP server

Each file like process_target_mode_test.go should have a corresponding test file
like process_target_mode_test.go. If you add new functionality to a file,
the test file should be extended to cover the new functionality.

Tests should look like the following:
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

Notice that:
- Tests are often in the same package but suffixed with _test.
- The test names are prefixed with Test and are named after the function or module they are testing.
- 'require' and 'require.NoError' are used to check for things that would cause the rest of the test case to fail.
- 'assert' is used to check for expected values where the rest of the test is not expected to fail.

When writing tests, please don't include an explanation in each
test case in your responses. I am just interested in the tests.

Use table-driven tests when testing multiple similar cases (e.g., different inputs producing different outputs). Reviewers prefer this pattern over repeating near-identical test functions.

## Acceptance Tests

- Located in `acceptance/` with nested directory structure.
- Each test directory contains `databricks.yml`, `script`, and `output.txt`.
- Source files: `test.toml`, `script`, `script.prepare`, `databricks.yml`, etc.
- Tests are configured via `test.toml`. Config schema and explanation is in `acceptance/internal/config.go`. Config is inherited from parent directories. Certain options are also dumped to `out.test.toml` so that inherited values are visible on PRs.
- Generated output files start with `out`: `output.txt`, `out.test.toml`, `out.requests.txt`. Never edit these directly — use `-update` to regenerate. Exception: mass string replacement when the change is predictable and much cheaper than re-running the test suite.
- Run a single test: `go test ./acceptance -run TestAccept/bundle/<path>/<to>/<folder>`
- Run a specific variant by appending EnvMatrix values to the test name: `go test ./acceptance -run 'TestAccept/.../DATABRICKS_BUNDLE_ENGINE=direct'`. When there are multiple EnvMatrix variables, they appear in alphabetical order.
- Useful flags: `-v` for verbose output, `-tail` to follow test output (requires `-v`), `-logrequests` to log all HTTP requests/responses (requires `-v`).
- Run tests on cloud: `deco env run -i -n aws-prod-ucws -- <go test command>` (requires `deco` tool and access to test env).
- Use `-update` flag to regenerate expected output files. When a test fails because of stale output, re-run with `-update` instead of editing output files.
- All EnvMatrix variants share the same output files — they MUST produce identical output. Exception: filenames containing `$DATABRICKS_BUNDLE_ENGINE` (e.g. `output.direct.txt`) are recorded per-engine.
- `-update` with divergent variant outputs is destructive: overwrites with last variant, breaking others. To debug: run a single variant you consider correct with `-update`, then debug the other variant to find why it diverges.
- `test.toml` is inherited — put common options into a parent directory.
- Add test artifacts (e.g. `.databricks`) to `Ignore` in `test.toml`.
- `script.prepare` files from parent directories are concatenated into the test script — use them for shared bash helpers.

**Helper scripts** in `acceptance/bin/` are available on `PATH` during test execution:
- `contains.py SUBSTR [!SUBSTR_NOT]` — passthrough filter (stdin→stdout) that checks substrings are present (or absent with `!` prefix). Errors are reported on stderr.
- `print_requests.py //path [^//exclude] [--get] [--sort] [--keep]` — print recorded HTTP requests matching path filters. Requires `RecordRequests=true` in `test.toml`. Clears `out.requests.txt` afterwards unless `--keep`. Use `--get` to include GET requests (excluded by default). Use `^` prefix to exclude paths.
- `replace_ids.py [-t TARGET]` — read deployment state and add `[NAME_ID]` replacements for all resource IDs.
- `read_id.py [-t TARGET] NAME` — read ID of a single resource from state, print it, and add a `[NAME_ID]` replacement.
- `add_repl.py VALUE REPLACEMENT` — add a custom replacement (VALUE will be replaced with `[REPLACEMENT]` in output).
- `update_file.py FILENAME OLD NEW` — replace all occurrences of OLD with NEW in FILENAME. Errors if OLD is not found. Cannot be used on `output.txt`.
- `find.py REGEX [--expect N]` — find files matching regex in current directory. `--expect N` to assert exact count.
- `diff.py DIR1 DIR2` or `diff.py FILE1 FILE2` — recursive diff with test replacements applied.
- `print_state.py [-t TARGET] [--backup]` — print deployment state (terraform or direct).
- `edit_resource.py TYPE ID < script.py` — fetch resource by ID, execute Python on it (resource in `r`), then update it. TYPE is `jobs` or `pipelines`.
- `gron.py` — flatten JSON into greppable discrete assignments (simpler than `jq` for searching JSON).
- `jq` is also available for JSON processing.

**Update workflow**: Run `make test-update` to regenerate outputs. Then run `make fmt` and `make lint` — if these modify files in `acceptance/`, there's an issue in source files. Fix the source, regenerate, and verify lint/fmt pass cleanly.

**Template tests**: Tests in `acceptance/bundle/templates` include materialized templates in output directories. These directories follow the same `out` convention — everything starting with `out` is generated output. Sources are in `libs/template/templates/`. Use `make test-update-templates` to regenerate. If linters or formatters find issues in materialized templates, do not fix the output files — fix the source in `libs/template/templates/`, then regenerate.
