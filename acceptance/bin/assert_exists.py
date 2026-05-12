#!/usr/bin/env python3
import os, sys

errors = 0

for filename in sys.argv[1:]:
    if not os.path.exists(filename):
        sys.stderr.write(f"Unexpected: {filename} does not exist.\n")
        errors += 1

if errors:
    sys.exit(1)
