---
description: New changelog entries go in .nextchanges/, not CHANGELOG.md
globs:
  - "CHANGELOG.md"
  - ".nextchanges/**"
---

**RULE: Do not add new release entries to `CHANGELOG.md`.** For a user-visible change, add a fragment file under `.nextchanges/<section>/` (sections: `notable-changes`, `cli`, `air`, `bundles`, `dependency-updates`, `api-changes`). Each PR adds its own file, so entries never conflict. `CHANGELOG.md` is generated automatically from these fragments by the release process — never hand-edit it. See `.nextchanges/README.md`.

**RULE: Add one entry for each user-visible change, not one for each PR.** When a feature lands across a stack of PRs, add a single fragment for the net result in the last PR of the stack, linking that PR. Steps of a feature that is not released yet, such as refactors, follow-up fixes and tests, get no entry of their own.
