#!/usr/bin/env python3
"""
Drop in replacement for golangci-lint that runs it only on changed packages.

Changes are calculated as diff against main by default, use --ref or -H/--head to change this.
"""

import os
import sys
import argparse
import subprocess


def parse_lines(cmd):
    # print("+ " + " ".join(cmd), file=sys.stderr, flush=True)
    result = subprocess.run(cmd, stdout=subprocess.PIPE, encoding="utf-8", check=True)
    return [x.strip() for x in result.stdout.split("\n") if x.strip()]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--ref", default="main", help="Reference to calculate diff against.")
    parser.add_argument("-H", "--head", action="store_true", help="Shortcut for '--ref HEAD' - test uncommitted changes only")
    parser.add_argument("args", nargs=argparse.REMAINDER, help="golangci-lint command and options")
    args = parser.parse_args()

    if not args.args:
        args.args = ["run"]

    if args.head:
        args.ref = "HEAD"

    gitroot = parse_lines(["git", "rev-parse", "--show-toplevel"])[0]
    os.chdir(gitroot)

    changed = parse_lines(["git", "diff", "--name-only", args.ref, "--", "."])

    # We need to pass packages to golangci-lint, not individual files.
    dirs = set()
    for filename in changed:
        if "/testdata/" in filename:
            continue
        if filename.endswith(".go"):
            d = os.path.dirname(filename)
            dirs.add(d)

    if not dirs:
        sys.exit(0)

    dirs = ["./" + d for d in sorted(dirs)]

    cmd = ["golangci-lint"] + args.args + dirs
    print("+ " + " ".join(cmd), file=sys.stderr, flush=True)
    os.execvp(cmd[0], cmd)


if __name__ == "__main__":
    main()
