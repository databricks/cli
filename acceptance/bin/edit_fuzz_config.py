#!/usr/bin/env python3
"""
Edit a `comment`/`description` scalar in a generated databricks.yml so a redeploy is an
in-place update, not a recreate. Used by the `update` invariant. These fields are safe
to edit across resource types.

gen_fuzz_config.py emits one scalar per line as `key: <json>`, so a regex match suffices
(no YAML dependency).

  edit_fuzz_config.py PATH            edit in place; exit 1 if no editable field
  edit_fuzz_config.py PATH --detect   exit 0 if an editable field exists, else 1
"""

import argparse
import re
import sys

# Allow an optional "- " so a comment/description that is the first key of a list-item
# dict still matches; the captured prefix is preserved verbatim on rewrite.
FIELD_RE = re.compile(r'^(\s*(?:- )?)(comment|description): (".*")\s*$')

NEW_VALUE = '"fuzz_edited_value"'


def find_line(lines):
    for i, line in enumerate(lines):
        m = FIELD_RE.match(line)
        if m:
            return i, m
    return -1, None


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("path")
    parser.add_argument("--detect", action="store_true", help="only check, don't edit")
    args = parser.parse_args()

    with open(args.path) as f:
        lines = f.readlines()

    i, m = find_line(lines)
    if m is None:
        sys.exit(1)
    if args.detect:
        return

    prefix, key, _ = m.groups()
    lines[i] = f"{prefix}{key}: {NEW_VALUE}\n"
    with open(args.path, "w") as f:
        f.writelines(lines)
    sys.stderr.write(f"edited {key} at line {i + 1}\n")


if __name__ == "__main__":
    main()
