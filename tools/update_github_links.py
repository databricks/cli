#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
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

# Canonical form: ([#1234](https://github.com/databricks/cli/pull/1234))
CONVERTED_LINK_RE = re.compile(
    r"\(\[#(?P<num_text>\d+)\]\("  # ([#1234](
    r"https://github\.com/databricks/cli/pull/(?P<num_url>\d+)"  # …/pull/1234
    r"\)\)"  # ))
)

# Double-paren form produced by a previous incorrect run:
#   (([#1234](https://github.com/databricks/cli/pull/1234)))
DOUBLE_PAREN_LINK_RE = re.compile(
    r"\(\(\[#(?P<num>\d+)\]\("
    r"https://github\.com/databricks/cli/pull/\d+"
    r"\)\)\)"
)

# Raw reference already wrapped in parens: (#1234)
PAREN_RAW_REF_RE = re.compile(r"\(#(?P<num>\d+)\)")

# Bare raw reference not already part of a converted link or paren-wrapped ref.
# Negative look-behinds: '[' means it's inside a converted link; '(' means
# it will be handled by PAREN_RAW_REF_RE above.
RAW_REF_RE = re.compile(r"(?<!\[)(?<!\()#(?P<num>\d+)\b")


def find_mismatched_links(text):
    """Return texts of mismatching converted links.

    >>> find_mismatched_links("([#1234](https://github.com/databricks/cli/pull/1234))")
    []
    >>> find_mismatched_links("([#1234](https://github.com/databricks/cli/pull/9999))")
    ['Converted link numbers differ: text #1234 vs URL #9999 — …([#1234](https://github.com/databricks/cli/pull/9999))…']
    """
    mismatches = []
    for m in CONVERTED_LINK_RE.finditer(text):
        num_text, num_url = m.group("num_text"), m.group("num_url")
        if num_text != num_url:
            context = text[max(0, m.start() - 20) : m.end() + 20]
            mismatches.append(f"Converted link numbers differ: text #{num_text} vs URL #{num_url} — …{context}…")
    return mismatches


def convert_raw_references(text):
    """Convert raw `#1234` references to markdown links.

    Already-converted single-paren links are left unchanged:

    >>> convert_raw_references("([#1234](https://github.com/databricks/cli/pull/1234))")
    '([#1234](https://github.com/databricks/cli/pull/1234))'

    Double-paren links from a previous incorrect run are collapsed to single-paren:

    >>> convert_raw_references("(([#1234](https://github.com/databricks/cli/pull/1234)))")
    '([#1234](https://github.com/databricks/cli/pull/1234))'

    A raw reference with surrounding parens becomes a single-paren link (not double):

    >>> convert_raw_references("(#3456)")
    '([#3456](https://github.com/databricks/cli/pull/3456))'

    A bare raw reference gets wrapped in a single-paren link:

    >>> convert_raw_references("#3456")
    '([#3456](https://github.com/databricks/cli/pull/3456))'

    Idempotent: running twice produces the same result:

    >>> t = "(#3456) and #7890"
    >>> convert_raw_references(convert_raw_references(t)) == convert_raw_references(t)
    True
    """

    def _make_link(num):
        return f"([#{num}](https://github.com/databricks/cli/pull/{num}))"

    # Fix existing double-paren links produced by a previous incorrect run.
    text = DOUBLE_PAREN_LINK_RE.sub(lambda m: _make_link(m.group("num")), text)

    # Convert (#1234) — parens already present, replace the whole token.
    text = PAREN_RAW_REF_RE.sub(lambda m: _make_link(m.group("num")), text)

    # Convert bare #1234 — not preceded by [ (converted) or ( (paren-wrapped).
    text = RAW_REF_RE.sub(lambda m: _make_link(m.group("num")), text)

    return text


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
