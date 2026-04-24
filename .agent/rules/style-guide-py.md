---
description: Style Guide for Python
globs: **/*.py
paths:
  - "**/*.py"
---

# Style Guide for Python

## General guidance

Python in this codebase is written as scripts. Bias for conciseness.

**RULE: Use Python 3.11.**

**RULE: Use `#!/usr/bin/env python3` as the shebang.**

**RULE: Prefer `pathlib.Path` over `os.path`.** Exception: when it would make the code longer.

**RULE: Do not catch exceptions just to add a nicer message.** Only catch if you can add critical information the caller can't produce.

**RULE: Avoid redundant comments.**

**RULE: Keep code small and minimize abstractions.**

**RULE: Format with `ruff format -n <path>` before committing.**
