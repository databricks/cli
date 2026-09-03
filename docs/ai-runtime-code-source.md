# AI Runtime - DABs Integration

An AI Runtime task (`ai_runtime_task`) runs a command on serverless GPU compute. Put one in a
Databricks Asset Bundle and it deploys, versions, schedules, and composes with the rest of Jobs
like any other task — no `air` CLI required. This page covers the task itself, how your code
reaches it, multi-task workflows, scheduling, and converting an existing `air run` config.

## The task

```yaml
resources:
  jobs:
    train:
      tasks:
        - task_key: train
          environment_key: default
          max_retries: 3
          ai_runtime_task:
            experiment: my-experiment          # required, no spaces
            code_source_path: ./src            # your code (see below)
            deployments:
              - command_path: ./command.sh     # the script to run
                compute:
                  accelerator_type: GPU_1xA10  # GPU_1xA10 | GPU_1xH100 | GPU_8xH100 (case-sensitive)
                  accelerator_count: 1         # total GPUs; a multiple of the per-node count
      environments:
        - environment_key: default
          spec:
            environment_version: "5"
            dependencies: [numpy]
```

- `command_path` is the launch script. A `./`-relative path is uploaded on deploy; an absolute
  `/Workspace/…` path is used as-is.
- Scale up by raising `accelerator_count` (a multiple of the per-node count: 1 for
  `GPU_1xA10`/`GPU_1xH100`, 8 for `GPU_8xH100`) or switching `accelerator_type`.
- Retries, timeouts, and permissions are not inside `ai_runtime_task` — they live on the standard
  task wrapper (`max_retries`, `timeout_seconds`, …), exactly like any other Jobs task.
- The `environments` block is required: Jobs installs the task's dependencies from its `spec`.

Deploy and run it like any bundle:

```bash
databricks bundle deploy
databricks bundle run train
```

## Shipping your code

`code_source_path` tells the bundle where your code is. On deploy the CLI packages it into a
tarball, uploads it, and rewrites the path to its workspace location. At runtime the tarball is
extracted to `/databricks/code_source/<dir>`, and that path is exported as `$CODE_SOURCE_PATH`.

`code_source_path` takes one of three things, depending on how much control you need.

### A local directory

The common case. Point it at the directory and deploy — the CLI packages it for you, respecting
`.gitignore`. Nothing else to declare.

```yaml
code_source_path: ./src
```

### A tgz artifact you declare

Use this when you want a subset of the directory or a specific git ref instead of the whole
working tree. Declare a `tgz` artifact and point `code_source_path` at its output file:

```yaml
artifacts:
  code:
    type: tgz
    path: .
    include: [src]          # or, instead of include:  git: {branch: main}
    files:
      - source: ./dist/code.tgz
resources:
  jobs:
    train:
      tasks:
        - task_key: train
          ai_runtime_task:
            code_source_path: ./dist/code.tgz
```

`include` packs the working tree filtered by `.gitignore`; `git` snapshots a committed ref with
`git archive`. Archive entries are named relative to `path`, so choose `path` so the top-level
directory matches the `/databricks/code_source/<dir>` your command expects.

### A workspace or volume path

A `/Workspace/…` or `/Volumes/…` value is used as-is. Nothing is packaged or uploaded.

> **Note:** The runtime extracts your code and exports `$CODE_SOURCE_PATH`, but it runs your
> command from the launch directory — it does not `cd` into the code for you. Start your command
> with `cd "$CODE_SOURCE_PATH"` so relative paths resolve.

## Multi-task workflows

An `ai_runtime_task` composes with the rest of Jobs. Chain tasks with `depends_on`, mix GPU and
non-GPU (e.g. a serverless-CPU prep notebook) tasks in one job, and run different accelerator
types per task:

```yaml
tasks:
  - task_key: prep          # e.g. a notebook on serverless CPU
    notebook_task: {notebook_path: ./prep.py}
  - task_key: train
    depends_on: [{task_key: prep}]
    ai_runtime_task: {…}
```

> **Note:** An `ai_runtime_task` can't use Jobs task values (`{{tasks.<key>.values.<name>}}` /
> `dbutils.jobs.taskValues`) — it runs a bash script and has no parameter/output field. To hand
> data between a prep step and the GPU task, write it to a shared location both can read (a UC
> Volume or workspace file) and reference that path from each.

## Scheduling and production

Scheduling and dev→prod promotion are plain Jobs/bundle features — the AI Runtime task composes
with them unchanged. Add a `schedule:` to the job and a production target:

```yaml
resources:
  jobs:
    train:
      schedule:
        quartz_cron_expression: "0 0 9 * * ?"
        timezone_id: UTC
        pause_status: PAUSED        # deploy without firing runs; unpause when ready
targets:
  prod:
    mode: production
```

## Custom containers (DCS)

To run inside your own image instead of the databricks-ai image, register it once with AI Compute,
then the task runs on it. The bundle still declares an `environments` block (Jobs requires it), but
the container owns its Python environment, so no `databricks-ai` venv or dependency install happens.

```bash
databricks experimental air register image docker.io/nvidia/cuda:13.1.0-devel-ubuntu24.04
```

## Converting an `air run` config

`databricks experimental air convert-to-dabs <config>.yaml` turns an existing `air` run config
into a bundle. A plain snapshot becomes a local-directory `code_source_path`; a snapshot that pins
a git ref or narrows `include_paths` becomes a `tgz` artifact. Dependencies fold into the
`environments` spec, and the command is copied into `command.sh` verbatim — so a config that
already `cd`s into `$CODE_SOURCE_PATH` runs unchanged.

Unlike `air run` (an ephemeral submit that the platform reaps), a deployed bundle creates a
persistent job. When you're done, remove it with `databricks bundle destroy`.
