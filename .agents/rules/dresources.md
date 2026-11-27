---
description: Rules for implementing resources in bundle/direct/dresources
globs:
  - "bundle/direct/dresources/**/*.go"
  - "bundle/config/resources/*.go"
paths:
  - "bundle/direct/dresources/**/*.go"
  - "bundle/config/resources/*.go"
---

**RULE: Before implementing or modifying a resource or its config fields, read `bundle/direct/dresources/README.md`.** It covers constraints, field classification, update mask rules, async patterns, testing requirements, and state compatibility.
