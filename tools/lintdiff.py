#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""
Drop in replacement for golangci-lint that runs it only on changed packages.

Changes are calculated as diff against the merge base with main by default, use --ref or -H/--head to change this.
"""

import os
import sys
import argparse
import subprocess

# Each entry is a path prefix: "tools" also covers "tools/task", "tools/other", etc.
NESTED_MODULES = ("bundle/internal/tf/codegen", "tools")


def parse_lines(cmd):
    # print("+ " + " ".join(cmd), file=sys.stderr, flush=True)
    result = subprocess.run(cmd, stdout=subprocess.PIPE, encoding="utf-8")
    if result.returncode != 0:
        return
    return [x.strip() for x in result.stdout.split("\n") if x.strip()]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--ref", default="main", help="Reference to calculate diff against.")
    parser.add_argument(
        "-H", "--head", action="store_true", help="Shortcut for '--ref HEAD' - test uncommitted changes only"
    )
    parser.add_argument("args", nargs=argparse.REMAINDER, help="golangci-lint command and options")
    args = parser.parse_args()

    if not args.args:
        args.args = ["run"]

    if args.head:
        args.ref = "HEAD"

    # All paths are relative to repo root, so we need to ensure we're in the right directory.
    gitroot = parse_lines(["git", "rev-parse", "--show-toplevel"])
    if gitroot:
        os.chdir(gitroot[0])

    # Diff the merge base against the working tree so we only lint files changed
    # on this branch, not files that advanced on the ref since the branch point.
    # Unlike testmask's three-dot "ref...HEAD" form (tools/testmask/git.go), this
    # includes the dirty tree's uncommitted changes, which is the point of --fix.
    changed = parse_lines(["git", "diff", "--name-only", "--merge-base", args.ref, "--", "."])

    cmd = args.args[:]

    # `golangci-lint run` typechecks against the target go.mod and errors on
    # paths under a different module; `fmt` walks the filesystem and is
    # cross-module safe. Apply the nested-module filter only for `run`.
    filter_nested = "run" in cmd

    if changed is not None:

        def in_nested_module(path):
            return filter_nested and any(path == m or path.startswith(m + "/") for m in NESTED_MODULES)

        # We need to pass packages to golangci-lint, not individual files.
        # QQQ for lint we should also pass all dependent packages
        dirs = set()
        for filename in changed:
            if "/testdata/" in filename:
                continue
            if filename.endswith(".go"):
                d = os.path.dirname(filename)
                if in_nested_module(d):
                    continue
                dirs.add(d)

        dirs = ["./" + d for d in sorted(dirs) if os.path.exists(d)]

        if not dirs:
            sys.exit(0)

        cmd += dirs

    print("+ " + " ".join(cmd), file=sys.stderr, flush=True)
    os.execvp(cmd[0], cmd)


if __name__ == "__main__":
    main()
