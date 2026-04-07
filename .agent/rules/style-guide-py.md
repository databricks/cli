---
description: Style Guide for Python
globs: "**/*.py"
paths:
  - "**/*.py"
---

# Style Guide for Python

## General guidance

When writing Python scripts, we bias for conciseness. We think of Python in this code base as scripts.
- use Python 3.11
- Do not catch exceptions to make nicer messages, only catch if you can add critical information
- use pathlib.Path in almost all cases over os.path unless it makes code longer
- Do not add redundant comments.
- Try to keep your code small and the number of abstractions low.
- After done, format your code with `ruff format -n <path>`
- Use `#!/usr/bin/env python3` shebang.
