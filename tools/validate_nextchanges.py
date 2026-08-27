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

# README.md is allowed both at the .nextchanges root (the docs) and inside each
# section directory: the release renderer skips it, so a committed README.md
# keeps otherwise-empty section directories present in git without being
# mistaken for a fragment.
README = "README.md"

# nextversion.go embeds the version file above so the build can report the next
# release version. It lives here because go:embed cannot reach a parent
# directory, and keeping it here avoids a second copy of the version that could
# drift. The release renderer only reads *.md fragments, so it ignores this.
NEXTVERSION_GO = "nextversion.go"


def is_valid_semver(version_str):
    """Check if a string is a valid semantic version.

    Valid formats (bare or v-prefixed, with optional pre-release/metadata):
    >>> is_valid_semver("1.0.0")
    True
    >>> is_valid_semver("v1.0.0")
    True
    >>> is_valid_semver("1.2.3-alpha")
    True
    >>> is_valid_semver("1.2.3+build")
    True
    >>> is_valid_semver("1.2.3-rc.1+build.123")
    True

    Invalid formats:
    >>> is_valid_semver("1.0")
    False
    >>> is_valid_semver("v1.0")
    False
    >>> is_valid_semver("not-a-version")
    False
    >>> is_valid_semver("")
    False

    Whitespace is stripped, so "1.0.0" with leading/trailing whitespace is valid:
    >>> is_valid_semver("  1.0.0  ")
    True
    """
    return bool(SEMVER_RE.match(version_str.strip()) if version_str else False)


def load_sections(root):
    """Return the section slugs from .codegen.json, in changelog order.

    A missing or malformed .codegen.json raises: it is not something a PR author
    can be at fault for, so it crashes with the original traceback rather than
    being reported as a fragment problem."""
    codegen = json.loads((root / CODEGEN_FILE).read_text(encoding="utf-8"))

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

        # Root-level: only the version file and root documentation belong here. This prevents
        # someone accidentally putting a .md into .nextchanges thinking it would be picked up.
        if len(rel.parts) == 1:
            if name not in (VERSION_FILE, README, NEXTVERSION_GO):
                problems.append((path, "unexpected file at .nextchanges root"))
            continue

        # Section-level: .nextchanges/<section>/<file>.
        if len(rel.parts) == 2 and rel.parts[0] in known_sections:
            # README.md holds section docs and keeps the directory in git; the
            # renderer skips it, so it is not treated as a fragment.
            if name == README:
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
    elif not is_valid_semver(version_path.read_text(encoding="utf-8")):
        problems.append((version_path, "not a valid semver version (e.g. 1.4.0)"))
    return problems


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--root", type=pathlib.Path, default=pathlib.Path.cwd(), help="repository root")
    args = parser.parse_args(argv)

    changelog_dir = args.root / CHANGELOG_DIR
    if not changelog_dir.is_dir():
        return

    sections = load_sections(args.root)

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
