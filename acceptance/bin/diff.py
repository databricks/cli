#!/usr/bin/env python3
"""This script implements "diff -r -U2 dir1 dir2" but it applies replacements from repls.json to files found in dir2."""

import sys
import os
import difflib
import json
import re


def replaceAll(patterns, s):
    for comp, new in patterns:
        s = comp.sub(new, s)
    return s


def main():
    d1, d2 = sys.argv[1:]

    with open("repls.json") as f:
        repls = json.load(f)

    patterns = []
    for r in repls:
        try:
            c = re.compile(r["Old"])
            patterns.append((c, r["New"]))
        except re.error as e:
            print(f"Regex error for pattern {r}: {e}", file=sys.stderr)

    files1 = []
    for root, dirs, fs in os.walk(d1):
        for f in fs:
            files1.append(os.path.relpath(os.path.join(root, f), d1))

    files2 = []
    for root, dirs, fs in os.walk(d2):
        for f in fs:
            files2.append(os.path.relpath(os.path.join(root, f), d2))

    set1 = set(files1)
    set2 = set(files2)

    for f in sorted(set1 | set2):
        p1 = os.path.join(d1, f)
        p2 = os.path.join(d2, f)
        if f not in set2:
            print(f"Only in {d1}: {f}")
        elif f not in set1:
            print(f"Only in {d2}: {f}")
        else:
            a = [replaceAll(patterns, x) for x in open(p1).readlines()]
            b = [replaceAll(patterns, x) for x in open(p2).readlines()]
            if a != b:
                p1 = p1.replace("\\", "/")
                p2 = p2.replace("\\", "/")
                for line in difflib.unified_diff(a, b, p1, p2, "", "", 2):
                    print(line, end="")


if __name__ == '__main__':
    main()
