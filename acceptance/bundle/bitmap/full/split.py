#!/usr/bin/env python3
"""Split the bitmap schema from stdin: resources.<type>.* goes to output/<type>.txt, the rest to output/root.txt.

Each line is "<path>\t<bit>", so a file still shows where each of its fields sits in the schema."""

import os
import sys

files = {}
os.makedirs("output", exist_ok=True)
for bit, line in enumerate(sys.stdin):
    path = line.rstrip("\n")
    parts = path.split(".")
    name = parts[1] if parts[0] == "resources" and len(parts) > 1 else "root"
    if name not in files:
        files[name] = open(f"output/{name}.txt", "w", newline="\n")
    files[name].write(f"{path}\t{bit}\n")
for f in files.values():
    f.close()
