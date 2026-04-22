# NEXT CHANGELOG

## Release v0.299.0

### CLI

* Added `--limit` flag to all paginated list commands for client-side result capping ([#4984](https://github.com/databricks/cli/pull/4984)).
* Added interactive pagination for list commands that have a row template (jobs, clusters, apps, pipelines, etc.). When stdin, stdout, and stderr are all TTYs, `databricks <resource> list` now streams 50 rows at a time and prompts `[space] more  [enter] all  [q|esc] quit` on stderr. ENTER can be interrupted by `q`/`esc`/`Ctrl+C` between pages. Colors and alignment match the existing non-paged output; column widths stay stable across pages. Piped output and `--output json` are unchanged.
* Accept `yes` in addition to `y` for confirmation prompts, and show `[y/N]` to indicate that no is the default.

### Bundles

### Dependency updates
