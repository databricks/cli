#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""
Analyze downloaded GH logs and print a report. Use gh_report.py instead of this script directly.
"""

import sys
import json
import argparse
from collections import Counter
from pathlib import Path

# Total number of environments expected
TOTAL_ENVS = 10

# \u200b is zero-width space. It is added so that len of the string corresponds to real width.
# ❌, ✅, 🔄 each take space of 2 characters.
FLAKY = "🔄\u200bflaky"
FAIL = "❌\u200bFAIL"
PASS = "✅\u200bpass"
SKIP = "🙈\u200bskip"

# FAIL is replaced with BUG when test fails in all environments (and when we have >=TOTAL_ENVS-1 environments)
# This indicate that it's very likely that PR did broke this test rather than environment being flaky.
BUG = "🪲\u200bBUG"

# This happens when Eventually is used - there is output for the test but no result.
MISSING = "🤯\u200bMISS"
PANIC = "💥\u200bPANIC"

# These happen if test matches known_failures.txt
KNOWN_FAILURE = "🟨\u200bKNOWN"
RECOVERED = "💚\u200bRECOVERED"

# The order is important - in case of ambiguity, earlier one gets preference.
# For examples, each environment gets a summary icon which is earliest action in this list among all tests.
INTERESTING_ACTIONS = (PANIC, BUG, FAIL, KNOWN_FAILURE, MISSING, FLAKY, RECOVERED)
ACTIONS_WITH_ICON = INTERESTING_ACTIONS + (PASS, SKIP)

ACTION_MESSAGES = {
    "fail": FAIL,
    "pass": PASS,
    "skip": SKIP,
}


class KnownFailuresConfig:
    def __init__(self, rules):
        self.rules = rules

    def matches(self, package_name, test_name):
        for rule in self.rules:
            if rule.matches(package_name, test_name):
                return rule.original_line
        return ""


class KnownFailuresRule:
    def __init__(self, package_pattern, test_pattern, package_prefix, test_prefix, original_line):
        self.package_pattern = package_pattern
        self.test_pattern = test_pattern
        self.package_prefix = package_prefix
        self.test_prefix = test_prefix
        self.original_line = original_line

    def matches(self, package_name, test_name):
        # Check package pattern
        if self.package_prefix:
            package_match = self._matches_path_prefix(package_name, self.package_pattern)
        else:
            package_match = package_name == self.package_pattern

        if not package_match:
            return False

        # Check test pattern - this matches the Go logic
        if self.test_prefix:
            return self._matches_path_prefix(test_name, self.test_pattern) or self._matches_path_prefix(
                self.test_pattern, test_name
            )
        else:
            return test_name == self.test_pattern or self._matches_path_prefix(self.test_pattern, test_name)

    def _matches_path_prefix(self, s, pattern):
        if pattern == "":
            return True
        if s == pattern:
            return True
        return s.startswith(pattern + "/")


def parse_known_failures(content):
    """
    Parse known failures config content.

    >>> _test_parse_known_failures()
    """
    rules = []
    for line_num, line in enumerate(content.splitlines(), 1):
        line = line.strip()
        if not line or line.startswith("#"):
            continue

        # Remove comments
        if "#" in line:
            line = line[: line.index("#")].strip()
            if not line:
                continue

        parts = line.split()
        if len(parts) != 2:
            continue

        package_pattern, test_pattern = parts
        package_pattern, package_prefix = _parse_pattern(package_pattern)
        test_pattern, test_prefix = _parse_pattern(test_pattern)

        rule = KnownFailuresRule(package_pattern, test_pattern, package_prefix, test_prefix, line)
        rules.append(rule)

    return KnownFailuresConfig(rules)


def _parse_pattern(pattern):
    if pattern == "*":
        return "", True
    if pattern.endswith("/"):
        return pattern[:-1], True
    return pattern, False


def _test_parse_known_failures():
    """Test cases from Go testrunner/main_test.go as table tests."""
    # Table of test cases: (input, package_name, testcase, expected_match)
    test_cases = [
        # Exact matches
        ("bundle TestDeploy", "bundle", "TestDeploy", True),
        ("bundle TestDeploy", "libs", "TestDeploy", False),
        ("bundle TestDeploy", "bundle", "TestSomethingElse", False),
        # Package prefix matches
        ("libs/ TestSomething", "libs/auth", "TestSomething", True),
        ("libs/ TestSomething", "libs", "TestSomething", True),
        ("libs/ TestSomething", "libsother", "TestSomething", False),
        # Test prefix matches
        ("bundle TestAccept/", "bundle", "TestAcceptDeploy", False),
        ("bundle TestAccept/", "bundle", "TestAccept", True),
        ("bundle TestAccept/", "bundle", "TestAccept/Deploy", True),
        # Wildcard matches
        ("* *", "any/package", "AnyTest", True),
        ("* TestAccept/", "any/package", "TestAcceptDeploy", False),
        ("* TestAccept/", "any/package", "TestAccept/Deploy", True),
        ("libs/ *", "libs/auth", "AnyTest", True),
        # Path prefix edge cases
        ("TestAccept/ TestAccept/", "TestAccept", "TestAccept", True),
        ("TestAccept/ TestAccept/", "TestAccept/bundle", "TestAccept/deploy", True),
        ("TestAccept/ TestAccept/", "TestAcceptSomething", "TestAcceptSomething", False),
        # Empty values cases
        ("* TestDeploy", "", "TestDeploy", True),
        ("bundle *", "bundle", "", True),
        # Subtest failure results in parent test failure as well
        (
            "acceptance TestAccept/bundle/templates/default-python/combinations/classic",
            "acceptance",
            "TestAccept",
            True,
        ),
        (
            "acceptance TestAccept/bundle/templates/default-python/combinations/classic",
            "acceptance",
            "TestAnother",
            False,
        ),
        (
            "acceptance TestAccept/bundle/templates/default-python/combinations/classic",
            "acceptance",
            "TestAccept/bundle/templates/default-python/combinations/classic/x",
            False,
        ),
        # Pattern version
        (
            "acceptance TestAccept/bundle/templates/default-python/combinations/classic/",
            "acceptance",
            "TestAccept",
            True,
        ),
        (
            "acceptance TestAccept/bundle/templates/default-python/combinations/classic/",
            "acceptance",
            "TestAnother",
            False,
        ),
        (
            "acceptance TestAccept/bundle/templates/default-python/combinations/classic/",
            "acceptance",
            "TestAccept/bundle/templates/default-python/combinations/classic/x",
            True,
        ),
        (
            "acceptance TestAccept/bundle/templates/default-python/combinations/classic/",
            "acceptance",
            "TestAccept/bundle/templates/default-python/combinations/classic",
            True,
        ),
        (
            "acceptance TestAccept/bundle/templates/default-python/combinations/classic/",
            "acceptance",
            "TestAccept/bundle/templates/default-python/combinations",
            True,
        ),
    ]

    for input_str, package_name, testcase, expected_match in test_cases:
        config = parse_known_failures(input_str)
        result = config.matches(package_name, testcase)

        # Convert result to boolean for comparison
        actual_match = bool(result)

        if actual_match != expected_match:
            raise AssertionError(
                f"Test failed for input='{input_str}', package='{package_name}', test='{testcase}': "
                f"expected {expected_match}, got {actual_match} (result: '{result}')"
            )


def load_known_failures():
    try:
        known_failures_path = Path(".gh-logs/known_failures.txt")
        if known_failures_path.exists():
            content = known_failures_path.read_text()
            return parse_known_failures(content)
    except Exception:
        pass
    return None


def cleanup_env(name):
    """
    >>> cleanup_env("test-output-aws-prod-is-linux-ubuntu-latest")
    'aws linux'

    >>> cleanup_env("test-output-gcp-prod-is-windows-server-latest")
    'gcp windows'

    >>> cleanup_env("test-output-azure-prod-ucws-is-linux-ubuntu-latest")
    'azure-ucws linux'
    """
    if not name.startswith("test-output-"):
        return ""
    name = name.removeprefix("test-output-")
    name = name.replace("-prod-ucws-is-", "-ucws-")
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
            print(f"{path}: {ex}\n{line!r}\n")
            break
        testname = data.get("Test")
        if not testname:
            continue
        if filter and filter not in testname:
            continue

        package_name = data.get("Package", "").removeprefix("github.com/databricks/cli/")
        test_key = (package_name, testname)

        action = data.get("Action")
        action = ACTION_MESSAGES.get(action, action)

        if action in (FAIL, PASS, SKIP):
            prev = results.get(test_key)
            if prev == FAIL and action == PASS:
                results[test_key] = FLAKY
            else:
                results[test_key] = action

        out = data.get("Output")
        if out:
            outputs.setdefault(test_key, []).append(out.rstrip())

    for test_key, lines in outputs.items():
        if test_key in results:
            continue
        if "panic: " in str(lines):
            results.setdefault(test_key, PANIC)
        else:
            results.setdefault(test_key, MISSING)

    return results, outputs


def mark_known_failures(results, known_failures_config):
    """Mark tests as KNOWN_FAILURE or RECOVERED based on known failures config."""
    marked_results = {}
    for test_key, action in results.items():
        package_name, testname = test_key
        if known_failures_config and action == FAIL and known_failures_config.matches(package_name, testname):
            marked_results[test_key] = KNOWN_FAILURE
        elif known_failures_config and action == PASS and known_failures_config.matches(package_name, testname):
            marked_results[test_key] = RECOVERED
        else:
            marked_results[test_key] = action
    return marked_results


def print_report(filenames, filter, filter_env, show_output, markdown=False, omit_repl=False):
    known_failures_config = load_known_failures()
    outputs = {}  # test_key -> env -> [output]
    per_test_per_env_stats = {}  # test_key -> env -> action -> count
    all_test_keys = set()
    all_envs = set()
    count_files = 0
    count_results = 0
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
        test_results = mark_known_failures(test_results, known_failures_config)
        count_files += 1
        count_results += len(test_results)
        for test_key, action in test_results.items():
            per_test_per_env_stats.setdefault(test_key, {}).setdefault(env, Counter())[action] += 1
        for test_key, output in test_outputs.items():
            outputs.setdefault(test_key, {}).setdefault(env, []).extend(output)
        all_test_keys.update(test_results)

    print(f"Parsed {count_files} files: {count_results} results", file=sys.stderr, flush=True)

    # Check for missing tests
    for test_key in all_test_keys:
        package_name, testname = test_key
        # It is possible for test to be missing if it's parent is skipped, ignore test cases with a parent.
        # For acceptance tests, ignore tests with subtests produced via EnvMatrix
        if testname.startswith("TestAccept/") and "=" in testname:
            continue
        # For non-acceptance tests ignore all subtests.
        if not testname.startswith("TestAccept/") and "/" in testname:
            continue
        test_results = per_test_per_env_stats.get(test_key, {})
        for e in all_envs:
            if e not in test_results:
                test_results.setdefault(e, Counter())[MISSING] += 1

    # Check if we can convert FAIL to BUG
    def is_bug(test_results):
        if len(test_results) < TOTAL_ENVS - 1:
            # incomplete results
            return False
        count = 0
        for e, env_results in test_results.items():
            if PASS in env_results:
                return False
            if FLAKY in env_results:
                return False
            if SKIP in env_results:
                count -= 1
            else:
                count += 1
        return count >= 0

    for test_key in all_test_keys:
        test_results = per_test_per_env_stats.get(test_key, {})
        if not is_bug(test_results):
            continue
        for e, env_results in sorted(test_results.items()):
            if env_results[FAIL] > 0:
                env_results[FAIL] -= 1
                if not env_results[FAIL]:
                    env_results.pop(FAIL)
                env_results[BUG] += 1

    per_env_stats = {}  # env -> action -> count
    for test_key, items in per_test_per_env_stats.items():
        for env, stats in items.items():
            per_env_stats.setdefault(env, Counter()).update(stats)

    table = []
    columns = {" ", "Env"}
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
        columns.update(stats)

    def key(column):
        try:
            return (ACTIONS_WITH_ICON.index(column), "")
        except:
            return (-1, str(column))

    columns = sorted(columns, key=key)
    print(format_table(table, markdown=markdown, columns=columns))

    interesting_envs = set()
    for env, stats in per_env_stats.items():
        for act in INTERESTING_ACTIONS:
            if act in stats:
                interesting_envs.add(env)
                break

    simplified_results = {}  # test_key -> env -> action
    for test_key, items in sorted(per_test_per_env_stats.items()):
        package_name, testname = test_key
        per_testkey_result = simplified_results.setdefault(test_key, {})
        # first select tests with interesting actions (anything but pass or skip)
        for env, counts in items.items():
            for action in INTERESTING_ACTIONS:
                if action in counts:
                    per_testkey_result.setdefault(env, short_action(action))
                    break

        # Once we know test is interesting, complete the row
        if per_testkey_result:
            for env, counts in items.items():
                if env not in interesting_envs:
                    continue
                for action in (PASS, SKIP):
                    if action in counts:
                        per_testkey_result.setdefault(env, short_action(action))
                        break

        if not per_testkey_result:
            per_testkey_result = simplified_results.pop(test_key)

    table = []
    for test_key, items in simplified_results.items():
        package_name, testname = test_key
        table.append(
            {
                "Test Name": testname,
                **items,
            }
        )
    table_txt = format_table(table, markdown=markdown)
    if len(table) > 5:
        table_txt = wrap_in_details(table_txt, f"{len(table)} failing tests:")
    if table_txt:
        print(table_txt)

    if show_output:
        for test_key, stats in simplified_results.items():
            package_name, testname = test_key
            for env, action in stats.items():
                if action not in INTERESTING_ACTIONS:
                    continue
                output_lines = outputs.get(test_key, {}).get(env, [])
                if omit_repl:
                    output_lines = [
                        line
                        for line in output_lines
                        if not line.strip().startswith("REPL") and "Available replacements:" not in line
                    ]
                out = "\n".join(output_lines)

                if markdown:
                    print(f"### {env} {testname} {action}\n```\n{out}\n```")
                else:
                    print(f"### {env} {testname} {action}\n{out}")
                if out:
                    print()


# For test table, use shorter version of action.
# We have full action name in env table, so that is used as agenda.
def short_action(action):
    if len(action) >= 4 and action[1] == "\u200b":
        # include first non-emoji letter in case emoji rendering is broken
        return action[:3]
    return action


def format_table(table, columns=None, markdown=False):
    """
    Pretty-print a list-of-dicts as an aligned text table.

    Args:
        table (list[dict]): the data rows
        columns (list[str]): header names & column order
        markdown (bool): whether to output in markdown format
    """
    if not table:
        return []

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

    result = []
    write = result.append

    if markdown:
        # Header
        write("| " + " | ".join(autojust(str(col), w) for col, w in zip(columns, widths)) + " |")
        # Separator
        write("| " + " | ".join("-" * w for w in widths) + " |")
        # Data rows
        for row in table:
            write("| " + " | ".join(autojust(row.get(col, ""), w) for col, w in zip(columns, widths)) + " |")
    else:
        write(fmt(columns, widths))
        for ind, row in enumerate(table):
            write(fmt([row.get(col, "") for col in columns], widths))

    write("")

    return "\n".join(result)


def fmt(cells, widths):
    return "  ".join(autojust(cell, w) for cell, w in zip(cells, widths))


def autojust(value, width):
    # Note, this has no effect on how markdown is rendered, only relevant for terminal output
    value = str(value)
    if value.isdigit():
        return value.center(width)
    if len(value) <= 3:  # short action name
        return value.center(width)
    return value.ljust(width)


def wrap_in_details(txt, summary):
    return f"<details><summary>{summary}</summary>\n\n{txt}\n\n</details>"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("filenames", nargs="+", help="Filenames or directories to parse")
    parser.add_argument("--filter", help="Filter results by test name (substring match)")
    parser.add_argument("--filter-env", help="Filter results by env name (substring match)")
    parser.add_argument("--output", help="Show output for failed tests", action="store_true")
    parser.add_argument("--markdown", help="Output in GitHub-flavored markdown format", action="store_true")
    parser.add_argument(
        "--omit-repl",
        help="Omit lines starting with 'REPL' and containing 'Available replacements:'",
        action="store_true",
    )
    args = parser.parse_args()
    print_report(
        args.filenames,
        filter=args.filter,
        filter_env=args.filter_env,
        show_output=args.output,
        markdown=args.markdown,
        omit_repl=args.omit_repl,
    )


if __name__ == "__main__":
    main()
