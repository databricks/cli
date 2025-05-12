#!/usr/bin/env python3
"""
Usage: update_file.py FILENAME OLD NEW

Replace all strings OLD with NEW in FILENAME.

If OLD is not found in FILENAME, the script reports error.
"""

import sys

filename, old, new = sys.argv[1:]

# Acceptance tests by default keep output.txt open to append output to.
# Using update_file.py on output.txt can be flaky on windows, likely because
# of some internal buffering with the file handle. Thus we do not allow updating
# output.txt with this script.
#
# You are instead recommended to write the output to a different file and
# call update_file.py on that file.
assert filename != "output.txt"

data = open(filename).read()
newdata = data.replace(old, new)
if newdata == data:
    sys.exit(f"{old=} not found in {filename=}\n{data}")
with open(filename, "w") as fobj:
    fobj.write(newdata)
