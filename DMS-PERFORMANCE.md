# Local DMS performance experiments

Experimental branch: `denis/dms-fast-acceptance`.

The subject is the production DMS implementation. Jobs and other workspace dependencies
can be stateful test fakes. Do not replace DMS with the Go testserver implementation,
skip CLI operations, or change the invariant's assertions to improve timing.

## Reproduction

From the Universe root on the experimental `dms-fast-acceptance` branch:

```sh
DMS_CLI_FAST=1 USE_DBEXEC=0 bin/lite up \
  //bundle/metadata-service/test:cli-testenv-bin --name cli-testenv-bin
```

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
