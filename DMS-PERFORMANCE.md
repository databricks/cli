# Local DMS performance experiments

Experimental branch: `denis/dms-fast-acceptance`.

The subject is the production DMS implementation. Jobs and other workspace dependencies
can be stateful test fakes. Do not replace DMS with the Go testserver implementation,
skip CLI operations, or change the invariant's assertions to improve timing.

## Reproduction

From the CLI root, with a running local DMS fixture whose connection settings are in
the enclosing Universe checkout's `bundle/metadata-service/test/.lite/cli-endpoints.json`:

```sh
PATH=/home/denis.bilenko/.local/bin:$PATH DMS_CLI_TRACE=1 \
  go test ./acceptance \
  -run 'TestAccept/bundle/invariant/no_drift/DMS=true/INPUT_CONFIG=job.yml.tmpl/READPLAN=$' \
  -count=3 -v
```

`DMS_CLI_TRACE` reports elapsed time for each forwarded HTTP request, through reading
the response body. It does not change routing. For a pure-testserver baseline, run
the same command without a discoverable settings file and with
`DMS_CLI_ENDPOINTS_FILE` unset. Do not point it at a nonexistent file: that is an error.

Compare the leaf Go subtest duration, not package build/setup time. This current
fixture deploys a job, verifies no drift, removes an optional field, plans/redeploys,
verifies convergence, and destroys the bundle. All those operations remain enabled.

## Measurements (2026-10-14, same Arclet, Go 1.27.2)

| Setup | Three leaf subtest durations |
| --- | --- |
| Testserver baseline | 0.42s, 0.42s, 0.47s |

The first package invocation included a cold Go build (about 89s of parent setup).
That is separate from the leaf timings above. LITE was not running and there was
no discoverable settings file during these baseline runs.

The branch starts from CLI main `002627a95` plus the workspace-file routing change
from https://github.com/databricks/cli/pull/7051. It keeps ordinary file content in
testserver; initially, workspace hierarchy, permissions, and DMS calls still use LITE.
