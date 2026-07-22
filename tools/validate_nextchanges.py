#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""Validate changelog fragment placement under ``.nextchanges/``.

Each PR adds its own file under ``.nextchanges/<section>/`` (see
``.nextchanges/README.md``). This fails CI when a fragment is misfiled or
empty, so it is caught up front rather than silently dropped when the release
renders ``.nextchanges/`` into ``CHANGELOG.md`` (see
``internal/genkit/tagging.py``).
"""

import argparse
import json
import pathlib
import re
import sys

CHANGELOG_DIR = ".nextchanges"
CODEGEN_FILE = ".codegen.json"
NEXTCHANGES_SECTIONS_KEY = "nextchanges_sections"

# .nextchanges/version holds the next release version; the release reads it and
# bumps it. Accept a bare semver (optionally v-prefixed), e.g. 1.4.0 / v1.4.0.
VERSION_FILE = "version"
SEMVER_RE = re.compile(r"^v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$")

# Non-fragment files allowed to sit alongside fragments at any depth.
# Note that tagging workflow accepts README.md inside fragments, but we reject it here for clarity.
SCAFFOLDING = ("README.md", ".gitkeep")


def load_sections(root):
    codegen_path = root / CODEGEN_FILE
    try:
        codegen = json.loads(codegen_path.read_text(encoding="utf-8"))
    except FileNotFoundError as err:
        raise ValueError(f"{CODEGEN_FILE} is missing") from err
    except json.JSONDecodeError as err:
        raise ValueError(f"{CODEGEN_FILE} is not valid JSON: {err}") from err

    sections = codegen.get(NEXTCHANGES_SECTIONS_KEY)
    if not isinstance(sections, dict) or not sections:
        raise ValueError(f"{CODEGEN_FILE} must define a non-empty {NEXTCHANGES_SECTIONS_KEY} object")

    return tuple(sections)


def find_problems(changelog_dir, sections):
    """Return a list of ``(path, message)`` for anything unexpected under
    ``.nextchanges/``: files that aren't a section fragment or known scaffolding,
    empty fragments, and a missing/malformed version file."""
    problems = []
    known_sections = set(sections)
    for path in sorted(changelog_dir.rglob("*")):
        if path.is_dir():
            continue
        rel = path.relative_to(changelog_dir)
        name = path.name

        # Root-level: only the version file and scaffolding belong here.
        if len(rel.parts) == 1:
            if name != VERSION_FILE and name not in SCAFFOLDING:
                problems.append((path, "unexpected file at .nextchanges root"))
            continue

        # Section-level: .nextchanges/<section>/<file>.
        if len(rel.parts) == 2 and rel.parts[0] in known_sections:
            if name in SCAFFOLDING:
                continue
            if not name.endswith(".md"):
                problems.append((path, "unexpected file (fragments must be *.md)"))
            elif not path.read_text(encoding="utf-8").strip():
                problems.append((path, "empty fragment"))
            continue

        # Wrong depth or an unknown section directory.
        problems.append((path, "not in a known section directory"))

    version_path = changelog_dir / VERSION_FILE
    if not version_path.is_file():
        problems.append((version_path, "missing; expected the next release version (e.g. 1.4.0)"))
    elif not SEMVER_RE.match(version_path.read_text(encoding="utf-8").strip()):
        problems.append((version_path, "not a valid semver version (e.g. 1.4.0)"))
    return problems


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--root", type=pathlib.Path, default=pathlib.Path.cwd(), help="repository root")
    args = parser.parse_args(argv)

    changelog_dir = args.root / CHANGELOG_DIR
    if not changelog_dir.is_dir():
        return

    try:
        sections = load_sections(args.root)
    except ValueError as err:
        print(err, file=sys.stderr)
        sys.exit(1)

    problems = find_problems(changelog_dir, sections)
    if problems:
        for path, msg in problems:
            print(f"{path}: {msg}", file=sys.stderr)
        print(f"\nFragments must live at {CHANGELOG_DIR}/<section>/<name>.md", file=sys.stderr)
        print(f"Valid sections: {', '.join(sections)}", file=sys.stderr)
        print(f"{CHANGELOG_DIR}/{VERSION_FILE} must hold the next release version.", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
