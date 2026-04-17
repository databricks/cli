# NEXT CHANGELOG

## Release v0.298.0

### Notable Changes

### CLI

* Added `--limit` flag to all paginated list commands for client-side result capping ([#4984](https://github.com/databricks/cli/pull/4984)).
* Added interactive pagination for list commands that render as JSON. When stdin, stdout, and stderr are all TTYs and the command has no row template, `databricks <resource> list` now streams 50 JSON items at a time and prompts `[space] more  [enter] all  [q|esc] quit` on stderr. ENTER can be interrupted by `q`/`esc`/`Ctrl+C` between pages. Piped output still receives the full JSON array; early quit still produces a syntactically valid array.

### Bundles

### Dependency updates

* Bump `github.com/databricks/databricks-sdk-go` from v0.126.0 to v0.127.0 ([#4984](https://github.com/databricks/cli/pull/4984)).
* Bump Go toolchain to 1.25.9 ([#5004](https://github.com/databricks/cli/pull/5004))

### API Changes
