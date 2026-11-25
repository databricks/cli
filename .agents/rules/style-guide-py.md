---
description: Style Guide for Python
globs: **/*.py
paths:
  - "**/*.py"
---

# Style Guide for Python

Python outside `python/` (tools, acceptance helpers) is written as scripts: keep it small, minimize abstractions, and use `#!/usr/bin/env python3`. `python/` is the PyDABs library and supports the Python versions in `python/pyproject.toml`.

**RULE: Do not catch exceptions just to add a nicer message.** Only catch if you can add critical information the caller can't produce.

Format with `./task fmt-q`.
