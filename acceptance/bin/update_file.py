#!/usr/bin/env python3
"""
Usage: update_file.py FILENAME OLD NEW

Replace all strings OLD with NEW in FILENAME.

If OLD is not found in FILENAME, the script reports error.

The file is read and written as bytes so that only the matched bytes change.
Text mode on Windows rewrites every line ending as CRLF, which changes the
file's content hash and upload payload.
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

with open(filename, "rb") as fobj:
    data = fobj.read()

old_bytes = old.encode()
new_bytes = new.encode()

# Match the file's line endings so multi-line OLD/NEW work on CRLF checkouts.
if b"\r\n" in data:
    old_bytes = old_bytes.replace(b"\n", b"\r\n")
    new_bytes = new_bytes.replace(b"\n", b"\r\n")

newdata = data.replace(old_bytes, new_bytes)
if newdata == data:
    sys.exit(f"{old=} not found in {filename=}\n{data.decode(errors='replace')}")
with open(filename, "wb") as fobj:
    fobj.write(newdata)
