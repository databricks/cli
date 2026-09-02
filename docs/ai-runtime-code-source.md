# AI Runtime code source packaging

How an AI Runtime task's code reaches the workspace when you deploy a bundle.

## Background

An `ai_runtime_task` runs a command against a body of code named by its
`code_source_path`. At deploy the code is packaged into a gzipped tarball, uploaded,
and `code_source_path` is rewritten to the uploaded workspace path. The runtime
extracts the tarball to `/databricks/code_source/<dir>`, where `<dir>` is the tarball's
single top-level directory, and exports that path as `$CODE_SOURCE_PATH`. It runs the
task's command *without* changing into that directory, so the command is expected to
`cd "$CODE_SOURCE_PATH"` (or reference it) itself.

There are three ways `code_source_path` is handled, chosen by what it points at.

## 1. A local directory (auto-packaged)

```yaml
resources:
  jobs:
    train:
      tasks:
        - task_key: train
          ai_runtime_task:
            code_source_path: ./src
```

When `code_source_path` is a local directory, `bundle/config/mutator/aicode` synthesizes
a `tgz` artifact for it before the artifacts are prepared: `path` is the directory's
parent and `include` is its basename, so the archive nests entries under
`<basename>/…`. `code_source_path` is then rewritten to the built tarball, which is
uploaded through the normal artifact path. No `artifacts` block is needed.

`.gitignore` filters the packaged files; the bundle-wide `sync.include`/`sync.exclude`
do not apply (they scope bundle file sync, not a code artifact).

## 2. An explicit `tgz` artifact

Declare the artifact yourself when you need more control — a subset of files, a git
ref, or a custom build. `code_source_path` points at the artifact's output file (the
shared path links the two, so both resolve to the same uploaded location).

```yaml
artifacts:
  code:
    type: tgz
    path: .
    include: [src]          # subpaths of `path`; entries are named relative to `path`
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

- **`include`** packs the working tree, filtered by `.gitignore`.
- **`git: {branch|commit}`** instead snapshots a committed ref via `git archive` —
  the archive reflects the ref, not the working tree.

Entries are named relative to `path`, so `path: src` + `include: [foo]` and
`path: .` + `include: [src/foo]` produce different layouts — pick `path` so the
top-level matches the `/databricks/code_source/<dir>` your command expects.

## 3. A remote path

A `/Workspace/…` or `/Volumes/…` value is used as-is — nothing is packaged or uploaded.

## Converting from `air run`

`databricks experimental air convert-to-dabs` translates an AIR run config into a
bundle. A plain `code_source.snapshot` becomes a directory `code_source_path` (case 1);
a snapshot that pins `git` or narrows `include_paths` becomes an explicit `tgz` artifact
(case 2). The command is copied into `command.sh` verbatim, exactly as under `air run` —
so a command that already `cd "$CODE_SOURCE_PATH"`s (as air commands do) resolves against
the code without any change.
