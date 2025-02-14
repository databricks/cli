#!/usr/bin/env python3
"""This script implements "diff -r -U2 dir1 dir2" but applies replacements first"""

import sys
import difflib
import json
import re
from pathlib import Path


def replaceAll(patterns, s):
    for comp, new in patterns:
        s = comp.sub(new, s)
    return s


def main():
    d1, d2 = sys.argv[1:]
    d1, d2 = Path(d1), Path(d2)

    with open("repls.json") as f:
        repls = json.load(f)

    patterns = []
    for r in repls:
        try:
            c = re.compile(r["Old"])
            patterns.append((c, r["New"]))
        except re.error as e:
            print(f"Regex error for pattern {r}: {e}", file=sys.stderr)

    files1 = [str(p.relative_to(d1)) for p in d1.rglob("*") if p.is_file()]
    files2 = [str(p.relative_to(d2)) for p in d2.rglob("*") if p.is_file()]

    set1 = set(files1)
    set2 = set(files2)

    for f in sorted(set1 | set2):
        p1 = d1 / f
        p2 = d2 / f
        if f not in set2:
            print(f"Only in {d1}: {f}")
        elif f not in set1:
            print(f"Only in {d2}: {f}")
        else:
            a = [replaceAll(patterns, x) for x in p1.read_text().splitlines(True)]
            b = [replaceAll(patterns, x) for x in p2.read_text().splitlines(True)]
            if a != b:
                p1_str = p1.as_posix()
                p2_str = p2.as_posix()
                for line in difflib.unified_diff(a, b, p1_str, p2_str, "", "", 2):
                    print(line, end="")


if __name__ == "__main__":
    main()
