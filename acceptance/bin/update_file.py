#!/usr/bin/env python3
"""
Usage: update_file.py FILENAME OLD NEW

Replace all strings OLD with NEW in FILENAME.

If OLD is not found in FILENAME, the script reports error.
"""

import sys

filename, old, new = sys.argv[1:]
data = open(filename).read()
newdata = data.replace(old, new)
if newdata == data:
    sys.exit(f"{old=} not found in {filename=}\n{data}")
with open(filename, "w") as fobj:
    fobj.write(newdata)
