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
import pathlib
import re
import sys

CHANGELOG_DIR = ".nextchanges"

# Known section subdirectories. Mirrors NEXTCHANGES_SECTIONS in
# internal/genkit/tagging.py — keep the two in sync.
SECTIONS = ("notable-changes", "cli", "bundles", "dependency-updates", "api-changes")

# .nextchanges/version holds the next release version; the release reads it and
# bumps it. Accept a bare semver (optionally v-prefixed), e.g. 1.4.0 / v1.4.0.
VERSION_FILE = "version"
SEMVER_RE = re.compile(r"^v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$")


def iter_fragment_files(changelog_dir):
    """Yield every ``*.md`` fragment under *changelog_dir*, excluding READMEs."""
    for path in sorted(changelog_dir.rglob("*.md")):
        if path.name == "README.md":
            continue
        yield path


def find_problems(changelog_dir):
    """Return a list of ``(path, message)`` for misplaced/empty fragments or a
    missing/malformed version file."""
    problems = []
    for path in iter_fragment_files(changelog_dir):
        rel = path.relative_to(changelog_dir)
        if len(rel.parts) != 2 or rel.parts[0] not in SECTIONS:
            problems.append((path, "not in a known section directory"))
        elif not path.read_text(encoding="utf-8").strip():
            problems.append((path, "empty fragment"))

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

    problems = find_problems(changelog_dir)
    if problems:
        for path, msg in problems:
            print(f"{path}: {msg}", file=sys.stderr)
        print(f"\nFragments must live at {CHANGELOG_DIR}/<section>/<name>.md", file=sys.stderr)
        print(f"Valid sections: {', '.join(SECTIONS)}", file=sys.stderr)
        print(f"{CHANGELOG_DIR}/{VERSION_FILE} must hold the next release version.", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
