---
description: Fast and full verification commands for the aitools command group
globs:
  - "cmd/aitools/**"
  - "libs/aitools/**"
  - "experimental/aitools/**"
paths:
  - "cmd/aitools/**"
  - "libs/aitools/**"
  - "experimental/aitools/**"
---

# aitools verification

**RULE: After every edit under `cmd/aitools`, `libs/aitools` or `experimental/aitools`, run `./task aitools-quick`.** It runs golangci-lint and the unit tests of the aitools packages and compiles their importers, in parallel, and prints a summary for every step and detailed output for failed steps. `-- --json <path>` also writes a machine-readable report, with file:line findings when available.

**RULE: Before opening a PR that touches aitools, run `./task aitools-full`.** It adds the importers' unit tests, the acceptance tests that exercise aitools and the deadcode check, which catch output and wiring changes the quick tier misses.
