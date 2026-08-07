#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""
Regression test report generator for CLI acceptance tests.

Finds acceptance tests modified or added on the current branch vs origin/main, runs
them, and classifies each into one of these categories:

  Possible regression test  — passes on branch, fails on main's CLI.
                              Exercises behavior introduced by this branch.

  Unreleased behavior       — passes on branch and main's CLI, but fails with the
                              latest released CLI (-useversion latest). Tests
                              behavior already merged but not yet shipped; no
                              changelog entry needed for this PR.

  Additional coverage       — passes everywhere.

The three phases run the same tests against, respectively: this branch's CLI, a CLI
built from the base ref, and the latest released CLI. Phase 2 only swaps the CLI: a
binary is built from the base ref in a temp dir and the branch's own test runner and
tests run against it via -clipath, so the test infra and tests stay on this branch
while exercising main's CLI.

Run doctests: python3 -m doctest tools/regression_test_report.py

Usage:
    python3 tools/regression_test_report.py [--output PATH] [--max-tests N] [--base REF] [--cloud [ENV]] [FILTER ...]
    python3 tools/regression_test_report.py --commit [--max-tests N] [--base REF] [--cloud [ENV]] [FILTER ...]

FILTER arguments restrict the run to tests whose name contains one of the given
substrings (e.g. `no_drift alert.yml.tmpl` runs only matching invariant tests).

--cloud runs the acceptance tests against a real workspace instead of the mock
server, mirroring the testme-aws script (fetches workspace secrets via
testme-fetch-env and sets CLOUD_ENV). Defaults to the aws-prod-ucws environment.
"""

import argparse
import json
import os
import re
import shlex
import shutil
import subprocess
import sys
import tempfile
import time
from collections import Counter
from dataclasses import dataclass, field
from pathlib import Path

_git_root = subprocess.run(["git", "rev-parse", "--show-toplevel"], capture_output=True, text=True)
REPO_ROOT = Path(_git_root.stdout.strip()) if _git_root.returncode == 0 else Path(__file__).parent.parent
ACCEPTANCE_DIR = REPO_ROOT / "acceptance"
DEFAULT_MAX_TESTS = 20
INVARIANT_DIR = "bundle/invariant"
INVARIANT_CONFIGS_PREFIX = "acceptance/bundle/invariant/configs/"
INVARIANT_FILE_PREFIX = "acceptance/bundle/invariant/"
# Under an invariant subdir these regenerate whenever the EnvMatrix config list
# changes, so a change here must not trigger a full all-configs run — the per-config
# scoped runs (from configs/) cover the actual change.
INVARIANT_NON_TRIGGERING_NAMES = ("out.test.toml", "test.toml")


@dataclass(frozen=True)
class AccTest:
    """An acceptance test to run.

    For invariant tests a single config maps to many EnvMatrix variants; setting
    `config` scopes the -run pattern to that one INPUT_CONFIG instead of all of them.
    """

    parent: str  # real test-name prefix used to find leaves, e.g. "TestAccept/bundle/foo"
    is_new: bool  # True if added on this branch (False = modified)
    config: str = ""  # invariant INPUT_CONFIG to scope to; "" runs the whole test
    optional: bool = False  # drop if it produces no leaves (e.g. variant excluded in this subdir)

    @property
    def run(self):
        """The -run regex passed to `go test`.

        >>> AccTest("TestAccept/bundle/x", False).run
        'TestAccept/bundle/x'
        >>> AccTest("TestAccept/bundle/invariant/no_drift", True, "a.yml.tmpl").run
        'TestAccept/bundle/invariant/no_drift/.*/INPUT_CONFIG=a\\\\.yml\\\\.tmpl$'
        """
        if not self.config:
            return self.parent
        escaped = self.config.replace(".", r"\.")
        return rf"{self.parent}/.*/INPUT_CONFIG={escaped}$"

    @property
    def key(self):
        return self.run

    @property
    def label(self):
        return f"{self.parent} [{self.config}]" if self.config else self.parent


# ---------------------------------------------------------------------------
# Pure functions — no filesystem, git, or subprocess access.
# These encode all classification and parsing logic and are testable with
# `python3 -m doctest tools/regression_test_report.py`.
# ---------------------------------------------------------------------------


def config_from_path(rel_path):
    """Return the INPUT_CONFIG name for a changed invariant config file, or None.

    >>> config_from_path('acceptance/bundle/invariant/configs/job.yml.tmpl')
    'job.yml.tmpl'
    >>> config_from_path('acceptance/bundle/invariant/configs/job.yml.tmpl-init.sh')
    'job.yml.tmpl'
    >>> config_from_path('acceptance/bundle/invariant/configs/job.yml.tmpl-cleanup.sh')
    'job.yml.tmpl'
    >>> config_from_path('acceptance/bundle/invariant/no_drift/script') is None
    True
    >>> config_from_path('acceptance/bundle/invariant/configs/README.md') is None
    True
    """
    if not rel_path.startswith(INVARIANT_CONFIGS_PREFIX):
        return None
    name = rel_path[len(INVARIANT_CONFIGS_PREFIX) :]
    m = re.match(r"(.+\.yml\.tmpl)(-init\.sh|-cleanup\.sh)?$", name)
    return m.group(1) if m else None


def _is_invariant_config_list_file(rel_path):
    """Return True for an invariant out.test.toml/test.toml (tracks the config list).

    These regenerate when the EnvMatrix config list changes, so they must not trigger
    a full all-configs invariant run.

    >>> _is_invariant_config_list_file('acceptance/bundle/invariant/migrate/out.test.toml')
    True
    >>> _is_invariant_config_list_file('acceptance/bundle/invariant/test.toml')
    True
    >>> _is_invariant_config_list_file('acceptance/bundle/invariant/migrate/script')
    False
    >>> _is_invariant_config_list_file('acceptance/bundle/other/out.test.toml')
    False
    """
    return rel_path.startswith(INVARIANT_FILE_PREFIX) and Path(rel_path).name in INVARIANT_NON_TRIGGERING_NAMES


def matches_filters(name, filters):
    """Return True if name contains any filter substring, or no filters are given.

    >>> matches_filters('TestAccept/bundle/foo', [])
    True
    >>> matches_filters('TestAccept/bundle/foo', ['foo'])
    True
    >>> matches_filters('TestAccept/bundle/foo', ['bar'])
    False
    >>> matches_filters('TestAccept/bundle/foo', ['bar', 'foo'])
    True
    """
    return not filters or any(f in name for f in filters)


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
    """Return the symbolic base ref for comparison, falling back from origin/main to main."""
    if hint:
        return hint
    if git("rev-parse", "--verify", "origin/main").returncode == 0:
        return "origin/main"
    return "main"


def compute_merge_base(base_ref):
    """Return the merge-base SHA between HEAD and base_ref.

    Using the merge base rather than the tip of base_ref ensures the diff
    only reflects changes on the current branch, not commits that landed on
    main after the branch was created.
    """
    result = git("merge-base", "HEAD", base_ref)
    if result.returncode != 0:
        # Fall back to the tip of base_ref if merge-base fails (e.g. unrelated histories).
        return git("rev-parse", base_ref).stdout.strip()
    return result.stdout.strip()


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
    """Return a list of AccTest to run, added first.

    Invariant configs (acceptance/bundle/invariant/configs/*) are special: each one
    feeds every invariant subdir (no_drift, migrate, ...) as an EnvMatrix variant.
    A changed config maps to a scoped run for just that INPUT_CONFIG, so we don't
    re-run all ~30 configs. Editing an invariant subdir file (script/output) still
    triggers the full run for that subdir, but out.test.toml/test.toml changes do
    not (they track the config list and would otherwise force every config).
    """
    test_dirs = scan_acceptance_test_dirs(ACCEPTANCE_DIR)
    invariant_dirs = sorted(d for d in test_dirs if d.startswith(INVARIANT_DIR + "/"))

    tests = []

    # Regular acceptance tests (including full invariant-subdir runs when a file in
    # the subdir itself changed).
    mapping_files = [f for f in changed_files if not _is_invariant_config_list_file(f)]
    test_paths = sorted(
        {tp for tp in (find_acceptance_test_for_path(f, test_dirs) for f in mapping_files) if tp is not None}
    )
    for tp in test_paths:
        is_new = not file_exists_at_ref(f"acceptance/{tp}/script", base_ref)
        tests.append(AccTest(parent="TestAccept/" + tp.replace(os.sep, "/"), is_new=is_new))

    # Invariant config changes → one scoped run per invariant subdir for that config.
    configs = {c: config_from_path(f) for f in changed_files if (c := config_from_path(f))}
    for config in sorted(configs):
        is_new = not file_exists_at_ref(f"{INVARIANT_CONFIGS_PREFIX}{config}", base_ref)
        for sub in invariant_dirs:
            tests.append(
                AccTest(
                    parent="TestAccept/" + sub.replace(os.sep, "/"),
                    is_new=is_new,
                    config=config,
                    optional=True,
                )
            )

    # A full subdir run supersedes scoped runs of the same subdir.
    full_parents = {t.parent for t in tests if not t.config}
    tests = [t for t in tests if not t.config or t.parent not in full_parents]

    # Stable sort keeps regular-then-invariant order within each added/modified group.
    return sorted(tests, key=lambda t: not t.is_new)


def run_go_test(pattern, cwd, extra_args=None, env=None):
    """Run go test ./acceptance -run PATTERN -json and return CompletedProcess.

    When env carries CLOUD_ENV (i.e. --cloud), tests deploy to a real workspace, so
    we allow the same 1h timeout the testme script uses instead of the 5m local one.
    """
    timeout = "3600s" if env and env.get("CLOUD_ENV") else "300s"
    cmd = [
        "go",
        "test",
        "./acceptance",
        f"-run={pattern}",
        "-json",
        "-count=1",
        f"-timeout={timeout}",
    ] + (extra_args or [])
    return subprocess.run(cmd, capture_output=True, text=True, cwd=str(cwd), env=env)


def build_main_cli(ref):
    """Build a CLI binary from `ref` in a temp dir; return (cli_path, tmpdir).

    Only the CLI comes from main: we export main's source tree at `ref` (via
    `git archive`, no .git or full checkout) and `go build` it. The acceptance test
    runner and the tests themselves still come from the current branch — Phase 2 runs
    them with this binary via -clipath. The caller must remove tmpdir.
    """
    tmpdir = Path(tempfile.mkdtemp(prefix="regression_main_cli_"))
    src = tmpdir / "src"
    src.mkdir()
    r = subprocess.run(
        f"git archive --format=tar {shlex.quote(ref)} | tar -x -C {shlex.quote(str(src))}",
        shell=True,
        cwd=str(REPO_ROOT),
        capture_output=True,
        text=True,
    )
    if r.returncode != 0:
        shutil.rmtree(tmpdir, ignore_errors=True)
        sys.exit(f"[ERROR] failed to export main source at {ref}: {r.stderr.strip()}")

    cli = src / ("databricks.exe" if os.name == "nt" else "databricks")
    print(f"  Building CLI from {ref[:12]} ...", flush=True)
    # -buildvcs=false: the exported tree has no .git, so VCS stamping would fail.
    r = subprocess.run(
        ["go", "build", "-buildvcs=false", "-o", str(cli), "."],
        cwd=str(src),
        capture_output=True,
        text=True,
    )
    if r.returncode != 0:
        shutil.rmtree(tmpdir, ignore_errors=True)
        sys.exit(f"[ERROR] failed to build CLI from {ref}: {r.stderr.strip()}")
    return str(cli), tmpdir


def run_acceptance_on_main(subtest, main_cli, env=None):
    """Run a branch acceptance subtest with a CLI built from main (via -clipath)."""
    return run_go_test(subtest, REPO_ROOT, extra_args=["-clipath", main_cli], env=env)


def run_acceptance_with_latest_release(subtest, version, cwd, env=None):
    """Run acceptance subtest against the given released CLI version."""
    return run_go_test(subtest, cwd, extra_args=["-useversion", version], env=env)


def load_cloud_env(env_name):
    """Return the environment for running acceptance tests on the `env_name` cloud.

    Mirrors the testme-aws / testme-env scripts: fetch the workspace secrets into
    ~/.cache/testme-envs/<env>.env via testme-fetch-env (refreshing when missing or
    stale), source them, and point DATABRICKS_CONFIG_FILE at /dev/null so the vault
    secrets win. The resulting CLOUD_ENV / auth / TEST_METASTORE_ID variables are what
    make the acceptance framework target a real workspace instead of the mock server.
    """
    cache_file = Path.home() / ".cache" / "testme-envs" / f"{env_name}.env"
    ttl_minutes = int(os.environ.get("TESTME_TTL_MINUTES", "720"))
    stale = not cache_file.exists() or (time.time() - cache_file.stat().st_mtime) > ttl_minutes * 60
    if stale:
        cache_file.parent.mkdir(parents=True, exist_ok=True)
        r = subprocess.run(["testme-fetch-env", "--env", env_name, "--out", str(cache_file)])
        if r.returncode != 0:
            if not cache_file.exists():
                sys.exit(f"[ERROR] testme-fetch-env failed and no cache exists at {cache_file}")
            print(f"[WARN] testme-fetch-env failed; using stale cache at {cache_file}", file=sys.stderr)

    # Source the env file in bash and capture the resulting environment (env -0 is
    # null-delimited so values with spaces or newlines survive).
    proc = subprocess.run(
        ["bash", "-c", f"set -a; source {shlex.quote(str(cache_file))}; env -0"],
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        sys.exit(f"[ERROR] failed to source {cache_file}: {proc.stderr.strip()}")
    env = dict(kv.split("=", 1) for kv in proc.stdout.split("\0") if "=" in kv)
    env["DATABRICKS_CONFIG_FILE"] = "/dev/null"
    if not env.get("CLOUD_ENV"):
        sys.exit(f"[ERROR] {cache_file} did not set CLOUD_ENV; cannot run on cloud")
    return env


def fetch_latest_release_version():
    """Return the latest released CLI version (e.g. '0.321.0'), with a 1-hour file cache."""
    import urllib.request

    r = subprocess.run(["go", "env", "GOOS", "GOARCH"], capture_output=True, text=True, cwd=str(REPO_ROOT))
    if r.returncode == 0:
        goos, goarch = r.stdout.strip().split("\n", 1)
        cache = ACCEPTANCE_DIR / "build" / f"{goos}_{goarch}" / "latest_version.txt"
        if cache.exists():
            if time.time() - cache.stat().st_mtime < 3600:
                version = cache.read_text().strip()
                if version:
                    return version
    else:
        cache = None

    url = "https://api.github.com/repos/databricks/cli/releases/latest"
    with urllib.request.urlopen(url, timeout=10) as resp:
        data = json.loads(resp.read())
    version = data.get("tag_name", "").lstrip("v")
    if not version:
        return None
    if cache is not None:
        cache.parent.mkdir(parents=True, exist_ok=True)
        cache.write_text(version)
    return version


# ---------------------------------------------------------------------------
# Report rendering
# ---------------------------------------------------------------------------


def render_report(
    commit_header,
    base_commit,
    acc_selected,
    acc_branch_info,
    acc_main_info,
    acc_latest_info,
    latest_version,
):
    """Return the full markdown report as a string.

    acc_selected is a list of AccTest; acc_branch_info is keyed by AccTest.key.
    """
    n_acc_added = sum(1 for at in acc_selected if at.is_new)

    PASS = "✅"
    FAIL = "❌"
    NA = "➖"

    def mark(status):
        return PASS if status in ("pass", "skip") else (NA if not status else FAIL)

    # ---- classify ----
    acc_regression, acc_unreleased, acc_coverage, acc_failing = [], [], [], []
    for at in acc_selected:
        info = acc_branch_info[at.key]
        if not info["passing_leaves"]:
            acc_failing.append(at)
            continue
        cats = {
            classify_acceptance_result(
                acc_main_info.get(leaf, {}).get("status", ""),
                acc_latest_info.get(leaf, {}).get("status", ""),
            )
            for leaf in info["passing_leaves"]
        }
        if "regression" in cats:
            acc_regression.append(at)
        elif "unreleased" in cats:
            acc_unreleased.append(at)
        else:
            acc_coverage.append(at)

    # ---- summary header ----
    def _summary(label, total, n_added, cats):
        parts = [f"{n} {name}" for name, n in cats if n]
        base = f"{label}: {total} ({n_added} added, {total - n_added} modified)"
        return base + (" — " + ", ".join(parts) if parts else "")

    acc_summary = _summary(
        "Acceptance tests",
        len(acc_selected),
        n_acc_added,
        [
            ("regression", len(acc_regression)),
            ("unreleased", len(acc_unreleased)),
            ("coverage", len(acc_coverage)),
            ("failing", len(acc_failing)),
        ],
    )

    lines = [
        "# Regression Test Report",
        "",
        commit_header,
        "",
        f"<!-- {acc_summary} -->",
        "",
    ]

    # ---- table ----
    col_main = f"main ({base_commit})"
    col_latest = f"latest (v{latest_version})" if latest_version else "latest"
    columns = ["test", "branch", col_main, col_latest]
    rows = []

    for at in acc_selected:
        info = acc_branch_info[at.key]
        leaves = info["leaves"]
        passing_set = set(info["passing_leaves"])
        if leaves:
            for leaf in leaves:
                is_pass = leaf in passing_set
                if is_pass:
                    m = mark(acc_main_info.get(leaf, {}).get("status", ""))
                    lat = mark(acc_latest_info.get(leaf, {}).get("status", ""))
                else:
                    m = lat = NA
                rows.append({"test": leaf, "branch": PASS if is_pass else FAIL, col_main: m, col_latest: lat})
        else:
            rows.append({"test": at.label, "branch": FAIL, col_main: NA, col_latest: NA})

    if rows:
        lines.append("| " + " | ".join(columns) + " |")
        lines.append("| " + " | ".join("---" for _ in columns) + " |")
        for row in rows:
            lines.append("| " + " | ".join(row.get(c, "") for c in columns) + " |")
        lines.append("")

    # ---- details sections for failing tests ----
    for at in acc_selected:
        info = acc_branch_info[at.key]
        leaf_entries_branch = info.get("leaf_entries", {})

        if not info["leaves"]:
            leaf = at.label
            entry = leaf_entries_branch.get(leaf)
            text = entry.output if entry else readable_output(info.get("raw_output", ""), max_lines=10000)
            lines.append("<details>")
            lines.append(f"<summary>{leaf} {FAIL} | {col_main} {NA} | {col_latest} {NA}</summary>")
            lines.append("")
            if text.strip():
                lines.append("**branch:**")
                lines.append("```")
                lines.append(text.rstrip())
                lines.append("```")
                lines.append("")
            lines.append("</details>")
            lines.append("")
            continue

        for leaf in info["leaves"]:
            branch_pass = leaf in set(info["passing_leaves"])
            main_status = acc_main_info.get(leaf, {}).get("status", "")
            latest_status = acc_latest_info.get(leaf, {}).get("status", "")

            has_branch_fail = not branch_pass
            has_main_fail = main_status not in ("pass", "skip", "")
            has_latest_fail = latest_status not in ("pass", "skip", "")

            if not (has_branch_fail or has_main_fail or has_latest_fail):
                continue

            b_mark = PASS if branch_pass else FAIL
            m_mark = mark(main_status) if main_status else NA
            l_mark = mark(latest_status) if latest_status else NA
            summary = f"{leaf} {b_mark} | {col_main} {m_mark} | {col_latest} {l_mark}"

            lines.append("<details>")
            lines.append(f"<summary>{summary}</summary>")
            lines.append("")

            if has_branch_fail:
                entry = leaf_entries_branch.get(leaf)
                text = entry.output if entry else readable_output(info.get("raw_output", ""), max_lines=10000)
                if text.strip():
                    lines.append("**branch:**")
                    lines.append("```")
                    lines.append(text.rstrip())
                    lines.append("```")
                    lines.append("")

            if has_main_fail:
                raw = acc_main_info.get(leaf, {}).get("output", "")
                parsed = parse_json_output(raw)
                entry = parsed.get(leaf)
                text = entry.output if entry else readable_output(raw, max_lines=10000)
                if text.strip():
                    lines.append(f"**{col_main}:**")
                    lines.append("```")
                    lines.append(text.rstrip())
                    lines.append("```")
                    lines.append("")

            if has_latest_fail:
                raw = acc_latest_info.get(leaf, {}).get("output", "")
                parsed = parse_json_output(raw)
                entry = parsed.get(leaf)
                text = entry.output if entry else readable_output(raw, max_lines=10000)
                if text.strip():
                    lines.append(f"**{col_latest}:**")
                    lines.append("```")
                    lines.append(text.rstrip())
                    lines.append("```")
                    lines.append("")

            lines.append("</details>")
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
        help=f"Max acceptance tests to analyze (default: {DEFAULT_MAX_TESTS}), added tests prioritized",
    )
    parser.add_argument(
        "--base",
        default=None,
        help="Base git ref for comparison (default: origin/main or main)",
    )
    parser.add_argument(
        "--cloud",
        nargs="?",
        const="aws-prod-ucws",
        default=None,
        metavar="ENV",
        help="Run acceptance tests on a real cloud workspace (default env: aws-prod-ucws), mirroring testme-aws",
    )
    parser.add_argument(
        "filters",
        nargs="*",
        help="Only check tests whose name contains one of these substrings",
    )
    args = parser.parse_args()

    cloud_env = None
    if args.cloud:
        print(f"Running acceptance tests on cloud: {args.cloud}")
        cloud_env = load_cloud_env(args.cloud)

    base_ref = resolve_base_ref(args.base)
    merge_base = compute_merge_base(base_ref)
    base_commit = git("rev-parse", "--short", merge_base).stdout.strip()

    head_hash = git("rev-parse", "--short", "HEAD").stdout.strip()
    head_title = git("log", "-1", "--format=%s", "HEAD").stdout.strip()
    is_dirty = bool(git("status", "--porcelain", "--untracked-files=no").stdout.strip())
    commit_header = f"Tested commit: {head_hash}{'-dirty' if is_dirty else ''} {head_title}"

    print(f"Analyzing branch vs {base_ref} (merge base: {base_commit})")
    print(f"HEAD: {commit_header}")

    changed_files = get_changed_files(merge_base)
    print(f"Files changed vs merge base ({base_commit}): {len(changed_files)}")

    if args.filters:
        print(f"Filtering tests by substring: {', '.join(args.filters)}")

    acc_tests = collect_changed_acceptance_tests(changed_files, merge_base)
    acc_tests = [t for t in acc_tests if matches_filters(t.label, args.filters)]
    acc_added = [t for t in acc_tests if t.is_new]
    acc_modified = [t for t in acc_tests if not t.is_new]
    acc_selected = acc_added[: args.max_tests]
    acc_selected += acc_modified[: args.max_tests - len(acc_selected)]
    print(f"Acceptance tests: {len(acc_added)} added, {len(acc_modified)} modified → {len(acc_selected)} selected")

    if not acc_selected:
        report = f"# Regression Test Report\n\n{commit_header}\n\nNo acceptance tests were changed on this branch.\n"
        _emit(report, args)
        return

    # Phase 1: run on current branch
    acc_branch_info: dict[str, dict] = {}

    if acc_selected:
        print("\n=== Phase 1: Running acceptance tests on current branch ===")
        acc_effective = []
        for at in acc_selected:
            tag = "(added)" if at.is_new else "(modified)"
            print(f"  {at.label} {tag} ...", flush=True)
            proc = run_go_test(at.run, REPO_ROOT, env=cloud_env)
            tests = parse_json_output(proc.stdout)
            leaves = get_leaf_subtests(at.parent, tests)
            # An optional (scoped invariant) run with no leaves means the variant is
            # excluded in this subdir; drop it rather than report it as failing.
            if at.optional and not leaves and proc.returncode == 0:
                print("    not applicable (variant excluded), skipping")
                continue
            parent_entry = tests.get(at.parent)
            parent_status = parent_entry.status if parent_entry else ("fail" if proc.returncode != 0 else "")
            passing_leaves = [leaf for leaf in leaves if tests.get(leaf, TestEntry(name=leaf)).status == "pass"]
            acc_branch_info[at.key] = {
                "parent_status": parent_status,
                "leaves": leaves,
                "passing_leaves": passing_leaves,
                "leaf_entries": tests,
            }
            acc_effective.append(at)
            print(f"    status={parent_status}, leaves={len(leaves)}, passing={len(passing_leaves)}")
            if proc.returncode != 0:
                print(readable_output(proc.stdout + proc.stderr))
        acc_selected = acc_effective

    # Phase 2: compare against a CLI built from main
    acc_main_info: dict[str, dict] = {}

    if acc_selected:
        print("\n=== Phase 2: Testing acceptance tests against main CLI ===")
        main_cli, main_cli_dir = build_main_cli(merge_base)
        try:
            for at in acc_selected:
                for leaf in acc_branch_info[at.key]["passing_leaves"]:
                    print(f"  {leaf} ...", flush=True)
                    proc = run_acceptance_on_main(leaf, main_cli, env=cloud_env)
                    tests = parse_json_output(proc.stdout)
                    entry = tests.get(leaf)
                    status = entry.status if entry else ("fail" if proc.returncode != 0 else "unknown")
                    acc_main_info[leaf] = {"status": status, "output": proc.stdout}
                    print(f"    main status: {status}")
        finally:
            shutil.rmtree(main_cli_dir, ignore_errors=True)

    # Phase 3: compare acceptance tests against latest release
    acc_latest_info: dict[str, dict] = {}
    latest_version = None

    if acc_selected:
        print("\n=== Phase 3: Testing acceptance tests against latest release ===")
        try:
            latest_version = fetch_latest_release_version()
        except Exception as e:
            print(f"  Warning: could not fetch latest release version: {e}")
        if latest_version:
            print(f"  Latest release: v{latest_version}")
            for at in acc_selected:
                for leaf in acc_branch_info[at.key]["passing_leaves"]:
                    print(f"  {leaf} ...", flush=True)
                    proc = run_acceptance_with_latest_release(leaf, latest_version, REPO_ROOT, env=cloud_env)
                    tests = parse_json_output(proc.stdout)
                    entry = tests.get(leaf)
                    status = entry.status if entry else ("fail" if proc.returncode != 0 else "unknown")
                    acc_latest_info[leaf] = {"status": status, "output": proc.stdout}
                    print(f"    latest status: {status}")

    report = render_report(
        commit_header,
        base_commit,
        acc_selected,
        acc_branch_info,
        acc_main_info,
        acc_latest_info,
        latest_version,
    )

    _print_counts(acc_selected, acc_branch_info, acc_main_info, acc_latest_info)
    _emit(report, args)


def _print_counts(acc_selected, acc_branch_info, acc_main_info, acc_latest_info):
    def acc_cat(at):
        pl = acc_branch_info[at.key]["passing_leaves"]
        if not pl:
            return "failing"
        cats = {
            classify_acceptance_result(
                acc_main_info.get(leaf, {}).get("status", ""),
                acc_latest_info.get(leaf, {}).get("status", ""),
            )
            for leaf in pl
        }
        return "regression" if "regression" in cats else ("unreleased" if "unreleased" in cats else "coverage")

    ac = Counter(acc_cat(at) for at in acc_selected)
    print(
        f"\nAcceptance: regression={ac['regression']} unreleased={ac['unreleased']} "
        f"coverage={ac['coverage']} failing={ac['failing']}"
    )


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
