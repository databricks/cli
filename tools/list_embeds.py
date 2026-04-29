#!/usr/bin/env python3
"""Print a single brace-expansion glob of every //go:embed target in the repo.

Used as the EMBED_SOURCES dynamic var in Taskfile.yml so tasks can reference
embed dependencies without hand-maintaining a list. Testdata directories are
covered by a separate static `**/testdata/**` glob, not this script.
"""

import os
import subprocess
import sys

EMBED = "//go:embed "


def main():
    root = os.path.normpath(os.path.join(os.path.dirname(os.path.abspath(sys.argv[0])), ".."))
    os.chdir(root)
    # git grep only scans tracked files — new //go:embed directives in untracked
    # files are silently missed until the file is staged or committed.
    out = subprocess.check_output(
        ["git", "grep", "--no-color", "-E", "^" + EMBED, "--", "*.go"],
        text=True,
    )
    paths = set()
    for line in out.splitlines():
        file_path, _, directive = line.partition(":")
        if not directive.startswith(EMBED):
            continue
        directory = os.path.dirname(file_path)
        for pat in directive[len(EMBED) :].split():
            if pat.startswith("all:"):
                pat = pat[len("all:") :]
            full = os.path.join(directory, pat) if directory else pat
            if os.path.isdir(full):
                paths.add(full + "/**")
            elif os.path.isfile(full):
                paths.add(full)
    if paths:
        print("{" + ",".join(sorted(paths)) + "}")


if __name__ == "__main__":
    main()
