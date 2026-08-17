#!/usr/bin/env python3
"""
Helper to sort lines in text file. Similar to 'sort' but no dependence on locale or presence of 'sort' in PATH.
With --repl, applies the test replacements ($ACC_REPLS) as sort key for stable output across different environments.
"""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from repls import compile_repls, replace_all

use_repl = "--repl" in sys.argv[1:]

lines = sys.stdin.readlines()

if use_repl:
    patterns = compile_repls()
    lines.sort(key=lambda line: replace_all(patterns, line))
else:
    lines.sort()

sys.stdout.write("".join(lines))
