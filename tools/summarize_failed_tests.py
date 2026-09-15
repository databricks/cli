#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""
Summarize failed tests from a gotestsum --jsonfile output for a GitHub job summary.

`task test` prints a lot of output that GitHub is slow to render, so this surfaces
just the failed test names as a Markdown table. It reads the concatenated
gotestsum --jsonfile output (go test -json events, one JSON object per line) and
prints Markdown to stdout; the workflow tees that into $GITHUB_STEP_SUMMARY.
"""

import argparse
import json

# go test -json result actions. Only these mark the terminal state of a test or
# package; intermediate actions like "run" and "output" are ignored.
RESULT_ACTIONS = ("pass", "fail", "skip")


def load_events(path):
    """Yield the result events (parsed JSON objects) from a gotestsum jsonfile."""
    with open(path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            event = json.loads(line)
            if event.get("Action") in RESULT_ACTIONS:
                yield event


def last_action_by_key(events, key):
    """Map each key to the Action of its last event.

    >>> events = [
    ...     {"Action": "run", "id": 1},
    ...     {"Action": "pass", "id": 1},
    ...     {"Action": "fail", "id": 2},
    ... ]
    >>> last_action_by_key(events, lambda e: e["id"])
    {1: 'pass', 2: 'fail'}

    Empty events returns empty dict:
    >>> last_action_by_key([], lambda e: e["id"])
    {}

    Later occurrences override earlier ones:
    >>> events = [
    ...     {"Action": "pass", "test": "A"},
    ...     {"Action": "fail", "test": "A"},
    ... ]
    >>> last_action_by_key(events, lambda e: e["test"])
    {'A': 'fail'}
    """
    last = {}
    for event in events:
        last[key(event)] = event["Action"]
    return last


def failed_tests(events):
    """Return sorted (package, test) pairs whose last result was a failure.

    A test can appear more than once because of --rerun-fails, so we group by
    package+test and keep only those whose last result was a failure (recovered
    flakes end on "pass").

    >>> events = [
    ...     {"Package": "pkg1", "Test": "TestA", "Action": "fail"},
    ...     {"Package": "pkg1", "Test": "TestB", "Action": "pass"},
    ... ]
    >>> failed_tests(events)
    [('pkg1', 'TestA')]

    Flakes that recover are not reported:
    >>> events = [
    ...     {"Package": "pkg1", "Test": "TestFlake", "Action": "fail"},
    ...     {"Package": "pkg1", "Test": "TestFlake", "Action": "pass"},
    ... ]
    >>> failed_tests(events)
    []

    Multiple failures sorted by package, then test:
    >>> events = [
    ...     {"Package": "pkg2", "Test": "B", "Action": "fail"},
    ...     {"Package": "pkg1", "Test": "A", "Action": "fail"},
    ... ]
    >>> failed_tests(events)
    [('pkg1', 'A'), ('pkg2', 'B')]

    Package-level events (Test=None) are ignored:
    >>> events = [
    ...     {"Package": "pkg1", "Test": None, "Action": "fail"},
    ...     {"Package": "pkg1", "Test": "TestX", "Action": "fail"},
    ... ]
    >>> failed_tests(events)
    [('pkg1', 'TestX')]
    """
    test_events = [e for e in events if e.get("Test") is not None]
    last = last_action_by_key(test_events, lambda e: (e["Package"], e["Test"]))
    return sorted(key for key, action in last.items() if action == "fail")


def failed_packages_without_test(events, failed_test_packages):
    """Return sorted packages that failed with no failing test (build error / panic).

    A build error or panic surfaces as a package-level "fail" event (Test == null)
    with no failing test in the package. go test -json also emits a package-level
    "fail" for every package that merely has a failing test, so exclude packages
    already reported via failed_tests; otherwise those get double-reported here as
    spurious build errors.

    >>> events = [
    ...     {"Package": "pkg1", "Test": None, "Action": "fail"},
    ...     {"Package": "pkg2", "Test": None, "Action": "fail"},
    ... ]
    >>> failed_packages_without_test(events, set())
    ['pkg1', 'pkg2']

    Packages with failing tests are excluded:
    >>> events = [
    ...     {"Package": "pkg1", "Test": None, "Action": "fail"},
    ... ]
    >>> failed_packages_without_test(events, {"pkg1"})
    []

    Passed packages are ignored:
    >>> events = [
    ...     {"Package": "pkg1", "Test": None, "Action": "pass"},
    ...     {"Package": "pkg2", "Test": None, "Action": "fail"},
    ... ]
    >>> failed_packages_without_test(events, set())
    ['pkg2']

    Empty events returns empty:
    >>> failed_packages_without_test([], set())
    []
    """
    package_events = [e for e in events if e.get("Test") is None]
    last = last_action_by_key(package_events, lambda e: e["Package"])
    return sorted(
        package for package, action in last.items() if action == "fail" and package not in failed_test_packages
    )


def render(tests, packages):
    """Render failed tests and build-error packages as markdown.

    >>> print(render([("pkg1", "TestA")], []))
    ## Failed tests
    <BLANKLINE>
    | Package | Test |
    | --- | --- |
    | pkg1 | `TestA` |

    Build errors without tests:
    >>> print(render([], ["pkg1", "pkg2"]))
    ## Failed tests
    <BLANKLINE>
    ### Packages that failed without a test (build error / panic)
    <BLANKLINE>
    ```
    pkg1
    pkg2
    ```

    Both tests and build errors:
    >>> print(render([("pkg1", "T1")], ["pkg2"]))
    ## Failed tests
    <BLANKLINE>
    | Package | Test |
    | --- | --- |
    | pkg1 | `T1` |
    <BLANKLINE>
    ### Packages that failed without a test (build error / panic)
    <BLANKLINE>
    ```
    pkg2
    ```

    No failures:
    >>> print(render([], []))
    ## Failed tests
    <BLANKLINE>
    No failed tests found in test-output.json (the failure may be outside the test run).
    """
    lines = ["## Failed tests"]
    if tests:
        lines += ["", "| Package | Test |", "| --- | --- |"]
        lines += [f"| {package} | `{test}` |" for package, test in tests]
    if packages:
        lines += ["", "### Packages that failed without a test (build error / panic)", "", "```"]
        lines += packages
        lines += ["```"]
    if not tests and not packages:
        lines += ["", "No failed tests found in test-output.json (the failure may be outside the test run)."]
    return "\n".join(lines)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("jsonfile", help="Path to the gotestsum --jsonfile output")
    args = parser.parse_args()

    try:
        events = list(load_events(args.jsonfile))
    except FileNotFoundError:
        print(f"{args.jsonfile} not found; skipping failed-test summary")
        return

    tests = failed_tests(events)
    packages = failed_packages_without_test(events, {package for package, _ in tests})
    print(render(tests, packages))


if __name__ == "__main__":
    main()
