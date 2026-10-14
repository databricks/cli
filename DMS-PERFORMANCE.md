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
| Real DMS + original LITE workspace fake, warm | 7.90s, 6.90s, 9.68s |
| Real DMS + reuse display-name configuration, warm | 4.00s, 4.46s, 3.66s |

The first package invocation included a cold Go build (about 89s of parent setup).
That is separate from the leaf timings above. LITE was not running and there was
no discoverable settings file during these baseline runs.

The branch starts from CLI main `002627a95` plus the workspace-file routing change
from https://github.com/databricks/cli/pull/7051. It keeps ordinary file content in
testserver; initially, workspace hierarchy, permissions, and DMS calls still use LITE.

## Experiment 1: remove repeated configuration construction in the workspace fake

The real-DMS baseline used `-count=4`: the first, cold leaf run took 13.13s;
the three warm runs are above. Each run forwarded 51 HTTP requests. Across the
four runs, the 20 `mkdirs` requests took 9.46s and 76 `get-status` requests took
8.81s (aggregate time, not one test's duration).

In the LITE REPL, constructing a Simulation workspace client averaged 0.18ms
over 20 constructions. Ten `GetStatus` calls on one reused client averaged
55.59ms each. Reusing clients alone cannot eliminate that RPC cost.

A local JFR recording of 200 `GetStatus` calls showed repeated
`FakeTreeStoreV2.getDisplayName -> WebappConf.default -> new WebappConf` work,
including configuration/flag initialization and hash-map growth. The fake was
constructing the whole webapp configuration for each returned node's display name.
The Universe experiment reuses one lazily initialized default configuration for
that conversion. DMS itself, its MySQL-backed EntityStore, routing, and the CLI
test's operations are unchanged.

JFR reproduction (use the current test-environment JVM PID):

```sh
jcmd "$DMS_TESTENV_PID" JFR.start name=dms_latency settings=profile \
  duration=30s filename=/tmp/dms-simulation.jfr
jfr print --json --stack-depth 100 --events jdk.ExecutionSample \
  /tmp/dms-simulation.jfr
```

The cold leaf after restarting for this change took 8.87s. The same test still
forwarded 51 requests; no calls were skipped or served from a DMS-response cache.

## Experiment 2: reuse the workspace fake's ACL client

The next JFR recording showed `loadHomeFolderAndTrashFolder` building a fresh
`AclTestApiClientBuilder` client for every workspace RPC. Its first per-org
lookup builds Central clients, configurations, and SSL arguments again. Reusing
only the caller's gRPC stub does not remove this server-side construction.

The second Universe change reuses one lazily constructed ACL client within each
`WorkspaceServiceV2FakePlugin.initializeComponents` lifetime. Resetting the fake
creates a new lazy client, so another fixture does not inherit old endpoints.
The permission grants and checks still execute; only client construction is reused.
