#!/usr/bin/env python3
"""
Add entry to ACC_REPLS without clobbering existing ones.

If entry already exists, it'll add suffix in _<number> format.
"""

import os
import argparse
from pathlib import Path

ACC_REPLS = Path(os.environ["TEST_TMP_DIR"]) / "ACC_REPLS"


def get_repls():
    result = {}
    if ACC_REPLS.exists():
        for line in ACC_REPLS.open():
            value, repl = line.strip().rsplit(":", 1)
            result[repl] = value
    return result


def add_repl(value, repl):
    existing = get_repls()
    for extra in range(1, 100):
        if extra >= 2:
            r = f"{repl}_{extra}"
        else:
            r = repl
        if r in existing:
            continue
        with ACC_REPLS.open("a") as fobj:
            fobj.write(f"{value}:{r}\n")
        break


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("value")
    parser.add_argument("replacement")
    args = parser.parse_args()
    add_repl(args.value, args.replacement)


if __name__ == "__main__":
    main()
