#!/usr/bin/python3
import sys
import os
import re
import argparse


parser = argparse.ArgumentParser()
parser.add_argument('regex')
parser.add_argument('--expect', type=int)
args = parser.parse_args()

regex = re.compile(args.regex)
count = 0

for root, dirs, files in os.walk("."):
    for filename in files:
        path = os.path.join(root, filename).lstrip('./')
        if regex.search(path):
            print(path)
            count += 1

if args.expect is not None:
    if args.expect != count:
        sys.exit(f'Expected {args.expect}, got {count}')
