# NEXT CHANGELOG

## Release v1.8.0

### Notable Changes

### CLI

### Bundles

* An `ai_runtime_task.code_source_path` that points at a local directory is now packaged into a tarball during `bundle deploy`, uploaded to the user's workspace code snapshot directory, and rewritten to the uploaded path, mirroring the AIR CLI's `code_source` behavior. The snapshot honors `.gitignore` and the top-level `sync.include`/`sync.exclude` globs, and is content-addressed so re-deploying unchanged code skips the re-upload. The task's `deployments[].command_path` is translated to its workspace path, and a `requirements.yaml` derived from the job's serverless `environments` is written alongside it so the workload's environment is set up correctly.

### Dependency updates

### API Changes
