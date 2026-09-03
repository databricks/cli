# AI Runtime - DABs Integration

[**AI Runtime**](https://docs.databricks.com/aws/en/machine-learning/ai-runtime/) is a way to access Databricks serverless GPU compute for training and fine-tuning models — give it a command and a GPU spec and launch your workload, no compute setup necessary! [**Databricks Asset Bundles**](https://docs.databricks.com/aws/en/dev-tools/bundles/) package Databricks resources (jobs, code, pipelines) as one version-controlled, deployable unit. DABs is most commonly used as an orchestrator of multiple tasks.

Integrated together AI Runtime x DABs lets you manage a GPU training workload the way you manage any other production job: define it once, deploy it across environments, schedule it, and compose it into larger workflows.

## Why run AI Runtime in a bundle

- **Version-controlled and reviewable** — the workload's configuration lives in source control alongside the rest of your project.
- **One definition, many environments** — the same bundle promotes from a development target to production.
- **Composable** — the GPU task sits in a job next to notebooks, SQL, and pipelines, chained by dependencies and gated by schedules like any other Jobs task.

## What it looks like

An AI Runtime workload is a BYOT (Bring Your Own Training) task (`ai_runtime_task`) in a job: you point it at your code, name the GPU it needs, and give it a command to run.

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

Retries, timeouts, permissions, and dependencies are configured the same way as any Jobs task, so existing bundle practices carry over.

## Getting your code to the workload

`code_source_path` says where your training code lives; the bundle packages and uploads it on deploy. Three options, by how much control you need:

- **A local directory** (`./src`) — the common case. The bundle packages the directory for you, honoring `.gitignore`.
- **A declared artifact** — when you want only a subset of files, or a specific committed git revision rather than your current working tree.
- **An existing workspace or volume path** (`/Workspace/…`, `/Volumes/…`) — used as-is, when your code is already uploaded.

## Multi-step workflows

Because an AI Runtime task is a Jobs task, it composes with everything Jobs offers: chain a prep step into a GPU training step with `depends_on`, mix GPU and CPU tasks in one job, and use different GPU types per task. For example, a prep notebook on serverless CPU that stages data, then a GPU training step that runs after it — on a daily schedule:

```yaml
resources:
  jobs:
    train_pipeline:
      tasks:
        - task_key: prep                    # e.g. a notebook on serverless CPU
          notebook_task:
            notebook_path: ./prep.py
        - task_key: train                   # GPU task, runs after prep
          depends_on:
            - task_key: prep
          ai_runtime_task:
            experiment: my-experiment
            code_source_path: ./src
            deployments:
              - command_path: ./command.sh
                compute:
                  accelerator_type: GPU_1xA10
                  accelerator_count: 1
      schedule:
        quartz_cron_expression: "0 0 9 * * ?"  # daily at 09:00
        timezone_id: UTC
        pause_status: PAUSED                   # deploy without firing; unpause when ready
```

> One capability to design around: an AI Runtime task can't pass values through the Jobs task-values mechanism. To hand data between steps, write it to a shared UC Volume or workspace file that both steps read.

## Scheduling and production

Scheduling and Dev → Prod promotion are standard bundle features, and the AI Runtime task inherits them unchanged. The example above adds a daily `schedule:` (shipped `PAUSED`, so deploying it never fires runs on its own — unpause it when you're ready); a production target promotes it, exactly as you would for any bundle job.

## Custom container images

To run inside your own Docker image instead of the default AI Runtime image, register the image with AI Compute once, then reference it — useful when you need specific system libraries or a locked-down environment. See the [AI Runtime Custom Image docs](https://docs.databricks.com/aws/en/machine-learning/ai-runtime/cli/docker-images) for details.
