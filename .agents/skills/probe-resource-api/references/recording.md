# Record API behavior in acceptance tests

The harness records requests against both the local fake and real workspaces. It can also log responses for diagnosis. These captures help implement stateful handlers in `libs/testserver`; there is no general response replay or generator that turns a capture into a testserver implementation.

Read the [testing rules](../../../../.agents/rules/testing.md) before adding a test. The implementation is in [server preparation](../../../../acceptance/internal/prepare_server.go), [the harness](../../../../acceptance/acceptance_test.go), and [the cloud proxy](../../../../libs/testproxy/server.go).

## Configure and run a capture

Place settings in the test's `test.toml` or an appropriate parent:

```toml
Cloud = true
RecordRequests = true

# A raw API contract test does not exercise deployment history.
EnvMatrix.DMS = []

# Needed for literal leading-/ API paths on Windows Git Bash.
Env.MSYS_NO_PATHCONV = "1"
```

`Cloud = true` adds a real-workspace run when `CLOUD_ENV` is configured; the same test still runs locally. Inspect generated `out.test.toml` for inherited settings. `CloudSlow` only skips an enabled cloud test under `-short`; `CloudEnvs` can restrict which clouds run it. `RunsOnDbr` and `RecordRequests` cannot both be true. See [the config fields](../../../../acceptance/internal/config.go).

For a concrete existing raw API test:

```sh
deco env run -i -n aws-cli -- go test ./acceptance \
  -run '^TestAccept/bundle/resources/postgres_projects/factcheck/update_mask_spec$' \
  -count=1 -v -tail -logrequests
```

The installed `databricks-cli-dev:run-cloud-tests` skill uses isolated dedicated CLI environments: `aws-cli`, `azure-cli`, or `gcp-cli`. Select the cloud on which the resource is available. Confirm verbose output shows the selected test ran; an unavailable cloud or `-short` can skip it. `-count=1` avoids accepting a cached test result.

Run the same case locally after implementing the fake:

```sh
go test ./acceptance \
  -run '^TestAccept/bundle/resources/postgres_projects/factcheck/update_mask_spec$' \
  -count=1 -v -tail -logrequests
```

Replace the test path with the case being developed. Add `-update` to regenerate intentional golden changes, inspect the diff, then rerun without it. Follow the testing rules for matrix variants; do not regenerate divergent variants into the same output files.

Both capture modes start the authenticated cloud proxy automatically. `Proxy = true` additionally routes a **local** test through the proxy to check that topology; see [recording selftests](../../../../acceptance/selftest/record_cloud/test.toml).

## Requests and responses are different artifacts

| Mechanism | What it captures | How to use it |
| --- | --- | --- |
| `RecordRequests = true` | Requests arriving at the server/proxy | Print selected requests into `output.txt` or a deliberate `out.*` golden |
| `-logrequests -v` | Request and response diagnostic text through `t.Logf` | Retain the outer test stdout/stderr privately when investigating response behavior |

The request buffer is `$OUT_REQUESTS`, currently `<outputDir>/out.requests.txt`. It contains concatenated indented JSON objects, separated by whitespace. It is neither a JSON array nor newline-delimited JSON:

```json
{
  "method": "PATCH",
  "path": "/api/2.0/postgres/projects/example",
  "q": {"update_mask": "spec.display_name"},
  "body": {"spec": {"display_name": "new"}}
}
{
  "method": "GET",
  "path": "/api/2.0/postgres/projects/example"
}
```

The record has `method` and `path`, plus optional `q`, `body`, `raw_body`, and `headers`. Query values can be strings or arrays. JSON bodies are parsed; other bodies use `raw_body`. Multipart bodies are normalized, with binary or large parts summarized by size. There is **no response or status field**.

`IncludeRequestHeaders = ["Header-Name"]` includes only the named headers; none are included by default. Use canonical header spelling. Body contents are not automatically redacted.

## Select requests with the existing helper

Use [print_requests.py](../../../../acceptance/bin/print_requests.py) inside the acceptance script:

```sh
# Assert the exact update payload in the main output.
trace print_requests.py //api/2.0/postgres/projects --method PATCH

# Or write a deliberate golden file for this phase.
print_requests.py //api/2.0/postgres/projects > out.requests.update.json
```

These are alternatives for a given buffer: the first call normally consumes it. The helper reads `$OUT_REQUESTS`, understands its concatenated format, and deletes the entire buffer after processing unless `--keep` is set. Use `$OUT_REQUESTS` when clearing a phase's capture; do not assume it is in the script's current directory.

Useful options:

- `//path` matches a path substring; `^//path` excludes one. Multiple positive filters are ORed. The extra slash avoids Git Bash path conversion.
- GET requests are excluded by default; `--get` includes them. `--method PATCH` selects a method and overrides that default.
- `--sort` stabilizes an order-independent set. Preserve order when asserting a lifecycle sequence.
- `--unique` removes consecutive duplicate records. Do not use it when retries are the assertion.
- DMS requests under `/api/2.0/bundle` are excluded unless `--dms` is passed.
- `--fname PATH` selects a different capture. `--keep` retains the buffer, so a later call can print earlier requests again.

The helper's older docstrings mention `out.requests.json`; its current default is `$OUT_REQUESTS`.

## Retention and redaction

Harness-owned files live in a sibling `outputDir`, outside the test working directory, to keep bundle sync from uploading them. The raw request buffer is explicitly excluded from golden comparison. It is not automatically committed by `-update`.

`-keeptmp` retains the test working directory, but **does not retain `outputDir`**, which still uses `t.TempDir()`. Before the test ends, print the selected evidence into an intentional `out.*` artifact. Keep complete exploratory logs outside the repository when they are needed for diagnosis.

For example, capture the outer command's output in a private temporary directory:

```sh
probe_log_dir=$(mktemp -d)
deco env run -i -n aws-cli -- go test ./acceptance \
  -run '^TestAccept/bundle/resources/postgres_projects/factcheck/update_mask_spec$' \
  -count=1 -v -tail -logrequests > "$probe_log_dir/http.log" 2>&1
```

`-logrequests` prints available headers and body contents without redaction. Structured request recording also retains body contents without redaction; JSON parsing and multipart normalization mean it is not byte-for-byte capture. Harness replacements apply later to compared output; they do not sanitize the raw recording or diagnostic log. Review and remove credentials, secret values, and unrelated user data before sharing or committing evidence. Prefer synthetic inputs and response allowlists for permanent tests.

## Limits of the cloud proxy

The proxy is useful for semantic request/body checks, but its output is not a complete wire transcript:

- It reports **200 for every successful upstream response**. Do not use it to establish whether the service returned 201, 202, or 204.
- It forwards only `X-Databricks-Org-Id` from upstream response headers; the logged response object does not contain a complete upstream header capture. It cannot establish ETag-header or Retry-After behavior.
- It uses the SDK internally, so SDK retries between proxy and service are not recorded as separate incoming requests. The request count is not necessarily the upstream attempt count.
- It forwards only the first value of each query parameter and request header. An incoming recording with repeated query values does not prove those values reached the service.
- API errors retain the upstream status and raw error body. Other transport failures become a proxy-generated 500.

For an exact status, header, retry, or repeated-query question, capture those properties through direct authenticated transport diagnostics and label that evidence separately. `$CLI api` preserves JSON response content but does not itself print response status or headers.

## Cleanup and useful precedents

Register an EXIT cleanup trap before creating test resources. `script.cleanup` is appended after the main script and **does not execute if the main script fails**. Record created IDs promptly, delete children before parents, use bounded waits, and retain cleanup failures in `LOG.cleanup`. Verify the final resource state; isolation and CI sweeps are not evidence that cleanup succeeded.

- [Postgres factcheck scripts](../../../../acceptance/bundle/resources/postgres_projects/factcheck/update_mask_spec/script) call the API without bundle deployment, print the exact PATCH, and read back after errors: a whole-spec update can partly write before failing. [PR #6767](https://github.com/databricks/cli/pull/6767) documents the real-workspace validation. Use [project_fields](../../../../acceptance/bundle/resources/postgres_projects/script.prepare) as a response-allowlist example.
- [Vector Search pending deletion](../../../../acceptance/bundle/resources/vector_search_indexes/recreate/pending_deletion/script) retains both CREATE attempts; `--unique` would erase the retry assertion. [PR #6143](https://github.com/databricks/cli/pull/6143) explains why GET disappearance precedes name reuse. This is a deterministic local regression case for observed service behavior.
- [Genie description clearing, PR #6789](https://github.com/databricks/cli/pull/6789) verified that an explicit empty string clears a field while omission preserves it, then changed both adapter serialization and the fake. The [current handler](../../../../libs/testserver/genie_spaces.go) checks JSON key presence.

`capture_tfstate.py` and `replay_tfstate.py` serve frozen Terraform migration fixtures against the local fake. They neither replay captured API responses nor generate handlers, and the capture workflow requires the historical Terraform engine. Do not use them for onboarding a new resource.
