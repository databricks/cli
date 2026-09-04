---
description: Rules for authoring PyDABs resource acceptance tests
globs: acceptance/bundle/python/**
paths:
  - "acceptance/bundle/python/**"
---

**RULE: Before adding a PyDABs resource acceptance test, read `acceptance/bundle/python/README.md`.** It covers the `<plural>-support/` fixture layout, how to source and adapt realistic field values, the version/engine `test.toml` knobs, and the determinism re-run. Every PyDABs resource needs one (enforced by `test_python_support_coverage`).
