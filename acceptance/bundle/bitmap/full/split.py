#!/usr/bin/env python3
"""Split the bitmap schema from stdin: resources.<type>.* goes to output/<type>.txt, the rest to output/root.txt.

Each line is "<bit> <path>", so a file still shows where each of its fields sits in the schema. Bits
are right-aligned to the width of the largest one, the same in every file."""

import os
import sys

paths = sys.stdin.read().splitlines()
width = len(str(len(paths) - 1))
files = {}
os.makedirs("output", exist_ok=True)
for bit, path in enumerate(paths):
    parts = path.split(".")
    name = parts[1] if parts[0] == "resources" and len(parts) > 1 else "root"
    if name not in files:
        files[name] = open(f"output/{name}.txt", "w", newline="\n")
    files[name].write(f"{bit:>{width}} {path}\n")
for f in files.values():
    f.close()
