#!/usr/bin/env python3
"""Split refschema output from stdin: resources.<type>.* goes to output/<type>.txt, the rest to output/root.txt."""

import os
import sys

files = {}
os.makedirs("output", exist_ok=True)
for line in sys.stdin:
    parts = line.split("\t", 1)[0].split(".")
    name = parts[1] if parts[0] == "resources" else "root"
    if name not in files:
        files[name] = open(f"output/{name}.txt", "w", newline="\n")
    files[name].write(line)
for f in files.values():
    f.close()
