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
reference is appended at the very end of an entry, so an entry wrapped over
several lines gets it on the last line.
"""

import argparse
import pathlib
import re

# A `* `/`- ` line starts a new entry; see the module docstring.
ENTRY_MARKER_RE = re.compile(r"^\s*[*-] ")

# A reference already at the *end* of an entry, either the raw `(#1234)` this
# tool writes or the `([#1234](…/pull/1234))` link update_github_links.py
# expands it to. Only a trailing reference makes us leave the entry alone; a
# `#1234` earlier in the body (e.g. "Fixes #6030: …") does not, so the PR is
# still appended at the end. Matching both forms is also what makes a re-run a
# no-op, so the workflow's own push cannot retrigger itself into a loop. A pull
# *link* is required rather than any `([#…](…))`, so an issue link at the end
# doesn't block the PR reference; a trailing period is tolerated.
END_REF_RE = re.compile(r"(?:\(#\d+\)|\(\[#\d+\]\([^)]*/pull/\d+\)\))\s*\.?\s*$")


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
    """Append ``(#pr)`` to the very end of one line.

    >>> append_reference("Added the `databricks quickstart` command.", 1234)
    'Added the `databricks quickstart` command. (#1234)'
    >>> append_reference("* No trailing period", 1234)
    '* No trailing period (#1234)'
    """
    return f"{line.rstrip()} (#{pr})"


def annotate_text(text, pr):
    r"""Append ``(#pr)`` to every entry in a fragment that has no reference.

    >>> annotate_text("Added the `databricks quickstart` command.\n", 1234)
    'Added the `databricks quickstart` command. (#1234)\n'

    Entries are annotated independently, and one that already ends with a PR
    reference is left untouched:

    >>> annotate_text("* one\n* two ([#99](https://github.com/databricks/cli/pull/99))\n", 1234)
    '* one (#1234)\n* two ([#99](https://github.com/databricks/cli/pull/99))\n'

    A reference to a prior PR or issue in the body does not count — the PR is
    still appended at the end:

    >>> annotate_text("Fixes [#6030](https://github.com/databricks/cli/issues/6030): a bug.\n", 1234)
    'Fixes [#6030](https://github.com/databricks/cli/issues/6030): a bug. (#1234)\n'

    A wrapped entry gets the reference at its end, not mid-sentence:

    >>> annotate_text("A long entry that wraps\nover two lines.\n", 1234)
    'A long entry that wraps\nover two lines. (#1234)\n'
    """
    lines = text.split("\n")
    for start, stop in entry_ranges(lines):
        content = [i for i in range(start, stop) if lines[i].strip()]
        if not content:
            continue
        last = lines[content[-1]]
        # Only a reference at the end of the entry blocks appending one.
        if END_REF_RE.search(last):
            continue
        lines[content[-1]] = append_reference(last, pr)
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
