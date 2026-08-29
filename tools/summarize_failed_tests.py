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
    """Map each key to the Action of its last event."""
    last = {}
    for event in events:
        last[key(event)] = event["Action"]
    return last


def failed_tests(events):
    """Return sorted (package, test) pairs whose last result was a failure.

    A test can appear more than once because of --rerun-fails, so we group by
    package+test and keep only those whose last result was a failure (recovered
    flakes end on "pass").
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
    """
    package_events = [e for e in events if e.get("Test") is None]
    last = last_action_by_key(package_events, lambda e: e["Package"])
    return sorted(
        package for package, action in last.items() if action == "fail" and package not in failed_test_packages
    )


def render(tests, packages):
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
