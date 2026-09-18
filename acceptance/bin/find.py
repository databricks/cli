#!/usr/bin/env python3
"""
Usage: find.py <regex>
Finds all files within current directory matching regex. The output is sorted and slashes are always forward.

If --expect N is provided, the number of matches must be N or error is printed.
If --include-dirs is provided, directories are matched and printed too (default: files only).
If --prune REGEX is provided, directories whose path matches REGEX are not descended into or printed.
"""

import argparse
import os
import re
import sys

parser = argparse.ArgumentParser()
parser.add_argument("regex")
parser.add_argument("--expect", type=int)
parser.add_argument("--include-dirs", action="store_true")
parser.add_argument("--prune")
args = parser.parse_args()

regex = re.compile(args.regex)
prune = re.compile(args.prune) if args.prune else None
result = []


def relpath(root, name):
    return os.path.join(root, name).replace("\\", "/").removeprefix("./")


for root, dirs, files in os.walk("."):
    if prune is not None:
        dirs[:] = [d for d in dirs if not prune.search(relpath(root, d))]
    names = files + dirs if args.include_dirs else files
    for name in names:
        path = relpath(root, name)
        if regex.search(path):
            result.append(path)

result.sort()
for item in result:
    print(item)
sys.stdout.flush()

if args.expect is not None:
    if args.expect != len(result):
        sys.exit(f"Expected {args.expect}, got {len(result)}")
