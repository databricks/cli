# Bugbash: deployment history recording

Records every `bundle deploy` and `bundle destroy` with the Deployment Metadata
Service (DMS), so a deployment has a server-side history and its resource state
lives in the workspace rather than only in the local cache.

## Get a CLI with the feature

```shell
bash <(curl -fsSL https://raw.githubusercontent.com/databricks/cli/main/internal/bugbash/exec.sh) bugbash-record-deployment-history
```

That drops you into a shell with `databricks` on `$PATH`. Check you have the right
build with `databricks --version`.

## Turn the feature on

Three things are needed. Missing any one of them means nothing is recorded.

```shell
export DATABRICKS_BUNDLE_ENGINE=direct
export DATABRICKS_BUNDLE_FORCE_ALLOW_RECORD_DEPLOYMENT_HISTORY=true
```

and in `databricks.yml`:

```yaml
experimental:
  record_deployment_history: true
```

The env var only unlocks the gate; the YAML flag is what enables recording. Without
the env var the CLI refuses:

```
Error: experimental.record_deployment_history is not supported yet
```

Recording is direct-engine only. On terraform the flag is rejected, and no
`/api/2.0/bundle/*` calls are made.

The feature must be enabled from the bundle's **first** deploy. Turning it on for a
bundle that already has deployed resources is refused, because DMS would then own a
resource set it never saw and the next deploy would create everything a second time.
The error spells out the three steps to start over.

## A bundle to start from

```yaml
bundle:
  name: my-dms-test

experimental:
  record_deployment_history: true

resources:
  jobs:
    hello:
      name: my-dms-test-job
      tasks:
        - task_key: main
          notebook_task:
            notebook_path: ./noop.py
```

with `noop.py` beside it:

```
# Databricks notebook source
print(1)
```

## Find the deployment

The CLI stores the deployment ID nowhere. DMS registers the deployment as a workspace
node, and that node's object ID *is* the deployment ID:

```shell
databricks workspace get-status \
  "/Workspace/Users/$(databricks current-user me | jq -r .userName)/.bundle/my-dms-test/default/state/resources.deployment.json" \
  -o json | python3 -c 'import sys,json; print(json.load(sys.stdin)["object_id"])'
```

Use `python3`, not `jq`, for that ID. It exceeds 2^53 and jq below 1.7 silently
rounds it, which looks like "deployment does not exist".

Or read it straight off the summary:

```shell
databricks bundle summary -o json | jq .bundle.deployment.history
```

## What to look at

```shell
databricks api get "/api/2.0/bundle/deployments/$DID"                    # the deployment
databricks api get "/api/2.0/bundle/deployments/$DID/versions"           # one version per deploy
databricks api get "/api/2.0/bundle/deployments/$DID/versions/$V/operations"
databricks api get "/api/2.0/bundle/deployments/$DID/resources"          # current resource state
databricks api get "/api/2.0/bundle/deployments"                         # all deployments
```

`resources` and `operations` paginate at 20 with a `next_page_token`. A bundle with
more than 20 resources is not truncated; page through it.

Jobs and pipelines carry a back-reference to the deployment, but the SDK hides those
fields, so read them raw:

```shell
databricks api get "/api/2.0/jobs/get?job_id=$JID" | jq .settings.deployment
databricks api get "/api/2.0/pipelines/$PID" | jq .spec.deployment
```

Both should show `deployment_id` and `version_id` next to `kind: BUNDLE`.

## Worth exercising

- **Iterate.** Deploy, change a field, deploy again. Each deploy claims a version;
  only changed resources get an operation.
- **Wipe the local cache.** `rm -rf .databricks`, then `bundle plan`. It should report
  your resources as unchanged, reconstructed from DMS. It must never plan to create
  something that already exists.
- **Break a resource.** Give a job an invalid cron expression. The failed resource is
  recorded with `status: OPERATION_STATUS_FAILED` and an `error_message`, the version
  completes with `VERSION_COMPLETE_FAILURE`, and a later plan wants to create it.
- **Destroy.** A destroy records its own version with a DELETE per resource, then
  deletes the deployment record.
- **Non-job resources.** Pipelines, schemas, volumes, experiments, registered models,
  secret scopes and dashboards are all recorded. Each has a differently-shaped
  resource id (numeric, UUID, `catalog.schema.name`, a scope name).
- **Targets.** Each target has its own state path, so `-t dev` and `-t prod` are
  separate deployments with separate version chains.
- **Provenance.** Deploy from a git repo and check `git_info` on the version;
  `deployment_mode` reflects the target's `mode`.

## Not bugs

- A redeploy with no changes still creates a version, with no operations under it.
- After `destroy`, `GetDeployment` still returns the record with
  `status: DEPLOYMENT_STATUS_DELETED`. That is a soft delete.
- `state` on an operation or resource is a **quoted JSON string**, not an embedded
  object. Parse it once to get `{"state": {...}, "depends_on": [...]}`.
- DMS resource keys have no `resources.` prefix (`jobs.foo`), unlike local state keys.
- Sub-resources get their own operation, e.g. `secret_scopes.mine.permissions`.
- Permissions are not set on the deployment node. It inherits from the state folder,
  which the bundle's `permissions:` section already governs.

## Reporting

Include the deployment ID, the version, and the request/response for anything that
looks wrong. `databricks bundle deploy --log-level debug` logs the DMS calls.
