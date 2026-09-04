#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
import glob
import os
import re
import subprocess
import sys


def load_ignores():
    ignores = set()
    fail = False
    for ind, line in enumerate(open(".wsignore")):
        line = line.strip()
        if not line:
            continue
        if line.startswith("#"):
            continue
        # include_hidden: without it `**` skips dot-prefixed names, so a directory
        # pattern silently fails to cover e.g. a vendored tree's .github/.
        expanded = glob.glob(line, recursive=True, include_hidden=True)
        if len(expanded) == 0:
            print(f".wsignore:{ind + 1}: No matches for line: {line}")
            fail = True
        ignores.update(expanded)
    if fail:
        sys.exit(1)
    return ignores


def count_trailing_newlines(s):
    """Count consecutive newlines at the end of a string.

    >>> count_trailing_newlines("hello")
    0
    >>> count_trailing_newlines("hello\\n")
    1
    >>> count_trailing_newlines("hello\\n\\n")
    2
    >>> count_trailing_newlines("\\n\\n\\n")
    3
    >>> count_trailing_newlines("")
    0
    """
    match = re.search(r"(\n+)$", s)
    return len(match.group(1)) if match else 0


def validate_contents(data):
    """Validate file contents and yield error messages for issues found.

    Returns empty for valid content (ends with single newline, no trailing spaces):
    >>> msgs = list(validate_contents(b'hello\\nworld\\n'))
    >>> len(msgs)
    0

    Detects missing final newline:
    >>> msgs = list(validate_contents(b'hello'))
    >>> ' File does not end with a newline' in msgs
    True

    Detects trailing whitespace:
    >>> msgs = list(validate_contents(b'hello  \\n'))
    >>> any('Trailing whitespace' in m for m in msgs)
    True

    Detects whitespace-only lines:
    >>> msgs = list(validate_contents(b'hello\\n  \\nworld\\n'))
    >>> any('Whitespace-only line' in m for m in msgs)
    True

    Detects multiple trailing newlines:
    >>> msgs = list(validate_contents(b'hello\\n\\n\\n'))
    >>> any('3 newlines at the end' in m for m in msgs)
    True

    Empty data yields nothing:
    >>> list(validate_contents(b''))
    []
    """
    if not data:
        return
    try:
        text = data.decode("utf")
    except Exception as ex:
        yield f" Failed to decode utf-8: {ex}"
        return

    for i, line in enumerate(text.split("\n")):
        if not line:
            continue
        if line.strip() == "":
            yield f"{i + 1}: Whitespace-only line"
            continue
        if line.rstrip() != line:
            yield f"{i + 1}: Trailing whitespace {line[-200:]!r}"

    newlines = count_trailing_newlines(text)

    if newlines == 0:
        yield " File does not end with a newline"

    if newlines >= 2:
        yield f" {newlines} newlines at the end"


def fix_contents(data):
    """Fix whitespace issues in file contents.

    Removes trailing whitespace and ensures exactly one final newline:
    >>> result = fix_contents(b'hello  \\nworld  \\n\\n\\n')
    >>> result == b'hello\\nworld\\n'
    True

    Adds missing final newline:
    >>> fix_contents(b'hello') == b'hello\\n'
    True

    Handles whitespace-only lines by removing trailing spaces:
    >>> result = fix_contents(b'hello\\n  \\nworld\\n')
    >>> result == b'hello\\n\\nworld\\n'
    True

    Returns empty input as-is:
    >>> fix_contents(b'') == b''
    True

    Preserves valid content unchanged:
    >>> fix_contents(b'hello\\nworld\\n') == b'hello\\nworld\\n'
    True
    """
    if not data:
        return data
    try:
        text = data.decode("utf")
    except Exception:
        # Can't decode, return as-is
        return data

    # Split into lines and fix each one
    lines = text.split("\n")
    fixed_lines = []
    for line in lines:
        # Remove trailing whitespace and whitespace-only lines
        fixed_lines.append(line.rstrip())

    # Join lines back together
    fixed_text = "\n".join(fixed_lines)

    # Ensure file ends with exactly one newline
    fixed_text = fixed_text.rstrip("\n") + "\n"

    return fixed_text.encode("utf")


def main():
    quiet = "-q" in sys.argv
    fix_mode = "--fix" in sys.argv
    files = subprocess.check_output(["git", "ls-files"], encoding="utf-8").split()
    ignores = load_ignores()
    n_checked = 0
    n_skipped = 0
    n_errored = 0
    n_fixed = 0
    for f in files:
        if not os.path.isfile(f):
            n_skipped += 1
            continue
        if f in ignores:
            n_skipped += 1
            continue
        with open(f, "rb") as file:
            data = file.read()

        if fix_mode:
            # Fix whitespace issues
            fixed_data = fix_contents(data)
            if fixed_data != data:
                with open(f, "wb") as file:
                    file.write(fixed_data)
                print(f"{f}: Fixed")
                n_fixed += 1
            n_checked += 1
        else:
            # Validate mode
            error = False
            for msg in validate_contents(data):
                print(f"{f}:{msg}")
                error = True
            n_checked += 1
            n_errored += 1 if error else 0

    if not quiet:
        if fix_mode:
            sys.stderr.write(f"{n_checked} checked, {n_skipped} skipped, {n_fixed} fixed.\n")
        else:
            sys.stderr.write(f"{n_checked} checked, {n_skipped} skipped, {n_errored} failed.\n")
    sys.exit(1 if n_errored else 0)


if __name__ == "__main__":
    try:
        main()
    except BrokenPipeError:
        sys.exit(1)
