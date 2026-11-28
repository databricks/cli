---
description: Rules for the testing strategy of this repo
paths:
  - "**/*_test.go"
  - "acceptance/**"
  - "integration/**"
---

# Rules for the testing strategy of this repo

## Choosing a test level

**RULE: For user-visible CLI output and changes to the bundle mutator pipeline, reach for acceptance tests first.** They exercise the full pipeline and capture the exact output the user sees. When the coverage overlaps with an existing test, extend the existing acceptance directory instead of creating a new one. Unit tests are the right tool for pure functions and helpers you can test without mocking the whole world; don't add a unit test that only duplicates what an acceptance test already covers.

**RULE: Do not add new tests under `integration/`. That tree is deprecated.** When coverage needs a real workspace, write an acceptance test and set `Cloud = true` in its `test.toml`.

In unit tests, use `require` for preconditions that make the rest of the test meaningless and `assert` for the rest.

## Acceptance tests

Tests live in `acceptance/`, one directory per test, with sources (`script`, `test.toml`, `databricks.yml`, inputs) and generated outputs (`output.txt` and anything starting with `out`). `test.toml` options are documented in `acceptance/internal/config.go` and inherited from parent directories; the effective values are dumped to `out.test.toml`. `script.prepare` files from parent directories are prepended to the script.

- Run one test: `go test ./acceptance -run TestAccept/bundle/<path>/<to>/<folder>`. Append `EnvMatrix` values to select a variant: `-run 'TestAccept/.../DMS=true'` (multiple variables appear in alphabetical order).
- Useful flags: `-v`, `-tail` (follow output, needs `-v`), `-logrequests` (log HTTP traffic, needs `-v`).
- Run on cloud: `deco env run -i -n aws-prod-ucws -- <go test command>`.

**RULE: `Cloud = true` adds a cloud run; it never removes the local run.** Every test under `acceptance/` runs locally against the fake server in `libs/testserver`. The `Cloud*` settings (`CloudSlow`, `CloudEnvs`) only narrow the cloud run. What skips a test *locally* is `GOOS`, `RunsOnDbr` and `DATABRICKS_TEST_SELECT_CHANGED`. Since `Cloud` is inherited, read `out.test.toml`, not the leaf `test.toml`, for a test's effective settings.

**RULE: Never edit generated output files directly; regenerate them with `-update` (or `./task test-update`).** Exception: a predictable mass string replacement that is much cheaper than re-running the suite. After regenerating, run `./task fmt` and `./task lint`; if they modify files in `acceptance/`, fix the source and regenerate.

**RULE: All `EnvMatrix` variants must produce identical output files**, unless the test sets `EnvVaryOutput`. Do not run `-update` while variants diverge: the last variant overwrites the others. Instead, `-update` a single variant you consider correct and debug why the other differs. `EnvMatrix.<VAR> = []` removes an inherited matrix variable (e.g. `EnvMatrix.DMS = []` opts a bundle test out of the DMS run).

**RULE: In `test.toml`, write map-valued settings (`Env`, `EnvMatrix`, `EnvMatrixExclude`, `EnvRepl`, `GOOS`, `CloudEnvs`) in dotted form at the top, never under a table header.** Under `[EnvMatrix]`, every following key — including a top-level one like `Ignore` after a blank line — belongs to that table. `[[Repls]]` and `[[Server]]` keep the header form; put dotted keys above the first such header.

```toml
EnvMatrix.READPLAN = ["", "1"]
Ignore = ["databricks.yml"]
```

**RULE: Commit static test inputs into the test directory instead of creating them in `script`.** Add other artifacts the test creates to `Ignore` in `test.toml` (`.databricks` and `.gitignore` are already removed by `acceptance/script.cleanup`). Put common `test.toml` options in a parent directory.

**RULE: On Windows, Git Bash converts a leading-`/` argument (e.g. `/api/2.0/...`) into a Windows path.** Disable it per command: `MSYS_NO_PATHCONV=1 $CLI api get /api/2.0/...`; quoting does not help. Do not set `Env.MSYS_NO_PATHCONV = "1"` test-wide if the script runs any `acceptance/bin/*.py` helper — the helpers rely on the conversion.

### Script helpers

`acceptance/script.prepare` defines helpers available in every `script`; see it for the full list (`withdir`, `git-repo-init`, `uuid`, `sethome`, `venv_activate`, `as-test-sp`, `readplanarg`, `envsubst`, `unset_ci_env`, ...).

**RULE: Wrap commands whose invocation should appear in `output.txt` with `trace`**, and label phases of a long script with `title "TEXT"`.

**RULE: Assert an expected failure with `musterr CMD`; capture a tolerated non-zero exit with `errcode CMD`.** Not `! cmd` (passes silently on success) or `cmd || true` (hides the exit code).

**RULE: Use the `acceptance/bin/` helpers (on `PATH` during tests) instead of inline `jq`/`grep` pipelines.**

- `contains.py SUBSTR [!SUBSTR_NOT]`: passthrough filter asserting substrings are present (or absent).
- `print_requests.py //path [^//exclude] [--get] [--sort] [--unique] [--oneline] [--keep]`: print recorded requests (needs `RecordRequests = true`). Excludes GET unless `--get`; clears `out.requests.txt` unless `--keep` (don't `--keep` if another call follows: it double-prints).
- `replace_ids.py [-t TARGET]`, `read_id.py [-t TARGET] NAME`: add `[NAME_ID]` replacements for resource IDs from state.
- `add_repl VALUE REPLACEMENT`: add a custom output replacement. A value that starts or ends with a letter or digit matches only as a whole alphanumeric run, so `add_repl 339 X` does not change `33939489e027`, but does change `job_339` and `339,339`. Don't write `$ACC_REPLS` directly.
- `update_file.py FILENAME OLD NEW`, `find.py REGEX [--expect N]`, `diff.py A B` (diff with replacements applied), `print_state.py`, `edit_resource.py TYPE ID < script.py`, `gron.py` (flatten JSON; prefer `gron.py | grep field` over `jq` paths for single values).
- `plan_filter.py [RESOURCE_KEY]`: filter `bundle plan -o json` for a golden, dropping `remote_state` and the deployment stamp.

**RULE: Filter recorded requests with `print_requests.py`, never with a hand-written `jq 'select(...)' out.requests.txt` pipeline or a local shell function that reimplements it.** Use `jq` on `out.requests.txt` only for what `print_requests.py` can't express, such as filtering on request body content.

**RULE: Record whole plan entries with `plan_filter.py` unless the test asserts on `remote_state` itself.** `remote_state` is the backend's GET payload and breaks goldens on unrelated backend changes.

**RULE: When printing an API GET response into `output.txt`, project it to an allow-list of the fields the test asserts.** Never dump the full payload or `jq 'del(...)'` today's volatile fields: the backend keeps adding fields and cloud runs break. Define a `<resource>_fields()` helper with `jq` in the directory's `script.prepare` (precedent: `acceptance/bundle/resources/postgres_*/script.prepare`); append `| with_entries(select(.value != null))` to drop optional fields.

**RULE: Route noisy or non-deterministic output to `LOG.<name>` instead of `output.txt` or `/dev/null`.** `LOG.*` files are shown under `go test -v` but excluded from the diff (see `acceptance/selftest/log/`).

### Test server

**RULE: Model API behavior in `libs/testserver/`, not in per-test `[[Server]]` response stubs.** When a test needs the fake server to validate input or return an error (404 on a missing parent, 409 on a duplicate, ...), extend the stateful handler so every test sees the real contract. Reserve `[[Server]]` for routes the testserver does not model at all and for responses a stateful handler can't express (for transient faults and disconnects, prefer the `fault.py` / kill helpers).

### Template tests

**RULE: Materialized templates under `acceptance/bundle/templates` are generated from `libs/template/templates/`.** Regenerate with `./task test-update-templates`; fix lint or format issues in the template source, not the output.
