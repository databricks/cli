#!/usr/bin/env python3
"""Update PR references in changelog files.

1. Convert occurrences of `#1234` to the canonical markdown link
   `([#1234](https://github.com/databricks/cli/pull/1234))`.
2. Validate that for existing converted references the PR number in the text
   and in the URL match.
"""

import argparse
import pathlib
import re
import sys

DEFAULT_FILES = ("NEXT_CHANGELOG.md", "CHANGELOG.md")

# Regex that matches an *already converted* link, e.g.:
#   ([#1234](https://github.com/databricks/cli/pull/1234))
# The groups capture the PR number in the text and in the URL respectively so
# they can be compared for consistency.
CONVERTED_LINK_RE = re.compile(
    r"\(\[#(?P<num_text>\d+)\]\("  # ([#1234](
    r"https://github\.com/databricks/cli/pull/(?P<num_url>\d+)"  # …/pull/1234
    r"\)\)"  # ))
)

# Regex that matches a *raw* reference, `#1234`, that is **not** already inside
# a converted link.  The negative look-behind ensures the # is not preceded by
# a literal '[' which would indicate an already converted link.
RAW_REF_RE = re.compile(r"(?<!\[)#(?P<num>\d+)\b")


def find_mismatched_links(text):
    """Return texts of mismatching converted links."""
    mismatches = []
    for m in CONVERTED_LINK_RE.finditer(text):
        num_text, num_url = m.group("num_text"), m.group("num_url")
        if num_text != num_url:
            context = text[max(0, m.start() - 20) : m.end() + 20]
            mismatches.append(f"Converted link numbers differ: text #{num_text} vs URL #{num_url} — …{context}…")
    return mismatches


def convert_raw_references(text):
    """Convert raw `#1234` references to markdown links."""

    def _repl(match):
        num = match.group("num")
        return f"([#{num}](https://github.com/databricks/cli/pull/{num}))"

    return RAW_REF_RE.sub(_repl, text)


def process_file(path):
    """Process a single file.

    Returns True if the file was *modified*.
    Raises `SystemExit` with non-zero status on mismatching converted links.
    """
    original = path.read_text(encoding="utf-8")

    mismatches = find_mismatched_links(original)
    if mismatches:
        for msg in mismatches:
            print(f"{path}:{msg}", file=sys.stderr)
        sys.exit(1)

    updated = convert_raw_references(original)
    if updated != original:
        path.write_text(updated, encoding="utf-8")
        print(f"Updated {path}")
        return True

    return False


def main(argv=None):
    parser = argparse.ArgumentParser(description="Convert #PR references in changelogs to links.")
    parser.add_argument("files", nargs="*", help=f"Markdown files to process (default: {DEFAULT_FILES})")
    args = parser.parse_args(argv)

    modified_any = False
    for file_path in args.files or DEFAULT_FILES:
        file_path = pathlib.Path(file_path)
        modified_any |= process_file(file_path)


if __name__ == "__main__":
    main()
