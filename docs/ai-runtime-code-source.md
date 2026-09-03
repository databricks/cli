# AI Runtime - DABs Integration

[**AI Runtime**](https://docs.databricks.com/aws/en/machine-learning/ai-runtime/) is serverless
GPU compute for training and fine-tuning models — give it a command and a GPU spec, no cluster to
set up. [**Databricks Asset Bundles**](https://docs.databricks.com/aws/en/dev-tools/bundles/)
package Databricks resources (jobs, code, pipelines) as one version-controlled, deployable unit.

Together they let you manage a GPU training workload the way you manage any production job: define
it once, deploy it across environments, schedule it, and compose it into larger workflows — instead
of ad-hoc, one-off submissions. No `air` CLI required.

## Why run AI Runtime in a bundle

- **Version-controlled and reviewable** — the workload's configuration lives in source control
  alongside the rest of your project.
- **One definition, many environments** — the same bundle promotes from a development target to
  production.
- **Composable** — the GPU task sits in a job next to notebooks, SQL, and pipelines, chained by
  dependencies and gated by schedules like any other Jobs task.
- **Migratable** — an existing `air run` config converts to a bundle with one command.

## What it looks like

An AI Runtime workload is a task (`ai_runtime_task`) in a job: you point it at your code, name the
GPU it needs, and give it a command to run.

```yaml
resources:
  jobs:
    train:
      tasks:
        - task_key: train
          ai_runtime_task:
            experiment: my-experiment
            code_source_path: ./src          # your code
            deployments:
              - command_path: ./command.sh   # what to run
                compute:
                  accelerator_type: GPU_1xA10 # or GPU_1xH100, GPU_8xH100
                  accelerator_count: 1
```

Deploy and run it like any bundle:

```bash
databricks bundle deploy
databricks bundle run train
```

Retries, timeouts, permissions, and dependencies are configured the same way as any Jobs task, so
existing bundle practices carry over.

## Getting your code to the workload

`code_source_path` says where your training code lives; the bundle packages and uploads it on
deploy. Three options, by how much control you need:

- **A local directory** (`./src`) — the common case. The bundle packages the directory for you,
  honoring `.gitignore`.
- **A declared artifact** — when you want only a subset of files, or a specific committed git
  revision rather than your current working tree.
- **An existing workspace or volume path** (`/Workspace/…`, `/Volumes/…`) — used as-is, when your
  code is already uploaded.

## Multi-step workflows

Because an AI Runtime task is a Jobs task, it composes with everything Jobs offers: chain a prep
step into a GPU training step with `depends_on`, mix GPU and CPU tasks in one job, and use
different GPU types per task.

> One capability to design around: an AI Runtime task can't pass values through the Jobs
> task-values mechanism. To hand data between steps, write it to a shared UC Volume or workspace
> file that both steps read.

## Scheduling and production

Scheduling and dev→prod promotion are standard bundle features, and the AI Runtime task inherits
them unchanged — add a schedule to run the job on a cadence, and a production target to promote it,
exactly as you would for any bundle job.

## Custom container images

To run inside your own Docker image instead of the default AI Runtime image, register the image
with AI Compute once, then reference it — useful when you need specific system libraries or a
locked-down environment. See the [AI Runtime docs](https://docs.databricks.com/aws/en/machine-learning/ai-runtime/)
for details.

## Migrating from `air run`

`databricks experimental air convert-to-dabs <config>.yaml` turns an existing `air run` config into
a bundle — mapping the code source, dependencies, and command across — so teams already using the
`air` CLI can adopt bundles without rewriting their workload by hand.

Unlike an `air run` submission (which the platform cleans up automatically), a deployed bundle
creates a persistent job; remove it with `databricks bundle destroy` when you're done.
