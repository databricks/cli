#!/usr/bin/env python3
import sys

must_find = []
must_not_find = []
for arg in sys.argv[1:]:
    if arg.startswith("!"):
        must_not_find.append(arg[1:])
    else:
        must_find.append(arg)

found = set()

for line in sys.stdin:
    sys.stdout.write(line)
    for t in must_find:
        if t in line:
            found.add(t)
    for t in must_not_find:
        if t in line:
            sys.stderr.write(f"contains error: {t!r} was not expected\n")

sys.stdout.flush()

not_found = set(must_find) - found
for item in sorted(not_found):
    sys.stderr.write(f"contains error: {item!r} not found in the output.\n")
