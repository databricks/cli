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
failed = False

for line in sys.stdin:
    sys.stdout.write(line)
    for t in must_find:
        if t in line:
            found.add(t)
    for t in must_not_find:
        if t in line:
            sys.stderr.write(f"contains error: {t!r} was not expected: {line.strip()!r}\n")
            failed = True

sys.stdout.flush()

not_found = set(must_find) - found
for item in sorted(not_found):
    sys.stderr.write(f"contains error: {item!r} not found in the output.\n")
    failed = True

# Exit non-zero so a failed assertion aborts the script (set -e -o pipefail)
# instead of silently baking the error line into the expected output.
if failed:
    sys.exit(1)
