#!/usr/bin/env python3
"""
Print the lineage hash of the state, as the CLI reports it in the user agent (lineage/...)
and telemetry (state_lineage). Update ACC_REPLS to replace it with [LINEAGE_HASH].

Use it after the deploy that mints a fresh lineage; fixture lineages hash to stable values
and need no replacement.

Usage: read_lineage.py [-t target]
"""

import argparse
import hashlib
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from add_repl import add_repl
from print_state import get_state_file

ALPHABET = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"


def shortid_hash(s):
    """Same as libs/shortid.Hash: first 64 bits of SHA-256, as 11 base62 digits."""
    n = int.from_bytes(hashlib.sha256(s.encode()).digest()[:8], "big")
    digits = []
    for _ in range(11):
        n, d = divmod(n, len(ALPHABET))
        digits.append(ALPHABET[d])
    return "".join(reversed(digits))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("-t", "--target")
    args = parser.parse_args()

    filename = get_state_file(args.target, backup=False)
    lineage = json.loads(open(filename).read())["lineage"]
    value = shortid_hash(lineage)
    add_repl(value, "LINEAGE_HASH")
    print(value)


if __name__ == "__main__":
    main()
