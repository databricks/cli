#!/usr/bin/env python3
import sys

targets = sys.argv[1:]
assert targets
found = set()

for line in sys.stdin:
    sys.stdout.write(line)
    for t in targets:
        if t in line:
            found.add(t)

not_found = set(targets) - found
for item in sorted(not_found):
    sys.stderr.write(f"contains error: {item!r} not found in the output.\n")
