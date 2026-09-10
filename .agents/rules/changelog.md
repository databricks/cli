---
description: New changelog entries go in .nextchanges/, not CHANGELOG.md
globs:
  - "CHANGELOG.md"
  - ".nextchanges/**"
---

**RULE: Do not add new release entries to `CHANGELOG.md`.** For a user-visible change, add a fragment file under `.nextchanges/<section>/` (sections: `notable-changes`, `cli`, `bundles`, `dependency-updates`, `api-changes`). Each PR adds its own file, so entries never conflict. `CHANGELOG.md` is generated automatically from these fragments by the release process — never hand-edit it. See `.nextchanges/README.md`.
