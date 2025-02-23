#!/usr/bin/env python3
"""
Helper to sort blocks in text file. A block is a set of lines separated from others by empty line.

This is to workaround non-determinism in the output.
"""

import sys

blocks = []

for line in sys.stdin:
    if not line.strip():
        if blocks and blocks[-1]:
            blocks.append("")
        continue
    if not blocks:
        blocks.append("")
    blocks[-1] += line

blocks.sort()
print("\n".join(blocks))
