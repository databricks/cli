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

# Read raw and normalize CRLF to LF so the search string (always LF in the
# script) matches on Windows, where the file is checked out with CRLF. Write
# with newline="" so Python does not turn the \n back into \r\n: the acceptance
# harness treats every file as LF, and a stray \r would change any uploaded
# bytes or content hash a test later asserts.
data = open(filename, newline="").read().replace("\r\n", "\n")
newdata = data.replace(old, new)
if newdata == data:
    sys.exit(f"{old=} not found in {filename=}\n{data}")
with open(filename, "w", newline="") as fobj:
    fobj.write(newdata)
