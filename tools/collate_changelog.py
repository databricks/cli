#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""Collate ``.nextchanges/`` fragments into ``NEXT_CHANGELOG.md``.

Each PR adds its own file under ``.nextchanges/<section>/`` instead of editing
the shared ``NEXT_CHANGELOG.md``. Because two PRs never touch the same path,
they never produce a merge conflict. At release time this script folds every
fragment into the matching section of ``NEXT_CHANGELOG.md`` and deletes the
fragment files; the existing release tooling (``internal/genkit/tagging.py``)
then consumes ``NEXT_CHANGELOG.md`` unchanged.

Usage:
    collate_changelog.py            # collate fragments into NEXT_CHANGELOG.md
    collate_changelog.py --check    # validate fragment placement only (no writes)
"""

import argparse
import pathlib
import re
import sys

CHANGELOG_DIR = ".nextchanges"
NEXT_CHANGELOG = "NEXT_CHANGELOG.md"

# Section subdirectory -> ``### `` header text in NEXT_CHANGELOG.md, in the
# order sections appear in the file. The slug is the header lowercased with
# spaces replaced by hyphens; the mapping is explicit because "CLI" and
# "API Changes" don't round-trip through a simple title-case rule.
SECTIONS = (
    ("notable-changes", "Notable Changes"),
    ("cli", "CLI"),
    ("bundles", "Bundles"),
    ("dependency-updates", "Dependency updates"),
    ("api-changes", "API Changes"),
)

SECTION_SLUGS = {slug for slug, _ in SECTIONS}

# A level-2 or level-3 Markdown heading, i.e. a section boundary.
HEADING_RE = re.compile(r"#{2,3} ")


def normalize_entry(text):
    """Return *text* as a Markdown bullet, adding the leading ``* `` if absent.

    The leading marker is optional in a fragment so authors can write just the
    entry text. A ``-`` marker is normalized to ``*`` to match the changelog.

    >>> normalize_entry("Added a flag (#1).")
    '* Added a flag (#1).'
    >>> normalize_entry("* Already a bullet (#2).")
    '* Already a bullet (#2).'
    >>> normalize_entry("- Dash bullet (#3).")
    '* Dash bullet (#3).'

    Only the first line is marked; continuation lines are left untouched so an
    author can write a multi-line entry or several explicit bullets:

    >>> normalize_entry("First line.\\n  continued")
    '* First line.\\n  continued'
    """
    text = text.strip()
    first, _, rest = text.partition("\n")
    if first.startswith("* "):
        pass
    elif first.startswith("- "):
        first = "* " + first[2:]
    elif first in ("*", "-"):
        first = "*"
    else:
        first = "* " + first
    return first + ("\n" + rest if rest else "")


def insert_entries(changelog, header, entries):
    r"""Insert *entries* under the ``### {header}`` section of *changelog*.

    Entries are appended after any existing content in the section, before the
    blank line that precedes the next section. Existing lines are left byte for
    byte intact so the diff is minimal.

    >>> cl = "## Release v1.0.0\n\n### CLI\n\n### Bundles\n"
    >>> print(insert_entries(cl, "CLI", ["* Added a flag (#1)."]), end="")
    ## Release v1.0.0
    <BLANKLINE>
    ### CLI
    * Added a flag (#1).
    <BLANKLINE>
    ### Bundles

    Appends after content already present in the section:

    >>> cl = "### CLI\n* Existing (#1).\n\n### Bundles\n"
    >>> print(insert_entries(cl, "CLI", ["* New (#2)."]), end="")
    ### CLI
    * Existing (#1).
    * New (#2).
    <BLANKLINE>
    ### Bundles

    Works for the last section in the file:

    >>> print(insert_entries("### API Changes\n", "API Changes", ["* X (#1)."]), end="")
    ### API Changes
    * X (#1).
    """
    lines = changelog.split("\n")

    header_line = f"### {header}"
    try:
        start = next(i for i, line in enumerate(lines) if line.strip() == header_line)
    except StopIteration:
        raise SystemExit(f"section '{header_line}' not found in {NEXT_CHANGELOG}")

    # End of the section: the next heading, or end of file.
    end = len(lines)
    for i in range(start + 1, len(lines)):
        if HEADING_RE.match(lines[i].strip()):
            end = i
            break

    # Skip trailing blank lines so new entries attach directly to existing
    # content (or to the header when the section is empty).
    insert_at = end
    while insert_at - 1 > start and lines[insert_at - 1].strip() == "":
        insert_at -= 1

    lines[insert_at:insert_at] = entries
    return "\n".join(lines)


def iter_fragment_files(changelog_dir):
    """Yield every ``*.md`` fragment under *changelog_dir*, excluding READMEs."""
    for path in sorted(changelog_dir.rglob("*.md")):
        if path.name == "README.md":
            continue
        yield path


def find_misplaced(changelog_dir):
    """Return fragment paths that are not ``.nextchanges/<section>/<name>.md``."""
    misplaced = []
    for path in iter_fragment_files(changelog_dir):
        rel = path.relative_to(changelog_dir)
        if len(rel.parts) != 2 or rel.parts[0] not in SECTION_SLUGS:
            misplaced.append(path)
    return misplaced


def check(root):
    """Validate fragment placement. Returns a process exit code."""
    changelog_dir = root / CHANGELOG_DIR
    if not changelog_dir.is_dir():
        return 0

    problems = []
    for path in find_misplaced(changelog_dir):
        problems.append(f"{path}: not in a known section directory")
    for path in iter_fragment_files(changelog_dir):
        if not path.read_text(encoding="utf-8").strip():
            problems.append(f"{path}: empty fragment")

    if problems:
        for msg in problems:
            print(msg, file=sys.stderr)
        valid = ", ".join(slug for slug, _ in SECTIONS)
        print(f"\nFragments must live at {CHANGELOG_DIR}/<section>/<name>.md", file=sys.stderr)
        print(f"Valid sections: {valid}", file=sys.stderr)
        return 1
    return 0


def collate(root):
    """Fold fragments into NEXT_CHANGELOG.md and delete them."""
    changelog_dir = root / CHANGELOG_DIR
    next_changelog = root / NEXT_CHANGELOG

    misplaced = find_misplaced(changelog_dir) if changelog_dir.is_dir() else []
    if misplaced:
        for path in misplaced:
            print(f"{path}: not in a known section directory", file=sys.stderr)
        raise SystemExit(1)

    content = next_changelog.read_text(encoding="utf-8")
    consumed = []
    total = 0
    for slug, header in SECTIONS:
        section_dir = changelog_dir / slug
        if not section_dir.is_dir():
            continue
        entries = []
        for path in sorted(section_dir.glob("*.md")):
            if path.name == "README.md":
                continue
            entries.append(normalize_entry(path.read_text(encoding="utf-8")))
            consumed.append(path)
        if entries:
            content = insert_entries(content, header, entries)
            total += len(entries)
            print(f"{header}: collated {len(entries)} entr{'y' if len(entries) == 1 else 'ies'}")

    if not consumed:
        print("No changelog fragments to collate.")
        return

    next_changelog.write_text(content, encoding="utf-8")
    for path in consumed:
        path.unlink()
    print(f"Collated {total} entries into {NEXT_CHANGELOG} and removed {len(consumed)} fragments.")


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--check", action="store_true", help="validate fragment placement without writing")
    parser.add_argument("--root", type=pathlib.Path, default=pathlib.Path.cwd(), help="repository root")
    args = parser.parse_args(argv)

    if args.check:
        sys.exit(check(args.root))
    collate(args.root)


if __name__ == "__main__":
    main()
