# NEXT CHANGELOG

## Release v1.8.0

### Notable Changes

### CLI

### Bundles

* An `ai_runtime_task.code_source_path` that points at a local directory is now packaged into a tarball during `bundle deploy` (git archive of a clean HEAD, otherwise a plain tar of the working tree), uploaded to the user's workspace code snapshot cache, and rewritten to the uploaded path, mirroring the AIR CLI's `code_source` behavior.

### Dependency updates

### API Changes
