#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""Fail if any acceptance/**/test.toml has no settings (empty or comment-only)."""

import sys
import tomllib
from pathlib import Path

empty = []
for path in sorted(Path("acceptance").rglob("test.toml")):
    with path.open("rb") as f:
        if not tomllib.load(f):
            empty.append(path)

if empty:
    print("test.toml files with no settings found (delete them instead of leaving them empty):")
    for p in empty:
        print(f"  {p.as_posix()}")
    sys.exit(1)
