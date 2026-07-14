# NEXT CHANGELOG

## Release v1.8.0

### Notable Changes

### CLI

### Bundles

* An `ai_runtime_task.code_source_path` that points at a local directory is now packaged into a gitignore-aware tarball during `bundle deploy`, uploaded to the user's workspace code snapshot directory, and rewritten to the uploaded path, mirroring the AIR CLI's `code_source` behavior. The archive is content-addressed, so re-deploying unchanged code skips the re-upload.

### Dependency updates

### API Changes
