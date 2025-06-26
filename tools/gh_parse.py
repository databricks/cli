#!/usr/bin/env python3
"""
Analyze downloaded GH logs and print a report. Use gh_report.py instead of this script directly.
"""

import sys
import json
import argparse
import re
from collections import Counter
from pathlib import Path


# \u200c is zero-width space. It is added so that len of the string corresponds to real width.
# ❌, ✅, 🔄 each take space of 2 characters.
FLAKY = "🔄\u200cflaky"
FAIL = "❌\u200cFAIL"
PASS = "✅\u200cpass"
SKIP = "🙈\u200cskip"

# This happens when Eventually is used - there is output for the test but no result.
MISSING = "🤯\u200cMISS"
PANIC = "💥\u200cPANIC"

INTERESTING_ACTIONS = (FAIL, FLAKY, PANIC, MISSING)
ACTIONS_WITH_ICON = INTERESTING_ACTIONS + (PASS, SKIP)

ACTION_MESSAGES = {
    "fail": FAIL,
    "pass": PASS,
    "skip": SKIP,
}


def cleanup_env(name):
    """
    >>> cleanup_env("test-output-aws-prod-is-linux-ubuntu-latest")
    'aws linux'

    >>> cleanup_env("test-output-gcp-prod-is-windows-server-latest")
    'gcp windows'

    >>> cleanup_env("test-output-azure-prod-ucws-is-linux-ubuntu-latest")
    'azure ucws linux'
    """
    if not name.startswith("test-output-"):
        return ""
    name = name.removeprefix("test-output-")
    name = name.replace("-prod-ucws-is-", " ucws-")
    name = name.replace("-prod-is-", "-")
    name = name.replace("-linux-ubuntu-latest", " linux")
    name = name.replace("-windows-server-latest", " windows")
    return name


def iter_path(filename):
    p = Path(filename)
    if p.is_file():
        yield filename
        return
    for dirpath, dirnames, filenames in p.walk():
        for f in filenames:
            yield dirpath / f


def iter_paths(paths):
    for path in paths:
        for filename in iter_path(path):
            yield filename


def parse_file(path, filter):
    results = {}
    outputs = {}
    for line in path.open():
        if not line.strip():
            continue
        try:
            data = json.loads(line)
        except Exception as ex:
            print(f"{filename}: {ex}\n{line!r}\n")
            break
        testname = data.get("Test")
        if not testname:
            continue
        if filter and filter not in testname:
            continue
        action = data.get("Action")

        action = ACTION_MESSAGES.get(action, action)

        if action in (FAIL, PASS, SKIP):
            prev = results.get(testname)
            if prev == FAIL and action == PASS:
                results[testname] = FLAKY
            else:
                results[testname] = action

        out = data.get("Output")
        if out:
            outputs.setdefault(testname, []).append(out.rstrip())

    for testname, lines in outputs.items():
        if testname in results:
            continue
        if "panic: " in str(lines):
            results.setdefault(testname, PANIC)
        else:
            results.setdefault(testname, MISS)

    return results, outputs


def print_report(filenames, filter, filter_env, show_output, markdown=False):
    outputs = {}  # testname -> env -> [output]
    per_test_per_env_stats = {}  # testname -> env -> action -> count
    all_testnames = set()
    all_envs = set()
    for filename in iter_paths(filenames):
        p = Path(filename)
        env = cleanup_env(p.parent.name)
        if not env:
            print(f"Ignoring {filename}: cannot extract env")
            continue
        if filter_env and filter_env not in env:
            continue
        all_envs.add(env)
        test_results, test_outputs = parse_file(p, filter)
        for testname, action in test_results.items():
            per_test_per_env_stats.setdefault(testname, {}).setdefault(env, Counter())[action] += 1
        for testname, output in test_outputs.items():
            outputs.setdefault(testname, {}).setdefault(env, []).extend(output)
        all_testnames.update(test_results)

    # Check for missing tests
    for testname in all_testnames:
        # It is possible for test to be missing if it's parent is skipped, ignore test cases with a parent.
        # For acceptance tests, ignore tests with subtests produced via EnvMatrix
        if testname.startswith("TestAccept/") and "=" in testname:
            continue
        # For non-acceptance tests ignore all subtests.
        if not testname.startswith("TestAccept/") and "/" in testname:
            continue
        test_results = per_test_per_env_stats.get(testname, {})
        for e in all_envs:
            if e not in test_results:
                test_results.setdefault(e, Counter())[MISSING] += 1

    per_env_stats = {}  # env -> action -> count
    for testname, items in per_test_per_env_stats.items():
        for env, stats in items.items():
            per_env_stats.setdefault(env, Counter()).update(stats)

    table = []
    for env, stats in sorted(per_env_stats.items()):
        status = "??"
        for action in ACTIONS_WITH_ICON:
            if action in stats:
                status = action[:2]
                break

        table.append(
            {
                " ": status,
                "Env": env,
                **stats,
            }
        )
    print_table(table, markdown=markdown)

    interesting_envs = set()
    for env, stats in per_env_stats.items():
        for act in INTERESTING_ACTIONS:
            if act in stats:
                interesting_envs.add(env)
                break

    simplified_results = {}  # testname -> env -> action
    for testname, items in sorted(per_test_per_env_stats.items()):
        per_testname_result = simplified_results.setdefault(testname, {})
        # first select tests with interesting actions (anything but pass or skip)
        for env, counts in items.items():
            for action in INTERESTING_ACTIONS:
                if action in counts:
                    per_testname_result.setdefault(env, action)
                    break

        # Once we know test is interesting, complete the row
        if per_testname_result:
            for env, counts in items.items():
                if env not in interesting_envs:
                    continue
                for action in (PASS, SKIP):
                    if action in counts:
                        per_testname_result.setdefault(env, action)
                        break

        if not per_testname_result:
            per_testname_result = simplified_results.pop(testname)

    table = []
    for testname, items in simplified_results.items():
        table.append(
            {
                "Test Name": testname,
                **items,
            }
        )
    print_table(table, markdown=markdown)

    if show_output:
        for testname, stats in simplified_results.items():
            for env, action in stats.items():
                if action not in INTERESTING_ACTIONS:
                    continue
                out = "\n".join(outputs.get(testname, {}).get(env, []))
                if markdown:
                    print(f"### {env} {testname} {action}\n```\n{out}\n```")
                else:
                    print(f"### {env} {testname} {action}\n{out}")
                if out:
                    print()


def print_table(table, columns=None, markdown=False):
    """
    Pretty-print a list-of-dicts as an aligned text table.

    Args:
        table (list[dict]): the data rows
        columns (list[str]): header names & column order
        markdown (bool): whether to output in markdown format
    """
    if not table:
        return

    if columns is None:
        columns = []
        seen = set()
        for row in table:
            for key in row:
                if key in seen:
                    continue
                seen.add(key)
                columns.append(key)
        columns.sort()

    widths = [len(col) for col in columns]
    for row in table:
        for i, col in enumerate(columns):
            widths[i] = max(widths[i], len(str(row.get(col, ""))))

    if markdown:
        # Header
        print("| " + " | ".join(str(col).ljust(w) for col, w in zip(columns, widths)) + " |")
        # Separator
        print("| " + " | ".join("-" * w for w in widths) + " |")
        # Data rows
        for row in table:
            print("| " + " | ".join(str(row.get(col, "")).ljust(w) for col, w in zip(columns, widths)) + " |")
    else:
        fmt = lambda cells: "  ".join(str(cell).ljust(w) for cell, w in zip(cells, widths))
        print(fmt(columns))
        for ind, row in enumerate(table):
            print(fmt([row.get(col, "") for col in columns]))

    print()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("filenames", nargs="+", help="Filenames or directories to parse")
    parser.add_argument("--filter", help="Filter results by test name (substring match)")
    parser.add_argument("--filter-env", help="Filter results by env name (substring match)")
    parser.add_argument("--output", help="Show output for failed tests", action="store_true")
    parser.add_argument("--markdown", help="Output in GitHub-flavored markdown format", action="store_true")
    args = parser.parse_args()
    print_report(args.filenames, filter=args.filter, filter_env=args.filter_env, show_output=args.output, markdown=args.markdown)


if __name__ == "__main__":
    main()
