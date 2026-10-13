#!/usr/bin/env python3
"""Keep CLI maintainers eligible to approve every CODEOWNERS rule."""

import sys
from pathlib import Path

MAINTAINERS = "@databricks/eng-deco-cli"


def validate_contents(data):
    r"""Return line-numbered errors for missing or misplaced maintainers.

    A maintainer catch-all followed by area owners is valid:

    >>> catch_all = '* @databricks/eng-deco-cli\n'
    >>> validate_contents(catch_all + '/bundle/ @databricks/eng-deco-cli @databricks/eng-deco-dabs\n')
    []

    Ignore blank lines and comments, including inline comments:

    >>> validate_contents('\n  # Owners\n\t*\t@databricks/eng-deco-cli # Default\r\n')
    []

    Missing owners, a different first owner, and a maintainer listed later fail:

    >>> validate_contents(catch_all + '/bundle/\n')
    ['2: first owner must be @databricks/eng-deco-cli']
    >>> validate_contents(catch_all + '/bundle/ @databricks/eng-deco-dabs\n')
    ['2: first owner must be @databricks/eng-deco-cli']
    >>> validate_contents(catch_all + '/bundle/ @databricks/eng-deco-dabs @databricks/eng-deco-cli\n')
    ['2: first owner must be @databricks/eng-deco-cli']

    A comment or similarly named team cannot satisfy the owner requirement:

    >>> validate_contents(catch_all + '/bundle/ # @databricks/eng-deco-cli\n')
    ['2: first owner must be @databricks/eng-deco-cli']
    >>> validate_contents('* @databricks/eng-deco-cli-extra\n')
    ['1: first owner must be @databricks/eng-deco-cli']

    The first rule must cover all paths, even if a catch-all appears later:

    >>> validate_contents('# Owners\n/bundle/ @databricks/eng-deco-cli\n' + catch_all)
    ["2: first rule must use '*' to cover all paths"]
    >>> validate_contents('')
    ["1: missing '* @databricks/eng-deco-cli' catch-all rule"]
    >>> validate_contents('\n# No rules\n')
    ["1: missing '* @databricks/eng-deco-cli' catch-all rule"]

    Report every invalid rule with its actual line number:

    >>> validate_contents(catch_all + '\n# Bundles\n/bundle/\n/cmd/bundle/ @databricks/eng-deco-dabs\n')
    ['4: first owner must be @databricks/eng-deco-cli', '5: first owner must be @databricks/eng-deco-cli']
    """
    errors = []
    found_rule = False
    for lineno, line in enumerate(data.splitlines(), 1):
        fields = line.split()
        if not fields or fields[0].startswith("#"):
            continue
        if not found_rule and fields[0] != "*":
            errors.append(f"{lineno}: first rule must use '*' to cover all paths")
        found_rule = True
        if fields[1:2] != [MAINTAINERS]:
            errors.append(f"{lineno}: first owner must be {MAINTAINERS}")
    if not found_rule:
        errors.append(f"1: missing '* {MAINTAINERS}' catch-all rule")
    return errors


def main():
    path = Path(".github/CODEOWNERS")
    errors = validate_contents(path.read_text(encoding="utf-8"))
    for error in errors:
        print(f"{path}:{error}")
    return 1 if errors else 0


if __name__ == "__main__":
    sys.exit(main())
