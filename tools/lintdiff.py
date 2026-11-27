#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""
Drop in replacement for golangci-lint that runs it only on changed packages.

Changes are calculated as diff against the merge base with main by default, use --ref or -H/--head to change this.
"""

import argparse
import os
import subprocess
import sys

# Each entry is a path prefix: "tools" also covers "tools/task", "tools/other", etc.
NESTED_MODULES = ("tools",)


def in_nested_module(path):
    """Check if a path is a nested module or lives under one.

    A path matches a module exactly or as a `/`-separated prefix, but a sibling
    that merely shares a leading substring does not:

    >>> in_nested_module("tools")
    True

    >>> in_nested_module("tools/task/subtask.go")
    True

    >>> in_nested_module("cmd/bundle")
    False

    >>> in_nested_module("toolz")
    False
    """
    return any(path == m or path.startswith(m + "/") for m in NESTED_MODULES)


def parse_lines(cmd):
    # print("+ " + " ".join(cmd), file=sys.stderr, flush=True)
    result = subprocess.run(cmd, stdout=subprocess.PIPE, encoding="utf-8")
    if result.returncode != 0:
        return
    return [x.strip() for x in result.stdout.split("\n") if x.strip()]


def go_dirs(paths, filter_nested):
    """Return ./dir patterns for the directories of the Go files among paths.

    Skips testdata and, with filter_nested, nested modules. Keeps only
    directories that still contain a Go file, so a package whose last Go file
    was deleted is dropped.
    """
    dirs = set()
    for filename in paths:
        if "/testdata/" in filename:
            continue
        if filename.endswith(".go"):
            d = os.path.dirname(filename)
            if filter_nested and in_nested_module(d):
                continue
            dirs.add(d)

    return ["./" + d for d in sorted(dirs) if os.path.isdir(d) and any(n.endswith(".go") for n in os.listdir(d))]


def changed_go_dirs(ref, filter_nested=True):
    """Return ./dir patterns for the Go packages changed since the merge base with ref, or None if git diff fails.

    Diff the merge base against the working tree so we only see files changed
    on this branch, not files that advanced on the ref since the branch point.
    Unlike testmask's three-dot "ref...HEAD" form (tools/testmask/git.go), this
    includes the dirty tree's uncommitted changes. --no-renames lists both sides
    of a rename, so the package a file moved out of is selected too.
    """
    changed = parse_lines(["git", "diff", "--name-only", "--no-renames", "--merge-base", ref, "--", "."])
    if changed is None:
        return None
    return go_dirs(changed, filter_nested)


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

    cmd = args.args[:]

    # `golangci-lint run` typechecks against the target go.mod and errors on
    # paths under a different module; `fmt` walks the filesystem and is
    # cross-module safe. Apply the nested-module filter only for `run`.
    filter_nested = "run" in cmd

    # The diff includes the dirty tree's uncommitted changes, which is the point of --fix.
    # We need to pass packages to golangci-lint, not individual files.
    # QQQ for lint we should also pass all dependent packages
    dirs = changed_go_dirs(args.ref, filter_nested)

    if dirs is not None:
        if not dirs:
            sys.exit(0)

        cmd += dirs

    print("+ " + " ".join(cmd), file=sys.stderr, flush=True)
    os.execvp(cmd[0], cmd)


if __name__ == "__main__":
    main()
