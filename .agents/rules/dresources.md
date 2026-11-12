---
description: Rules for implementing resources in bundle/direct/dresources
globs: bundle/direct/dresources/**/*.go
paths:
  - "bundle/direct/dresources/**/*.go"
---

**RULE: Before implementing or modifying a resource, read `bundle/direct/dresources/README.md`.** It covers constraints, field classification, update mask rules, async patterns, testing requirements, and state compatibility.

**RULE: When adding a new resource type, follow `.agents/skills/author-new-resource/SKILL.md` and run its API readiness gate before writing code.** Composite list keys, `OverrideChangeDesc` in a new resource, hand-written rules that mirror missing proto annotations, and engine changes to accommodate an API shape are API problems; raise them upstream instead of encoding them here.
