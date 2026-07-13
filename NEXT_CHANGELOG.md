# NEXT CHANGELOG

## Release v1.8.0

### Notable Changes

### CLI

* Added `databricks experimental air init`, which scaffolds a bundle from an AI Runtime (AIR) CLI `train.yaml`, generating a job with an `ai_runtime_task` so an ephemeral AIR run can be migrated to a durable, DABs-managed job.

### Bundles

* An `ai_runtime_task.code_source_path` that points at a local directory is now packaged into a tarball during `bundle deploy`, uploaded to the workspace (or the configured UC volume), and rewritten to the uploaded path, mirroring the AIR CLI's `code_source` behavior.

### Dependency updates

### API Changes
