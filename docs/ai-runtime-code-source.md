# AI Runtime task code source

An `ai_runtime_task` runs your command against a directory of code. Set `code_source_path`
and `databricks bundle deploy` packages that code into a tarball, uploads it, and rewrites
the path to where it landed in the workspace. At runtime the tarball is extracted to
`/databricks/code_source/<dir>` and that path is exported as `$CODE_SOURCE_PATH`.

`code_source_path` takes one of three things.

## A local directory

The common case. Point it at the directory and deploy — the CLI packages it for you,
respecting `.gitignore`. Nothing else to declare.

```yaml
resources:
  jobs:
    train:
      tasks:
        - task_key: train
          ai_runtime_task:
            code_source_path: ./src
```

## A tgz artifact you declare

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

`include` packs the working tree filtered by `.gitignore`; `git` snapshots a committed ref
with `git archive`. Archive entries are named relative to `path`, so choose `path` so the
top-level directory matches the `/databricks/code_source/<dir>` your command expects.

## A workspace or volume path

A `/Workspace/…` or `/Volumes/…` value is used as-is. Nothing is packaged or uploaded.

> **Note:** The runtime extracts your code and exports `$CODE_SOURCE_PATH`, but it runs your
> command from the launch directory — it does not `cd` into the code for you. Start your
> command with `cd "$CODE_SOURCE_PATH"` (as the `air` examples do) so relative paths resolve.

## Converting an `air run` config

`databricks experimental air convert-to-dabs <config>.yaml` turns an existing `air` run
config into a bundle. A plain snapshot becomes a local-directory `code_source_path`; a
snapshot that pins a git ref or narrows `include_paths` becomes a `tgz` artifact. The command
is copied into `command.sh` verbatim, so a config that already `cd`s into `$CODE_SOURCE_PATH`
runs unchanged.
