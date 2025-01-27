#!/usr/bin/env python3
"""
Helper to sort lines in text file. Similar to 'sort' but no dependence on locale or presence of 'sort' in PATH.
"""

import sys

lines = sys.stdin.readlines()
lines.sort()
sys.stdout.write("".join(lines))
