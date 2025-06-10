#!/usr/bin/env python3
"""
Drop in replacement for golangci-lint that runs it only on changed packages.

Changes are calculated as diff against HEAD with a fallback to diff against main.
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
    parser.add_argument("--main", action="store_true", help="Detect changes by running diff against main")
    parser.add_argument("--head", action="store_true", help="Detect changes by running diff against HEAD")
    parser.add_argument("args", nargs=argparse.REMAINDER, help="golangci-lint command and options")
    args = parser.parse_args()

    if not args.main and not args.head:
        args.main = True
        args.head = True

    gitroot = parse_lines(["git", "rev-parse", "--show-toplevel"])[0]
    os.chdir(gitroot)

    changed = []

    if args.head:
        changed = parse_lines(["git", "diff", "--name-only", "HEAD", "--", "."])

    if not changed and args.main:
        changed = parse_lines(["git", "diff", "--name-only", "refs/remotes/origin/main", "--", "."])

    dirs = set()
    for filename in changed:
        # We need to pass packages to golangci-lint, not individual files
        if filename.endswith(".go"):
            d = os.path.dirname(filename)
            dirs.add(d)
        elif "/testdata/" in filename:
            d = filename.split("/testdata/")[0]
            dirs.add(d)

    if not dirs:
        sys.exit(0)

    dirs = ["./" + d for d in sorted(dirs)]

    cmd = ["golangci-lint"] + args.args + dirs
    print("+ " + " ".join(cmd), file=sys.stderr, flush=True)
    os.execvp(cmd[0], cmd)


if __name__ == "__main__":
    main()
