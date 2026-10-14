# Local DMS performance experiments

Experimental branches: [CLI](https://github.com/databricks/cli/tree/denis/dms-fast-acceptance)
and [Universe](https://github.com/databricks-eng/universe-dev/tree/denis-bilenko_data/dms-fast-acceptance).

The subject is the production DMS implementation. Jobs and other workspace dependencies
can be stateful test fakes. Do not replace DMS with the Go testserver implementation,
skip CLI operations, or change the invariant's assertions to improve timing.

## Result

The final warm job no-drift test is **2.88× testserver** on this Arclet:
**1.095s versus 0.38s median**. The combined real-DMS mean is 1.126s versus
0.382s (2.95×). This measures the leaf subtest, not the package/build wall time.

The first run after restarting the JVM took **6.23s**. Twenty warm-up repetitions
preceded the final comparison. JVM initialization and compilation remain expensive;
the result applies to iterative testing against an already-running service.
All 23 DMS requests per repetition reach the production DMS server, with its real
handlers, MySQL-backed EntityStore, and permission checks. No DMS responses or
permission decisions are cached. The test script and assertions are unchanged.

## Reproduction

From the Universe root on the experimental `dms-fast-acceptance` branch:

```sh
DMS_CLI_FAST=1 DMS_CLI_SIMSMITH=1 USE_DBEXEC=0 bin/lite up \
  //bundle/metadata-service/test:cli-testenv-bin --name cli-testenv-bin
```

From the CLI root, with a running local DMS fixture whose connection settings are in
the enclosing Universe checkout's `bundle/metadata-service/test/.lite/cli-endpoints.json`:

```sh
PATH=/home/denis.bilenko/.local/bin:$PATH DMS_CLI_TRACE=1 \
  go test ./acceptance \
  -run 'TestAccept/bundle/invariant/no_drift/DMS=true/INPUT_CONFIG=job.yml.tmpl/READPLAN=$' \
  -count=10 -v
```

`DMS_CLI_TRACE` reports elapsed time for each forwarded HTTP request, through reading
the response body. It does not change routing. For a pure-testserver baseline, run
the same command without a discoverable settings file and with
`DMS_CLI_ENDPOINTS_FILE` unset. Do not point it at a nonexistent file: that is an error.

Compare the leaf Go subtest duration, not package build/setup time. This current
fixture deploys a job, verifies no drift, removes an optional field, plans/redeploys,
verifies convergence, and destroys the bundle. All those operations remain enabled.

## Measurements (2026-10-14, same Arclet, Go 1.27.2)

| Setup | Leaf subtest durations |
| --- | --- |
| Testserver baseline | 0.42s, 0.42s, 0.47s |
| Real DMS + original LITE workspace fake, warm | 7.90s, 6.90s, 9.68s |
| Real DMS + reuse display-name configuration, warm | 4.00s, 4.46s, 3.66s |
| Also reuse the fake's ACL client, warm | 2.35s, 2.15s, 2.05s, 2.13s |
| Also local missing-file lookups, one-time home grants, reused gateway clients, quieter Simulation logging | 2.28s, 2.00s, 1.93s, 1.72s, 1.76s |
| SimSmith dependency dispatcher, first five warm runs | 2.06s, 1.89s, 1.80s, 1.68s, 1.70s |
| SimSmith after further JVM warmup, with profiling | 1.66s, 1.57s, 1.52s, 1.57s, 1.47s, 1.48s, 1.47s, 1.48s |
| Also asynchronous MySQL flushing | 1.38s, 1.32s, 1.32s, 1.31s, 1.29s, 1.37s |
| Also direct testserver-to-DMS transport | 1.34s, 1.29s, 1.30s, 1.28s, 1.34s, 1.21s |
| Also existing signed-session-token cache | 1.23s, 1.16s, 1.18s, 1.15s, 1.20s, 1.17s |
| Fresh testserver-only comparison | 0.36s, 0.37s, 0.37s, 0.55s, 0.44s, 0.40s |

The first package invocation included a cold Go build (about 89s of parent setup).
That is separate from the leaf timings above. LITE was not running and there was
no discoverable settings file during these baseline runs.

The branch starts from [CLI main](https://github.com/databricks/cli/commit/002627a95)
plus the workspace-file routing change from [PR 7051](https://github.com/databricks/cli/pull/7051).
It keeps ordinary file content in
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

## Experiment 3: reduce redundant dependency work

Ordinary files, including missing-file status/deletion, are owned by testserver.
Mirrored directories still get their remote object IDs; `resources.deployment.json`
is always looked up remotely. This reduces forwarding from 51 to 40 HTTP requests,
while all 23 DMS requests still reach the real server.

The gateway reuses thread-safe workspace/tree stubs separately for each fixture user.
The workspace fake initializes home-folder permissions once per org/user/home-node,
clearing that bookkeeping on reset. It no longer overwrites later permission changes
on every workspace read. All operation-specific permission checks remain in place.
Raw-name conversion also reuses its default WebappConf.

The interactive fixture disables Simulation's full service-request and span logs;
`DMS_CLI_REQUEST_LOGS=1` restores them. Normal API suites retain detailed logging.
This does not disable error logging or DMS's own logs. Cold leaf after restart: 6.65s.
Warm timings above are from `-count=6`. This stage still misses the requested target.

## Experiment 4: dispatcher, database, and transport

`DMS_CLI_SIMSMITH=1` selects SimSmith for dependencies, retaining the existing JVM
workspace and ACL adapters and native Central fake. It still starts the real DMS
container with MySQL-backed EntityStore. A standalone test Dicer assigner supplies
assignments because SimSmith has no Dicer adapter. Changing dispatchers alone did
not provide a clear improvement. The first SimSmith attempt failed due to mixed
native/JVM ACL state; the final dependency set uses only the JVM ACL adapter.

The next profile found DMS waiting on workspace RPCs and database transactions.
For the disposable local database, setting `innodb_flush_log_at_trx_commit=2` and
`sync_binlog=0` reduced commit latency. This preserves SQL transactions but does
**not** test crash durability. The database is discarded on fixture teardown.

Optional `dms_url` and `dms_headers` settings let testserver call real DMS directly,
using the same mTLS certificates and synthetic fixture principal as the gateway.
Workspace requests still go to the gateway. Both destinations must be HTTPS
loopback origins. The direct headers are sent only on DMS requests, and the old
gateway-only settings remain supported. Direct transport by itself had little
effect on total time.

The existing `databricks.auth.enableJwsCache` flag reuses signed session tokens for
outbound workspace calls. It does not cache DMS responses or permission decisions.

These are incremental measurements on an increasingly warm JVM, not randomized
A/B samples. The startup-configured result must be measured again after restart.

## Experiment 5: fewer workspace RPCs and co-located permission clients

Workspace `get-status` now runs the existing permission validator inside the shared
workspace fake. The gateway no longer repeats the tree lookup and ACL RPC. Other
gateway path lookups use `getTreeNodeByPath` instead of `get-status` followed by a
second tree lookup. All checks still execute; the DMS API suite verifies grants,
revocations, user isolation, and that a read-only user cannot create a version.

Restarting the normal Simulation backend with all fast-mode settings applied at
startup gave a 6.92s cold leaf, then 2.24, 1.83, 1.73, 1.64, 1.52, 1.51, 1.44,
1.38, 1.38s. Another ten-run batch ended at 1.21s. A new JFR showed that RSA signing
was no longer prominent, but DMS still waited on workspace RPCs.

The first co-location experiment gave the workspace fake and gateway the ACL fake's
own in-memory client. This shares the live permission store, not a cache. It helped
little: after restart the cold leaf was 6.62s; the second ten-run batch ended at
1.22s. Inspection showed that the workspace permission validator actually uses its
RBAC client for these reads, so they still made an RPC to the co-located ACL fake.

The next experiment uses the existing in-process RBAC test client with the ACL fake's
same legacy RBAC handler. This runs backend validation hooks and shares ACL state
with RPC clients. The full DMS API suite passes all 14 cases, including read-only
write denial, grants and revocations. The normal, non-fast fixture keeps RPC clients.
Transport/service-identity conformance between dependency fakes is not covered by
this co-location mode; real DMS HTTP/mTLS transport and user authorization remain
covered.

The first twenty-run SimSmith batch with in-process ACL/RBAC started at 6.57s cold
and ended at 1.25s. Its final five leaves were 1.29, 1.29, 1.19, 1.22, 1.25s; this
still misses the requested ratio against the earlier 0.385s testserver median.

## Experiment 6: invoke co-located workspace handlers directly

The gateway and workspace fake already live in the same JVM, but every gateway
operation still serialized a request through Simulation before reaching that fake.
Fast mode now invokes the fake's registered blocking handlers directly, with a new
request context for the selected fixture user. The same workspace hooks, tree
storage, and read-ACL validator run. DMS's own outbound workspace calls are unchanged.

All 14 DMS API cases pass with this path and SimSmith, including grants, revocations,
cross-user isolation, read-only write denial, multipart imports, and file lifecycle.
This does not claim transport coverage for the removed gateway-to-fake loopback hop.

## Final comparison

Code revisions: [CLI](https://github.com/databricks/cli/commit/b43a267c0)
and [Universe](https://github.com/databricks-eng/universe-dev/commit/bb0e31172dfdd).
Later CLI commits only update this report. The Universe branch also includes the
earlier [LITE log-tail busy-loop fix](https://github.com/databricks-eng/universe/pull/2802564).

Run order: restart, 20 real-DMS warm-up repetitions, real-DMS A, testserver,
real-DMS B. No profiling or concurrent builds/tests during these three measured
batches. The DMS fixture stayed up but idle during the testserver batch; the
settings file was temporarily moved out of discovery. All repetitions passed.

| Batch | All leaf durations (seconds) | Median |
| --- | --- | --- |
| Real DMS A | 1.10, 1.09, 1.06, 1.06, 1.07, 1.16, 1.49, 1.21, 1.16, 1.05 | 1.095s |
| Testserver | 0.38, 0.39, 0.38, 0.38, 0.38, 0.39, 0.38, 0.38, 0.38, 0.38 | 0.38s |
| Real DMS B | 1.05, 1.20, 1.20, 1.14, 1.05, 1.02, 1.02, 1.03, 1.23, 1.12 | 1.085s |

There is variance: the slowest real-DMS sample was 1.49s. The 3× result is the
median and mean, not a guarantee that every individual run is below 3×.
Each ten-run real-DMS log contains 400 forwarded requests, 230 of them to DMS.

From the CLI root, warm up with the reproduction command using `-count=20`.
Then run the `-count=10` command twice, bracketing the baseline command below.
The measured logs were `/tmp/dms-speed-final-real-a.log`,
`/tmp/dms-speed-final-testserver-a.log`, and `/tmp/dms-speed-final-real-b.log`.
Their timing samples are retained above because `/tmp` is not persistent.

Exact baseline command on this Arclet (restores the settings file even on failure):

```sh
env -u DMS_CLI_ENDPOINTS_FILE PATH=/home/denis.bilenko/.local/bin:$PATH zsh -c '
settings=/home/denis.bilenko/universe/bundle/metadata-service/test/.lite/cli-endpoints.json
backup=$(mktemp -d /tmp/dms-baseline.XXXXXX)
mv "$settings" "$backup/cli-endpoints.json"
trap '\''mv "$backup/cli-endpoints.json" "$settings"'\'' EXIT
go test ./acceptance \
  -run '\''TestAccept/bundle/invariant/no_drift/DMS=true/INPUT_CONFIG=job.yml.tmpl/READPLAN=$'\'' \
  -count=10 -v > /tmp/dms-speed-final-testserver-a.log 2>&1
'
```

## Regression checks

The following passed after the last code change. These runs are correctness checks,
not timing comparisons; some ran alongside Universe builds.

From the CLI root (with the local fixture discoverable):

```sh
go test ./acceptance/internal -run TestLitebox -count=1
go test ./acceptance -run 'TestAccept/bundle/dms/no-drift$' -count=1 -v
go test ./acceptance \
  -run 'TestAccept/bundle/invariant/no_drift/DMS=(true)?$/INPUT_CONFIG=job.yml.tmpl' \
  -count=1 -v
```

The last command covers DMS on/off and READPLAN on/off (four passing variants).
The CLI routing tests cover file ownership/mirroring and direct mTLS forwarding,
including bearer-token stripping and limiting fixture identity headers to DMS.
`./task lint-q` passed with zero Go issues and passing Ruff checks.

From the Universe root:

```sh
bazel test //bundle/metadata-service/test:DmsCliApiSuite \
  --test_env=DMS_CLI_FAST=1 --test_env=DMS_CLI_SIMSMITH=1 \
  --tool_tag=ai-agent --test_output=errors \
  --noshow_progress --noshow_loading_progress
bazel test \
  //webapp/api/servicefakes/workspace-servicefake:FakeTreeStoreV2Suite \
  //webapp/api/servicefakes/workspace-servicefake:WorkspaceServiceFakePluginV2Suite \
  //bundle/metadata-service/test:bazel_lint_test \
  //common/logging:bazel_lint_test \
  --tool_tag=ai-agent --test_output=errors \
  --noshow_progress --noshow_loading_progress
```

The XML reports contain 14 passing DMS API cases and 33 passing workspace cases,
with zero errors/failures. The DMS API suite was also rerun without either
`--test_env` flag: all 14 cases pass in normal (non-fast) mode as well.
Both BUILD lint tests passed.

`runtests --quiet --test_type lint --force_gazelle` also passed all seven executed
lint targets (five Scala-format targets and two BUILD lint targets); Gazelle found
no BUILD changes. Its metadata lists 38 selected, seven passing, and 31 not run
in this lint-only invocation, with zero failed tests.
