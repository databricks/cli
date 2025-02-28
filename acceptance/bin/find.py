#!/usr/bin/env python3
import sys
import os
import re
import argparse


parser = argparse.ArgumentParser()
parser.add_argument("regex")
parser.add_argument("--expect", type=int)
args = parser.parse_args()

regex = re.compile(args.regex)
result = []

for root, dirs, files in os.walk("."):
    for filename in files:
        path = os.path.join(root, filename).lstrip("./\\").replace("\\", "/")
        if regex.search(path):
            result.append(path)

result.sort()
for item in result:
    print(item)
sys.stdout.flush()

if args.expect is not None:
    if args.expect != len(result):
        sys.exit(f"Expected {args.expect}, got {len(result)}")
