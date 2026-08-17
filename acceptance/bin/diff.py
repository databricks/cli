#!/usr/bin/env python3
"""This script implements "diff -r -U2 dir1 dir2" but applies replacements first"""

import difflib
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from repls import compile_repls, replace_all


def main():
    d1, d2 = sys.argv[1:]
    d1, d2 = Path(d1), Path(d2)

    patterns = compile_repls()

    if d1.is_dir() and d2.is_dir():
        files1 = [str(p.relative_to(d1)) for p in d1.rglob("*") if p.is_file() and not p.name.startswith("LOG")]
        files2 = [str(p.relative_to(d2)) for p in d2.rglob("*") if p.is_file() and not p.name.startswith("LOG")]
    else:
        assert d1.is_file(), d1
        assert d2.is_file(), d2
        diff_files(patterns, d1, d2)
        return

    set1 = set(files1)
    set2 = set(files2)

    for f in sorted(set1 | set2):
        p1 = d1 / f
        p2 = d2 / f
        if f not in set2:
            print(f"Only in {d1}: {f}")
        elif f not in set1:
            print(f"Only in {d2}: {f}")
        else:
            diff_files(patterns, p1, p2)


def diff_files(patterns, p1, p2):
    a = replace_all(patterns, p1.read_text()).splitlines(True)
    b = replace_all(patterns, p2.read_text()).splitlines(True)
    if a != b:
        p1_str = p1.as_posix()
        p2_str = p2.as_posix()
        for line in difflib.unified_diff(a, b, p1_str, p2_str, "", "", 2):
            print(line.rstrip())


if __name__ == "__main__":
    main()
