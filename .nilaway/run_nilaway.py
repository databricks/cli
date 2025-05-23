#!/usr/bin/env python3
"""
Build nilaway then run it and filter out diagnostics on test files.
"""

import os
import sys
import re
import subprocess

args = sys.argv[1:]
directory = os.path.dirname(__file__)

os.chdir(directory)
if os.system("make"):
    sys.exit(1)

os.chdir("..")
cmd = [directory + "/custom-gcl", "-c", directory + "/.golangci.yaml", *args]
print("+ " + " ".join(cmd), flush=True)
result = subprocess.run(cmd, stdout=subprocess.PIPE, encoding="utf-8")
errors = 0
skip_block = False

for line in result.stdout.split("\n"):
    m = re.match(r"^([^: ]+):\d+:\d+: ", line)
    if m is not None:
        path = m.group(1)
        if path.endswith("_test.go"):
            # print("SKIP", line)
            skip_block = True
        else:
            skip_block = False
            errors += 1
    if skip_block:
        continue
    print(line)


print(f"{errors} errors.")
if errors:
    sys.exit(1)
