#!/usr/bin/env python3
"""
Edit a `comment`/`description` scalar in a generated databricks.yml so a redeploy is an
in-place update, not a recreate. Used by the `update` invariant.

Most resources take a `comment`/`description` edit as an in-place update, but some
classify it as immutable (recreate on change) in the direct engine's resource spec
(e.g. model_serving_endpoints.description). Editing such a field replans as a recreate,
which the update invariant would wrongly flag as a bug, so we skip it and pick a mutable
field elsewhere (or report none).

gen_fuzz_config.py emits one scalar per line as `key: <json>`, so a regex match suffices
(no YAML dependency for the edit itself); the immutable set is read from resources.yml.

  edit_fuzz_config.py PATH            edit in place; exit 1 if no editable field
  edit_fuzz_config.py PATH --detect   exit 0 if an editable field exists, else 1
"""

import argparse
import re
import sys
from pathlib import Path

# Allow an optional "- " so a comment/description that is the first key of a list-item
# dict still matches; the captured prefix is preserved verbatim on rewrite.
FIELD_RE = re.compile(r'^(\s*(?:- )?)(comment|description): (".*")\s*$')

# A resource type header directly under `resources:` (two-space indent, as emitted by
# gen_fuzz_config.py and the curated templates).
TYPE_RE = re.compile(r"^  ([\w-]+):\s*$")

NEW_VALUE = '"fuzz_edited_value"'

# resources.yml is the source of truth for field mutability; acceptance/bin sits two
# levels below the repo root, and the real dir (not a copy) is on PATH, so __file__
# resolves here.
RESOURCES_YML = Path(__file__).resolve().parents[2] / "bundle" / "direct" / "dresources" / "resources.yml"


def immutable_fields():
    """Map resource type -> set of fields that recreate on change (immutable).

    resources.yml has a fixed two-space layout (`resources:` -> `  <type>:` ->
    `    recreate_on_changes:` -> `      - field: <path>`), so a small line parser
    avoids a YAML dependency the harness's Python does not have.
    """
    result = {}
    current_type = None
    in_recreate = False
    for raw in RESOURCES_YML.read_text().splitlines():
        if not raw.strip() or raw.lstrip().startswith("#"):
            continue
        indent = len(raw) - len(raw.lstrip())
        stripped = raw.strip()
        if indent == 2 and stripped.endswith(":"):
            current_type = stripped[:-1]
            in_recreate = False
        elif indent == 4 and stripped.endswith(":"):
            in_recreate = stripped == "recreate_on_changes:"
        elif in_recreate and current_type:
            m = re.match(r"-\s*field:\s*(\S+)", stripped)
            if m:
                result.setdefault(current_type, set()).add(m.group(1))
    return result


def find_line(lines, immutable):
    current_type = None
    in_resources = False
    for i, line in enumerate(lines):
        stripped = line.rstrip("\n")
        # Track the enclosing resource type so an immutable comment/description is skipped.
        if stripped == "resources:":
            in_resources = True
            current_type = None
            continue
        if in_resources:
            m_type = TYPE_RE.match(stripped)
            if m_type:
                current_type = m_type.group(1)
            elif stripped and not stripped[0].isspace():
                in_resources = False
                current_type = None
        m = FIELD_RE.match(line)
        if m and m.group(2) not in immutable.get(current_type, ()):
            return i, m
    return -1, None


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("path")
    parser.add_argument("--detect", action="store_true", help="only check, don't edit")
    args = parser.parse_args()

    with open(args.path) as f:
        lines = f.readlines()

    i, m = find_line(lines, immutable_fields())
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
