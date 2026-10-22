---
name: probe-resource-api
description: Establish Databricks resource API behavior on live test workspaces, including lifecycle edge cases, and turn observations into faithful testserver behavior and acceptance tests. Use during bundle onboarding or when SDK/docs, the adapter and the backend disagree.
---

# Probe a resource API

Use live evidence to resolve a specific API contract question. A passing test against `libs/testserver` verifies the fake; it cannot establish backend behavior. For bundle implementation, pair this workflow with [onboard-bundle-resource](../onboard-bundle-resource/SKILL.md). Follow [the testing rules](../../rules/testing.md) when preserving experiments as tests.

Read [recording and cloud tests](references/recording.md) when running acceptance probes or retaining evidence. It explains the available request recorder, response diagnostics and their limitations. There is no general recording-to-testserver generator.

## Define the experiment

Start with the pinned SDK models, serialization and methods, `.codegen/cli.json`, upstream API annotations when available, and current adapter/tests. Separate documented behavior, live observations and unresolved assumptions. Reuse existing evidence for unchanged behavior; probe the gaps relevant to the task.

Record the CLI/SDK revision, date, cloud and relevant service configuration with the conclusions. Backend rollout and permissions can change observations across workspaces. Resolve the resource's identity and parent dependencies before experimenting; do not infer supported updates from a similarly named data-plane method.

Use an isolated CLI test environment (`deco env run -i -n aws-cli`, `azure-cli` or `gcp-cli`) or the disposable workspace selected for the task. If available, follow the installed `databricks-cli-dev:run-cloud-tests` skill. Confirm the workspace and principal without printing credentials. Use uniquely named, test-owned resources and minimal dependencies. Missing access or unavailable service rollout is a coverage limitation to report.

Register cleanup before the first mutation, record returned IDs immediately, and delete children before parents. Use bounded waits for the resource's actual completion conditions. In acceptance tests, `script.cleanup` is appended to the script and is skipped if an earlier command aborts; use an `EXIT` trap for failure-path cleanup. Preserve the original failure, collect cleanup errors in `LOG.cleanup`, and report any remaining IDs. Isolated environments do not replace cleanup.

## Choose the cases

Select rows that can affect the resource's deployment contract. Start with minimal create/read/delete and add populated requests and the relevant edge cases. Do not turn this into an exhaustive cross-product of unrelated fields.

| Question | Useful experiments |
| --- | --- |
| Identity and parents | Full resource name vs leaf ID vs opaque ID; required parents; duplicate create; identity after same-name recreation; bind input and URL encoding. |
| Defaults and readback | Minimal vs populated create; backend defaults and normalization; requested vs effective values; output-only/write-only fields; fields returned only with include flags. |
| Updates and clearing | Set a non-default, then omit it or send null, empty string/list/map/object, false or zero as applicable. Test accepted mask forms (`*`, parent, explicit leaf), preservation of unrelated fields, nested merge/replacement and SDK force-send behavior. |
| Immutable or conditional fields | Change allegedly immutable fields or discriminator/oneof variants; omit required identity; send output-only fields; test invalid or empty masks. Check the final state as well as the error. |
| Async completion and failure | Create return vs readiness; LROs; transient read results; deletion vs name availability; repeated delete; error/timeout after a possibly successful mutation. |
| Permissions and grants | Supported public securable type, owner/ACL identity and inheritance, missing vs access-denied errors on test-owned objects. An arbitrary 403 does not mean deleted. |
| Singletons and drift | Update-as-create and reset-as-delete; parent lifetime; external edit/deletion; removal from local config; convergence after deploy or bind. |

For an AIP API, verify the service's actual implementation of names, masks, field behaviors and asynchronous methods. An annotation or familiar request shape alone does not establish how defaults, omission or nested masks behave.

## Execute and read back

Use generated commands or the SDK when investigating their normal serialization. Use `./cli api post|put|patch PATH --json @body.json` for deliberate wire-payload experiments: [the raw API command](../../../cmd/api/api.go) preserves those JSON payload bytes, including explicit nulls and zeros. For GET/DELETE/HEAD, `--json` supplies query parameters. The command uses normal auth/workspace routing and supports `--workspace-id` when needed. Keep static request bodies in input fixtures.

The ordinary API command does not expose response status or headers, and the acceptance proxy normalizes successful status codes and hides some upstream headers/retries. For questions about exact status, headers or transport attempts, use direct authenticated transport diagnostics that explicitly capture those properties; see the recording reference. Keep raw logs private and commit only sanitized, relevant evidence.

Run one hypothesis at a time. Read the resource after **every mutation, including failed or timed-out writes**. A 500 does not establish atomic rejection: Postgres mask probes have observed partial updates before an error. Use bounded polling to distinguish eventual consistency from stable behavior. Before retrying an ambiguous create, reconcile by its returned ID or test-owned identity to avoid duplicate resources.

Capture enough evidence to reproduce the conclusion:

- Exact method, path, query and request body, plus the initial relevant state.
- CLI outcome and API error; status/headers only when actually observed through a suitable capture.
- Relevant readback, operation state and timing, including readback after failures.
- The adapter, field-policy or fake behavior implied by the observation; any unresolved question.

Do not generalize one workspace's transient defect into a broad client fallback. Preserve the narrow observed condition, check current SDK/service behavior, and identify an upstream fix when appropriate.

## Preserve the useful cases

For new behavior, direct experiments can establish the contract before a fake route exists. Preserve stable cases as focused `Cloud = true` tests under `acceptance/bundle/resources/<plural>/`, using a `factcheck/` subtree for API-only cases when helpful. The Postgres examples in the recording reference show the pattern. Raw API tests can remove the inherited DMS matrix because they do not exercise deployment metadata; bundle lifecycle tests still need their applicable variants.

Model the observed state changes, required inputs, errors, defaults and normalization in shared `libs/testserver` handlers. Update handlers must mutate stored state, and clearing tests must distinguish an absent JSON key from a present empty value. A 404 may be a correct response for an absent object; verify routing before diagnosing a missing handler. Canned responses and goldens copied from the fake cannot establish fidelity.

Use `trace`, `musterr`, `contains.py` and `print_requests.py` from the acceptance harness. Project GET responses to the fields being asserted so unrelated backend additions do not churn output. Preserve operation order and duplicate attempts when they are the subject of the test. Recording is an observation aid, not an automatically generated behavioral model.

Run the same contract assertions locally and in the selected cloud; inspect effective settings and verbose test names to ensure the cloud case ran. Regenerate intentional output with `-update`, review it, then rerun without update. Add bundle lifecycle coverage proving the adapter converges after create, update and field removal; API-only probes do not establish bundle behavior.

Finish with evidence-backed conclusions, tests run, unavailable coverage, implementation implications and cleanup outcome. When access is unavailable, distinguish the unverified contract from fake-only assertions rather than treating local success as live validation.
