#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""Add a PR reference to changelog fragments that lack one.

A PR reference in a `.nextchanges/` fragment is optional (see
`.nextchanges/README.md`) and easy to forget. The `nextchanges-pr-link` workflow
runs this over the fragments a PR *adds* and appends `(#<pr>)` to every entry
that has no reference yet. Expanding that into the canonical markdown link is
left to `tools/update_github_links.py`, which the workflow runs next.

A fragment holds one entry per `* `/`- ` line, or a single entry when it has no
such marker (see `render_nextchanges` in `internal/genkit/release_tagging.py`,
which bullets the first line and passes continuation lines through). The
reference goes at the end of an entry, so an entry wrapped over several lines
gets it on the last line rather than mid-sentence.
"""

import argparse
import pathlib
import re

# A `* `/`- ` line starts a new entry; see the module docstring.
ENTRY_MARKER_RE = re.compile(r"^\s*[*-] ")

# Any `#1234` counts as a reference, raw or already expanded into a link: the
# author pointed at a PR (or a related issue) themselves, so leave the entry
# alone. This is also what makes a re-run a no-op, so the workflow's own push
# cannot retrigger itself into a loop.
EXISTING_REF_RE = re.compile(r"#\d+")


def entry_ranges(lines):
    """Return one ``(start, stop)`` line-index range per entry.

    >>> entry_ranges(["* first", "* second"])
    [(0, 1), (1, 2)]
    >>> entry_ranges(["one entry", "wrapped over two lines"])
    [(0, 2)]
    """
    starts = [i for i, line in enumerate(lines) if ENTRY_MARKER_RE.match(line)]
    # No marker at all: the whole fragment is a single entry.
    if not starts:
        starts = [0]
    stops = [*starts[1:], len(lines)]
    return list(zip(starts, stops, strict=True))


def append_reference(line, pr):
    """Append ``(#pr)`` to one line, before its trailing period.

    Existing entries in CHANGELOG.md put the link inside the sentence rather
    than after it, e.g. ``… on every run ([#6060](…)).``

    >>> append_reference("Added the `databricks quickstart` command.", 1234)
    'Added the `databricks quickstart` command (#1234).'
    >>> append_reference("* No trailing period", 1234)
    '* No trailing period (#1234)'
    """
    body = line.rstrip()
    if body.endswith("."):
        return f"{body[:-1]} (#{pr})."
    return f"{body} (#{pr})"


def annotate_text(text, pr):
    r"""Append ``(#pr)`` to every entry in a fragment that has no reference.

    >>> annotate_text("Added the `databricks quickstart` command.\n", 1234)
    'Added the `databricks quickstart` command (#1234).\n'

    Entries are annotated independently, and one that already references a PR
    is left untouched:

    >>> annotate_text("* one\n* two ([#99](https://github.com/databricks/cli/pull/99))\n", 1234)
    '* one (#1234)\n* two ([#99](https://github.com/databricks/cli/pull/99))\n'

    A wrapped entry gets the reference at its end, not mid-sentence:

    >>> annotate_text("A long entry that wraps\nover two lines.\n", 1234)
    'A long entry that wraps\nover two lines (#1234).\n'
    """
    lines = text.split("\n")
    for start, stop in entry_ranges(lines):
        entry = lines[start:stop]
        if EXISTING_REF_RE.search("\n".join(entry)):
            continue
        content = [i for i in range(start, stop) if lines[i].strip()]
        if not content:
            continue
        lines[content[-1]] = append_reference(lines[content[-1]], pr)
    return "\n".join(lines)


def process_file(path, pr):
    """Process a single fragment.

    Returns True if the file was *modified*.
    """
    original = path.read_text(encoding="utf-8")
    updated = annotate_text(original, pr)
    if updated != original:
        path.write_text(updated, encoding="utf-8")
        print(f"Updated {path}")
        return True

    return False


def main(argv=None):
    parser = argparse.ArgumentParser(description="Add a PR reference to changelog fragments that lack one.")
    parser.add_argument("--pr", type=int, required=True, help="pull request number to reference")
    parser.add_argument("files", nargs="+", help="fragment files to annotate")
    args = parser.parse_args(argv)

    for file_path in args.files:
        process_file(pathlib.Path(file_path), args.pr)


if __name__ == "__main__":
    main()
