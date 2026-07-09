#!/usr/bin/env python3
import os
import sys

errors = 0

for filename in sys.argv[1:]:
    if os.path.exists(filename):
        sys.stderr.write(f"Unexpected: {filename} exists.\n")
        errors += 1

if errors:
    sys.exit(1)
