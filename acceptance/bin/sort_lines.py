#!/usr/bin/env python3
"""
Helper to sort lines in text file. Similar to 'sort' but no dependence on locale or presence of 'sort' in PATH.
With --repl, applies TEST_TMP_DIR/repls.json replacements as sort key for stable output across different environments.
"""

import sys
import os
import json
import re
from pathlib import Path

use_repl = "--repl" in sys.argv[1:]

lines = sys.stdin.readlines()

if use_repl:
    repls = json.loads((Path(os.environ["TEST_TMP_DIR"]) / "repls.json").read_text())
    patterns = []
    for r in repls:
        try:
            patterns.append((re.compile(r["Old"]), r["New"]))
        except re.error as e:
            print(f"Regex error for pattern {r}: {e}", file=sys.stderr)

    def sort_key(line):
        for comp, new in patterns:
            line = comp.sub(new, line)
        return line

    lines.sort(key=sort_key)
else:
    lines.sort()

sys.stdout.write("".join(lines))
