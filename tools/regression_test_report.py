#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""
Regression test report generator for CLI acceptance and unit tests.

Finds tests modified or added on the current branch vs origin/main, runs them,
and classifies each into one of these categories:

  Possible regression test  — passes on branch, fails on main's code (or
                              cannot compile on main).
                              Exercises behavior introduced by this branch.

  Unreleased behavior       — passes on branch and main's code, but fails
  (acceptance only)           with the latest released CLI (-useversion latest).
                              Tests behavior already merged but not yet shipped;
                              no changelog entry needed for this PR.

  Additional coverage       — passes everywhere.

  Cannot compile            — test file cannot be compiled (branch or main).

For acceptance tests all three phases run. For unit tests only Phases 1 and 2
run (no -useversion support).

The worktree approach (Phase 2) keeps the working directory untouched: a
temporary git worktree is created from the base branch, the relevant test
files are copied into it, and go test runs there.

Run doctests: python3 -m doctest tools/regression_test_report.py

Usage:
    python3 tools/regression_test_report.py [--output PATH] [--max-tests N] [--base REF]
    python3 tools/regression_test_report.py --commit [--max-tests N] [--base REF]
"""

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from collections import Counter
from dataclasses import dataclass, field
from pathlib import Path

_git_root = subprocess.run(["git", "rev-parse", "--show-toplevel"], capture_output=True, text=True)
REPO_ROOT = Path(_git_root.stdout.strip()) if _git_root.returncode == 0 else Path(__file__).parent.parent
ACCEPTANCE_DIR = REPO_ROOT / "acceptance"
DEFAULT_MAX_TESTS = 20


# ---------------------------------------------------------------------------
# Pure functions — no filesystem, git, or subprocess access.
# These encode all classification and parsing logic and are testable with
# `python3 -m doctest tools/regression_test_report.py`.
# ---------------------------------------------------------------------------


def extract_test_functions(go_source):
    """Return Test* function names declared in Go source text.

    >>> extract_test_functions('func TestFoo(t *testing.T) {}')
    ['TestFoo']
    >>> extract_test_functions('func TestFoo(t *testing.T) {}\\nfunc TestBar(t *testing.T) {}')
    ['TestFoo', 'TestBar']
    >>> extract_test_functions('func helper() {}\\nfunc BenchmarkFoo(b *testing.B) {}')
    []
    >>> extract_test_functions('')
    []
    """
    return re.findall(r"^func (Test\w+)\(", go_source, re.MULTILINE)


def find_acceptance_test_for_path(rel_path, test_dirs):
    """Return the acceptance test dir (relative to acceptance/) for a changed file, or None.

    test_dirs is a collection of known test roots (relative to acceptance/).

    >>> find_acceptance_test_for_path('acceptance/bundle/foo/out.txt', {'bundle/foo', 'bundle/bar'})
    'bundle/foo'
    >>> find_acceptance_test_for_path('acceptance/bundle/foo/sub/x.txt', {'bundle/foo'})
    'bundle/foo'
    >>> find_acceptance_test_for_path('libs/foo/bar.go', {'bundle/foo'}) is None
    True
    >>> find_acceptance_test_for_path('acceptance/bundle/foo/out.txt', set()) is None
    True
    """
    p = Path(rel_path)
    if not p.parts or p.parts[0] != "acceptance":
        return None
    for depth in range(len(p.parts), 1, -1):
        candidate = str(Path(*p.parts[1:depth]))
        if candidate in test_dirs:
            return candidate
    return None


def is_compile_failure_json(json_text):
    """Return True if go test -json output indicates a build failure.

    >>> is_compile_failure_json('{"Action":"build-fail"}')
    True
    >>> is_compile_failure_json('{"Action":"fail","FailedBuild":"pkg/test"}')
    True
    >>> is_compile_failure_json('{"Action":"fail"}')
    False
    >>> is_compile_failure_json('')
    False
    """
    for line in json_text.splitlines():
        try:
            e = json.loads(line.strip())
        except (json.JSONDecodeError, ValueError):
            continue
        if e.get("Action") == "build-fail" or e.get("FailedBuild"):
            return True
    return False


def get_leaf_subtests(parent, test_names):
    """Return the leaf-level subtests for an acceptance test parent.

    If parent has EnvMatrix children in test_names, return them.
    Otherwise the parent itself is the leaf.

    >>> get_leaf_subtests('TestAccept/foo', ['TestAccept/foo', 'TestAccept/foo/A=1', 'TestAccept/foo/A=2'])
    ['TestAccept/foo/A=1', 'TestAccept/foo/A=2']
    >>> get_leaf_subtests('TestAccept/foo', ['TestAccept/foo'])
    ['TestAccept/foo']
    >>> get_leaf_subtests('TestAccept/foo', [])
    []
    """
    prefix = parent + "/"
    children = [t for t in test_names if t.startswith(prefix)]
    return children if children else ([parent] if parent in test_names else [])


def classify_acceptance_result(main_status, latest_status):
    """Classify one acceptance subtest based on its status on main and latest release.

    Returns 'regression', 'unreleased', or 'coverage'.
    pass and skip both count as success; anything else (fail, unknown, error) is a failure.

    >>> classify_acceptance_result('fail', 'fail')
    'regression'
    >>> classify_acceptance_result('fail', 'pass')
    'regression'
    >>> classify_acceptance_result('pass', 'fail')
    'unreleased'
    >>> classify_acceptance_result('skip', 'fail')
    'unreleased'
    >>> classify_acceptance_result('pass', 'pass')
    'coverage'
    >>> classify_acceptance_result('skip', 'skip')
    'coverage'
    """
    if main_status not in ("pass", "skip"):
        return "regression"
    if latest_status not in ("pass", "skip"):
        return "unreleased"
    return "coverage"


def classify_unit_result(compile_fail_on_main, function_results, passing_functions):
    """Classify a unit test package based on how it fares on main.

    Returns 'regression' or 'coverage'.
    compile_fail_on_main: True if the package could not compile on main.
    function_results: {func_name: status} for passing_functions after running on main.
    passing_functions: functions that passed on the current branch.

    >>> classify_unit_result(True, {}, ['TestFoo'])
    'regression'
    >>> classify_unit_result(False, {'TestFoo': 'fail'}, ['TestFoo'])
    'regression'
    >>> classify_unit_result(False, {'TestFoo': 'pass', 'TestBar': 'fail'}, ['TestFoo', 'TestBar'])
    'regression'
    >>> classify_unit_result(False, {'TestFoo': 'pass'}, ['TestFoo'])
    'coverage'
    >>> classify_unit_result(False, {'TestFoo': 'skip'}, ['TestFoo'])
    'coverage'
    >>> classify_unit_result(False, {}, [])
    'coverage'
    """
    if compile_fail_on_main:
        return "regression"
    if any(function_results.get(f) not in ("pass", "skip") for f in passing_functions):
        return "regression"
    return "coverage"


# ---------------------------------------------------------------------------
# Parsing go test -json output
# ---------------------------------------------------------------------------


@dataclass
class TestEntry:
    name: str
    status: str = ""  # pass | fail | skip | ""
    output_lines: list[str] = field(default_factory=list)

    @property
    def output(self):
        return "".join(self.output_lines)


def parse_json_output(json_text):
    """Parse go test -json output into {test_name: TestEntry}."""
    tests: dict[str, TestEntry] = {}
    for line in json_text.splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            entry = json.loads(line)
        except json.JSONDecodeError:
            continue
        name = entry.get("Test", "")
        if not name:
            continue
        if name not in tests:
            tests[name] = TestEntry(name=name)
        action = entry.get("Action", "")
        if action in ("pass", "fail", "skip"):
            tests[name].status = action
        elif action == "output":
            tests[name].output_lines.append(entry.get("Output", ""))
    return tests


def is_compile_failure(proc):
    """Return True if the CompletedProcess indicates a build failure."""
    return proc.returncode != 0 and is_compile_failure_json(proc.stdout)


def readable_output(json_text, max_lines=80):
    """Extract human-readable test output from a go test -json stream."""
    out_lines = []
    for line in json_text.splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            entry = json.loads(line)
        except json.JSONDecodeError:
            out_lines.append(line)
            continue
        action = entry.get("Action", "")
        if action in ("output", "build-output"):
            out_lines.append(entry.get("Output", ""))
    text = "".join(out_lines).replace("\r\n", "\n").replace("\r", "\n")
    lines = text.splitlines(keepends=True)
    if len(lines) > max_lines:
        lines = lines[:max_lines] + [f"\n... ({len(lines) - max_lines} more lines)\n"]
    return "".join(lines)


# ---------------------------------------------------------------------------
# Git helpers
# ---------------------------------------------------------------------------


def git(*args, **kwargs):
    kwargs.setdefault("capture_output", True)
    kwargs.setdefault("text", True)
    kwargs.setdefault("cwd", str(REPO_ROOT))
    return subprocess.run(["git"] + list(args), **kwargs)


def resolve_base_ref(hint=None):
    """Return the base ref for comparison, falling back from origin/main to main."""
    if hint:
        return hint
    if git("rev-parse", "--verify", "origin/main").returncode == 0:
        return "origin/main"
    return "main"


def get_changed_files(base_ref):
    """Return files changed between base_ref and HEAD."""
    result = git("diff", "--name-only", f"{base_ref}...HEAD")
    if result.returncode != 0:
        return []
    return [f for f in result.stdout.strip().split("\n") if f]


def file_exists_at_ref(path, ref):
    """Return True if path exists in the git tree at ref."""
    return git("show", f"{ref}:{path}").returncode == 0


# ---------------------------------------------------------------------------
# Acceptance test I/O
# ---------------------------------------------------------------------------


def scan_acceptance_test_dirs(acceptance_dir):
    """Return a frozenset of test dirs (relative to acceptance_dir) by finding 'script' files."""
    test_dirs = set()
    for dirpath, _dirnames, filenames in os.walk(acceptance_dir):
        if "script" in filenames:
            test_dirs.add(str(Path(dirpath).relative_to(acceptance_dir)))
    return frozenset(test_dirs)


def collect_changed_acceptance_tests(changed_files, base_ref):
    """Return (added, modified) lists of acceptance test paths."""
    test_dirs = scan_acceptance_test_dirs(ACCEPTANCE_DIR)
    test_paths = sorted(
        {tp for tp in (find_acceptance_test_for_path(f, test_dirs) for f in changed_files) if tp is not None}
    )
    added = [tp for tp in test_paths if not file_exists_at_ref(f"acceptance/{tp}/script", base_ref)]
    modified = [tp for tp in test_paths if file_exists_at_ref(f"acceptance/{tp}/script", base_ref)]
    return added, modified


def run_go_test(pattern, cwd, extra_args=None):
    """Run go test ./acceptance -run PATTERN -json and return CompletedProcess."""
    cmd = [
        "go",
        "test",
        "./acceptance",
        f"-run={pattern}",
        "-json",
        "-count=1",
        "-timeout=300s",
    ] + (extra_args or [])
    return subprocess.run(cmd, capture_output=True, text=True, cwd=str(cwd))


def run_acceptance_on_main(test_path, subtest, base_ref):
    """Run acceptance subtest using main's codebase with the current test directory."""
    with tempfile.TemporaryDirectory(prefix="regression_main_") as tmpdir:
        worktree_dir = Path(tmpdir) / "worktree"
        r = git("worktree", "add", "--detach", str(worktree_dir), base_ref)
        if r.returncode != 0:
            print(f"    [error] worktree creation failed: {r.stderr.strip()}", file=sys.stderr)
            return None
        try:
            src = ACCEPTANCE_DIR / test_path
            dst = worktree_dir / "acceptance" / test_path
            dst.parent.mkdir(parents=True, exist_ok=True)
            if dst.exists():
                shutil.rmtree(str(dst))
            shutil.copytree(str(src), str(dst))
            return run_go_test(subtest, worktree_dir)
        finally:
            git("worktree", "remove", "--force", str(worktree_dir))


def run_acceptance_with_latest_release(subtest, cwd):
    """Run acceptance subtest against the latest released CLI binary."""
    return run_go_test(subtest, cwd, extra_args=["-useversion", "latest"])


def read_latest_release_version():
    """Return the cached latest release version written by resolveLatestVersion, or None."""
    r = subprocess.run(["go", "env", "GOOS", "GOARCH"], capture_output=True, text=True, cwd=str(REPO_ROOT))
    if r.returncode != 0:
        return None
    goos, goarch = r.stdout.strip().split("\n", 1)
    cache = ACCEPTANCE_DIR / "build" / f"{goos}_{goarch}" / "latest_version.txt"
    if cache.exists():
        return cache.read_text().strip() or None
    return None


# ---------------------------------------------------------------------------
# Unit test I/O
# ---------------------------------------------------------------------------


def read_test_functions(file_path):
    """Read a Go test file and return its Test* function names."""
    try:
        return extract_test_functions(Path(file_path).read_text())
    except OSError:
        return []


def collect_changed_unit_tests(changed_files, base_ref):
    """Return (added, modified) lists of (package_dir, [changed_files], [functions])."""
    pkg_files: dict[str, list[str]] = {}
    for f in changed_files:
        p = Path(f)
        if not f.endswith("_test.go") or p.parts[0] == "acceptance":
            continue
        pkg_files.setdefault(str(p.parent), []).append(f)

    added, modified = [], []
    for pkg_dir in sorted(pkg_files):
        files = pkg_files[pkg_dir]
        functions = list(dict.fromkeys(fn for f in files for fn in read_test_functions(REPO_ROOT / f)))
        if not functions:
            continue
        entry = (pkg_dir, files, functions)
        if any(file_exists_at_ref(f, base_ref) for f in files):
            modified.append(entry)
        else:
            added.append(entry)
    return added, modified


def run_unit_tests(pkg_dir, functions, cwd):
    """Run specific test functions in a Go package."""
    run_pattern = "^(" + "|".join(re.escape(f) for f in functions) + ")$"
    cmd = [
        "go",
        "test",
        "./" + pkg_dir,
        f"-run={run_pattern}",
        "-json",
        "-count=1",
        "-timeout=120s",
    ]
    return subprocess.run(cmd, capture_output=True, text=True, cwd=str(cwd))


def run_unit_tests_on_main(pkg_dir, changed_files, functions, base_ref):
    """Run unit tests on main's codebase with the changed test files copied in."""
    with tempfile.TemporaryDirectory(prefix="regression_unit_") as tmpdir:
        worktree_dir = Path(tmpdir) / "worktree"
        r = git("worktree", "add", "--detach", str(worktree_dir), base_ref)
        if r.returncode != 0:
            print(f"    [error] worktree creation failed: {r.stderr.strip()}", file=sys.stderr)
            return None
        try:
            for rel_path in changed_files:
                dst = worktree_dir / rel_path
                dst.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(str(REPO_ROOT / rel_path), str(dst))
            return run_unit_tests(pkg_dir, functions, worktree_dir)
        finally:
            git("worktree", "remove", "--force", str(worktree_dir))


# ---------------------------------------------------------------------------
# Report rendering
# ---------------------------------------------------------------------------


def render_report(
    commit_header,
    base_commit,
    # acceptance
    acc_selected,
    acc_added,
    acc_branch_info,
    acc_main_info,
    acc_latest_info,
    latest_version,
    # unit
    unit_selected,
    unit_added_keys,
    unit_branch_info,
    unit_main_info,
):
    """Return the full markdown report as a string."""
    latest_label = f"v{latest_version}" if latest_version else "latest"
    n_acc_added = sum(1 for t in acc_selected if t in acc_added)
    n_unit_added = sum(1 for k in unit_selected if k in unit_added_keys)

    PASS = "✅"
    FAIL = "❌"
    NA = "➖"

    def mark(status):
        return PASS if status in ("pass", "skip") else (NA if not status else FAIL)

    # ---- classify ----
    acc_regression, acc_unreleased, acc_coverage, acc_failing = [], [], [], []
    for tp in acc_selected:
        info = acc_branch_info[tp]
        if not info["passing_leaves"]:
            acc_failing.append(tp)
            continue
        cats = {
            classify_acceptance_result(
                acc_main_info.get(l, {}).get("status", ""),
                acc_latest_info.get(l, {}).get("status", ""),
            )
            for l in info["passing_leaves"]
        }
        if "regression" in cats:
            acc_regression.append(tp)
        elif "unreleased" in cats:
            acc_unreleased.append(tp)
        else:
            acc_coverage.append(tp)

    unit_regression, unit_coverage, unit_failing = [], [], []
    for key in unit_selected:
        info = unit_branch_info[key]
        if info["compile_fail"] or not info["passing_functions"]:
            unit_failing.append(key)
            continue
        mr = unit_main_info.get(key, {})
        cat = classify_unit_result(
            mr.get("compile_fail", False),
            mr.get("function_results", {}),
            info["passing_functions"],
        )
        (unit_regression if cat == "regression" else unit_coverage).append(key)

    # ---- summary header ----
    def _summary(label, total, n_added, cats):
        parts = [f"{n} {name}" for name, n in cats if n]
        base = f"{label}: {total} ({n_added} added, {total - n_added} modified)"
        return base + (" — " + ", ".join(parts) if parts else "")

    lines = [
        "# Regression Test Report",
        "",
        commit_header,
        "",
        _summary(
            "Acceptance tests",
            len(acc_selected),
            n_acc_added,
            [
                ("regression", len(acc_regression)),
                ("unreleased", len(acc_unreleased)),
                ("coverage", len(acc_coverage)),
                ("failing", len(acc_failing)),
            ],
        ),
        _summary(
            "Unit tests",
            len(unit_selected),
            n_unit_added,
            [
                ("regression", len(unit_regression)),
                ("coverage", len(unit_coverage)),
                ("failing", len(unit_failing)),
            ],
        ),
        "",
    ]

    # ---- table ----
    col_main = f"main ({base_commit})"
    col_latest = f"latest ({latest_label})"
    columns = ["test", "branch", col_main, col_latest]
    rows = []

    for tp in acc_selected:
        info = acc_branch_info[tp]
        leaves = info["leaves"]
        passing_set = set(info["passing_leaves"])
        if leaves:
            for leaf in leaves:
                is_pass = leaf in passing_set
                if is_pass:
                    m = mark(acc_main_info.get(leaf, {}).get("status", ""))
                    l = mark(acc_latest_info.get(leaf, {}).get("status", ""))
                else:
                    m = l = NA
                rows.append({"test": leaf, "branch": PASS if is_pass else FAIL, col_main: m, col_latest: l})
        else:
            rows.append({"test": f"TestAccept/{tp}", "branch": FAIL, col_main: NA, col_latest: NA})

    for key in unit_selected:
        info = unit_branch_info[key]
        pkg = info["package_dir"]
        mr = unit_main_info.get(key, {})
        passing_set = set(info["passing_functions"])
        if info["compile_fail"]:
            rows.append({"test": f"{pkg} [cannot compile]", "branch": FAIL, col_main: NA, col_latest: NA})
            continue
        for func in info["all_functions"]:
            is_pass = func in passing_set
            if is_pass and not mr.get("compile_fail"):
                m = mark(mr.get("function_results", {}).get(func, ""))
            else:
                m = NA
            rows.append({"test": f"{pkg}: {func}", "branch": PASS if is_pass else FAIL, col_main: m, col_latest: NA})

    if rows:
        lines.append("| " + " | ".join(columns) + " |")
        lines.append("| " + " | ".join("---" for _ in columns) + " |")
        for row in rows:
            lines.append("| " + " | ".join(row.get(c, "") for c in columns) + " |")
        lines.append("")

    return "\n".join(lines) + "\n"


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument(
        "--output",
        default=str(REPO_ROOT / "REGRESSION_REPORT.md"),
        help="Output file path (default: REGRESSION_REPORT.md in repo root)",
    )
    parser.add_argument(
        "--commit",
        action="store_true",
        help="Create an empty git commit with the report as the message instead of writing a file",
    )
    parser.add_argument(
        "--max-tests",
        type=int,
        default=DEFAULT_MAX_TESTS,
        help=f"Max tests per category to analyze (default: {DEFAULT_MAX_TESTS}), added tests prioritized",
    )
    parser.add_argument(
        "--base",
        default=None,
        help="Base git ref for comparison (default: origin/main or main)",
    )
    args = parser.parse_args()

    base_ref = resolve_base_ref(args.base)
    base_commit = git("rev-parse", "--short", base_ref).stdout.strip()

    head_hash = git("rev-parse", "--short", "HEAD").stdout.strip()
    head_title = git("log", "-1", "--format=%s", "HEAD").stdout.strip()
    is_dirty = bool(git("status", "--porcelain", "--untracked-files=no").stdout.strip())
    commit_header = f"Tested commit: {head_hash}{'-dirty' if is_dirty else ''} {head_title}"

    print(f"Analyzing branch vs {base_ref}")
    print(f"HEAD: {commit_header}")

    changed_files = get_changed_files(base_ref)
    print(f"Files changed vs {base_ref}: {len(changed_files)}")

    acc_added, acc_modified = collect_changed_acceptance_tests(changed_files, base_ref)
    acc_selected = acc_added[: args.max_tests]
    acc_selected += acc_modified[: args.max_tests - len(acc_selected)]
    print(f"Acceptance tests: {len(acc_added)} added, {len(acc_modified)} modified → {len(acc_selected)} selected")

    unit_added, unit_modified = collect_changed_unit_tests(changed_files, base_ref)
    unit_selected_entries = unit_added[: args.max_tests]
    unit_selected_entries += unit_modified[: args.max_tests - len(unit_selected_entries)]
    unit_added_keys = {e[0] for e in unit_added}
    unit_selected = [e[0] for e in unit_selected_entries]
    unit_entry_map = {e[0]: e for e in unit_selected_entries}
    print(f"Unit tests:       {len(unit_added)} added, {len(unit_modified)} modified → {len(unit_selected)} selected")

    if not acc_selected and not unit_selected:
        report = (
            f"# Regression Test Report\n\n{commit_header}\n\nNo acceptance or unit tests were changed on this branch.\n"
        )
        _emit(report, args)
        return

    # Phase 1: run on current branch
    acc_branch_info: dict[str, dict] = {}
    unit_branch_info: dict[str, dict] = {}

    if acc_selected:
        print("\n=== Phase 1a: Running acceptance tests on current branch ===")
        for tp in acc_selected:
            tag = "(added)" if tp in acc_added else "(modified)"
            print(f"  {tp} {tag} ...", flush=True)
            pattern = "TestAccept/" + tp.replace(os.sep, "/")
            proc = run_go_test(pattern, REPO_ROOT)
            tests = parse_json_output(proc.stdout)
            parent_entry = tests.get(pattern)
            parent_status = parent_entry.status if parent_entry else ("fail" if proc.returncode != 0 else "")
            leaves = get_leaf_subtests(pattern, tests)
            passing_leaves = [l for l in leaves if tests.get(l, TestEntry(name=l)).status == "pass"]
            acc_branch_info[tp] = {
                "parent_status": parent_status,
                "leaves": leaves,
                "passing_leaves": passing_leaves,
            }
            print(f"    status={parent_status}, leaves={len(leaves)}, passing={len(passing_leaves)}")
            if proc.returncode != 0:
                print(readable_output(proc.stdout + proc.stderr))

    if unit_selected:
        print("\n=== Phase 1b: Running unit tests on current branch ===")
        for key in unit_selected:
            pkg_dir, changed_files_pkg, functions = unit_entry_map[key]
            tag = "(added)" if key in unit_added_keys else "(modified)"
            print(f"  {pkg_dir} {tag} ({len(functions)} functions) ...", flush=True)
            proc = run_unit_tests(pkg_dir, functions, REPO_ROOT)
            if is_compile_failure(proc):
                print("    cannot compile on branch")
                unit_branch_info[key] = {
                    "package_dir": pkg_dir,
                    "compile_fail": True,
                    "raw_output": proc.stdout,
                    "parent_status": "cannot_compile",
                    "all_functions": functions,
                    "passing_functions": [],
                }
                continue
            tests = parse_json_output(proc.stdout)
            passing = [f for f in functions if tests.get(f, TestEntry(name=f)).status == "pass"]
            parent_status = "pass" if passing else ("fail" if proc.returncode != 0 else "skip")
            unit_branch_info[key] = {
                "package_dir": pkg_dir,
                "compile_fail": False,
                "raw_output": proc.stdout,
                "parent_status": parent_status,
                "all_functions": functions,
                "passing_functions": passing,
            }
            print(f"    status={parent_status}, functions={len(functions)}, passing={len(passing)}")

    # Phase 2: compare against main (worktree)
    acc_main_info: dict[str, dict] = {}
    unit_main_info: dict[str, dict] = {}

    if acc_selected:
        print("\n=== Phase 2a: Testing acceptance tests against main (worktree) ===")
        for tp in acc_selected:
            for leaf in acc_branch_info[tp]["passing_leaves"]:
                print(f"  {leaf} ...", flush=True)
                proc = run_acceptance_on_main(tp, leaf, base_ref)
                if proc is None:
                    acc_main_info[leaf] = {"status": "error", "output": ""}
                    continue
                tests = parse_json_output(proc.stdout)
                entry = tests.get(leaf)
                status = entry.status if entry else ("fail" if proc.returncode != 0 else "unknown")
                acc_main_info[leaf] = {"status": status, "output": proc.stdout}
                print(f"    main status: {status}")

    if unit_selected:
        print("\n=== Phase 2b: Testing unit tests against main (worktree) ===")
        for key in unit_selected:
            info = unit_branch_info[key]
            if not info["passing_functions"]:
                continue
            pkg_dir, changed_files_pkg, functions = unit_entry_map[key]
            print(f"  {pkg_dir} ...", flush=True)
            proc = run_unit_tests_on_main(pkg_dir, changed_files_pkg, info["passing_functions"], base_ref)
            if proc is None:
                unit_main_info[key] = {"compile_fail": False, "function_results": {}, "raw_output": ""}
                continue
            if is_compile_failure(proc):
                print("    cannot compile on main")
                unit_main_info[key] = {"compile_fail": True, "raw_output": proc.stdout, "function_results": {}}
                continue
            tests = parse_json_output(proc.stdout)
            func_results = {f: (tests[f].status if f in tests else "not_found") for f in info["passing_functions"]}
            unit_main_info[key] = {"compile_fail": False, "function_results": func_results, "raw_output": proc.stdout}
            for f, s in func_results.items():
                print(f"    {f}: {s}")

    # Phase 3: compare acceptance tests against latest release
    acc_latest_info: dict[str, dict] = {}

    if acc_selected:
        print("\n=== Phase 3: Testing acceptance tests against latest release ===")
        for tp in acc_selected:
            for leaf in acc_branch_info[tp]["passing_leaves"]:
                print(f"  {leaf} ...", flush=True)
                proc = run_acceptance_with_latest_release(leaf, REPO_ROOT)
                tests = parse_json_output(proc.stdout)
                entry = tests.get(leaf)
                status = entry.status if entry else ("fail" if proc.returncode != 0 else "unknown")
                acc_latest_info[leaf] = {"status": status, "output": proc.stdout}
                print(f"    latest status: {status}")

    latest_version = read_latest_release_version()

    report = render_report(
        commit_header,
        base_commit,
        acc_selected,
        acc_added,
        acc_branch_info,
        acc_main_info,
        acc_latest_info,
        latest_version,
        unit_selected,
        unit_added_keys,
        unit_branch_info,
        unit_main_info,
    )

    _print_counts(
        acc_selected, acc_branch_info, acc_main_info, acc_latest_info, unit_selected, unit_branch_info, unit_main_info
    )
    _emit(report, args)


def _print_counts(
    acc_selected, acc_branch_info, acc_main_info, acc_latest_info, unit_selected, unit_branch_info, unit_main_info
):
    def acc_cat(tp):
        pl = acc_branch_info[tp]["passing_leaves"]
        if not pl:
            return "failing"
        cats = {
            classify_acceptance_result(
                acc_main_info.get(l, {}).get("status", ""),
                acc_latest_info.get(l, {}).get("status", ""),
            )
            for l in pl
        }
        return "regression" if "regression" in cats else ("unreleased" if "unreleased" in cats else "coverage")

    def unit_cat(key):
        info = unit_branch_info[key]
        if info["compile_fail"] or not info["passing_functions"]:
            return "failing"
        mr = unit_main_info.get(key, {})
        return classify_unit_result(
            mr.get("compile_fail", False),
            mr.get("function_results", {}),
            info["passing_functions"],
        )

    ac = Counter(acc_cat(tp) for tp in acc_selected)
    uc = Counter(unit_cat(k) for k in unit_selected)
    print(
        f"\nAcceptance: regression={ac['regression']} unreleased={ac['unreleased']} "
        f"coverage={ac['coverage']} failing={ac['failing']}"
    )
    print(f"Unit:       regression={uc['regression']} coverage={uc['coverage']} failing={uc['failing']}")


def _emit(report, args):
    """Write the report to a file or create an empty git commit."""
    if args.commit:
        r = git("commit", "--allow-empty", "-m", report)
        if r.returncode != 0:
            print(f"git commit failed: {r.stderr.strip()}", file=sys.stderr)
            sys.exit(1)
        print("\nEmpty commit created with regression report as message.")
    else:
        Path(args.output).write_text(report)
        print(f"\nReport saved to {args.output}")


if __name__ == "__main__":
    main()
