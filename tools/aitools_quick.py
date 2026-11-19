#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""
Fast check for the aitools command group, meant to run after every edit.

Runs golangci-lint (which includes go vet's analyzers) and the unit tests of the
aitools packages, and compiles the packages that import them, in parallel. It
prints a summary for every step and detailed output for failed steps; file:line
findings are captured when available. Packages changed on the branch (diff
against the merge base with main, as tools/lintdiff.py does) are linted and
tested too, so an edit to a shared package such as libs/telemetry runs that
package's tests. Beyond that there is no package selection: Go's build and test
caches re-run exactly the packages whose transitive inputs changed.

The tests run in a sanitized test environment: the host's coding-agent variables
and DATABRICKS_* variables are removed and HOME points at a dedicated directory.
This prevents agent-detection variables from changing aitools behavior relative
to CI.

With --full (`./task aitools-full`) it also runs the importers' unit tests, the
acceptance tests that exercise aitools and the deadcode check: the slower tier to
run before a PR.

Usage:
    ./task aitools-quick
    ./task aitools-quick -- --json report.json
    ./task aitools-full
"""

import argparse
import json
import os
import re
import subprocess
import sys
import time
from concurrent.futures import ThreadPoolExecutor

PACKAGES = ["./cmd/aitools/...", "./libs/aitools/...", "./experimental/aitools/..."]

# Packages outside aitools that import it. cmd/root calls aitools from the root
# command's pre-run, so a change there can break every command.
IMPORTERS = [".", "./cmd", "./cmd/air", "./cmd/apps", "./cmd/experimental", "./cmd/root", "./cmd/version"]

# Acceptance tests that run aitools: its own directories, plus the goldens that
# print aitools help or tips. The apps tests cover the `apps init` skills tip.
ACCEPTANCE = [
    "TestAccept/apps",
    "TestAccept/cmd/aitools",
    "TestAccept/experimental/aitools",
    "TestAccept/help$",
    "TestAccept/cmd/version",
    "TestAccept/quickstart",
    "TestAccept/air/help",
    "TestAccept/air/config-help",
]

# Coding-agent detection variables (keep in sync with the list in
# acceptance/acceptance_test.go) plus the agent config-dir overrides aitools reads.
AGENT_ENV = {
    "AGENT",
    "AI_AGENT",
    "AIDEVKIT_HOME",
    "AMP_CURRENT_THREAD_ID",
    "ANTIGRAVITY_AGENT",
    "AUGMENT_AGENT",
    "CLAUDECODE",
    "CLAUDE_CODE_SESSION_ID",
    "CLAUDE_CONFIG_DIR",
    "CLINE_ACTIVE",
    "CODEX_CI",
    "CODEX_HOME",
    "COPILOT_CLI",
    "CURSOR_AGENT",
    "GEMINI_CLI",
    "GEMINI_CLI_HOME",
    "GOOSE_PATH_ROOT",
    "GOOSE_TERMINAL",
    "KIRO",
    "OMNIGENT",
    "OPENCLAW_SHELL",
    "OPENCODE",
    "PI_CODING_AGENT",
    "PI_CODING_AGENT_DIR",
    "VSCODE_AGENT",
    "WINDSURF_AGENT",
    "XDG_CONFIG_HOME",
}

# file.go:line[:col]: message, as printed by the compiler, go vet and golangci-lint.
FINDING = re.compile(r"^(?P<file>[^\s:][^:]*\.go):(?P<line>\d+)(?::(?P<col>\d+))?: (?P<msg>.+)$")


def run(name, cmd, env=None):
    start = time.monotonic()
    proc = subprocess.run(cmd, capture_output=True, text=True, env=env)
    return {
        "step": name,
        "cmd": " ".join(cmd),
        "ok": proc.returncode == 0,
        "seconds": round(time.monotonic() - start, 2),
        "output": proc.stdout + proc.stderr,
    }


def findings(output):
    """Return file:line findings from compiler, vet or lint output.

    >>> findings("# pkg\\ncmd/aitools/x.go:3:5: undefined: y\\nok")
    [{'file': 'cmd/aitools/x.go', 'line': 3, 'col': 5, 'message': 'undefined: y'}]
    """
    result = []
    for line in output.splitlines():
        m = FINDING.match(line.strip())
        if m:
            col = int(m["col"]) if m["col"] else None
            result.append({"file": m["file"], "line": int(m["line"]), "col": col, "message": m["msg"]})
    return result


def test_failures(output):
    """Return failing tests and their output from `go test -json` output.

    Build failures are reported as a pseudo-test named "(build)".
    """
    logs = {}
    failed = []
    for line in output.splitlines():
        try:
            ev = json.loads(line)
        except json.JSONDecodeError:
            continue
        pkg = ev.get("Package", "") or ev.get("ImportPath", "")
        key = (pkg, ev.get("Test") or "(build)")
        if ev.get("Action") in ("output", "build-output"):
            logs.setdefault(key, []).append(ev.get("Output", ""))
        elif ev.get("Action") in ("fail", "build-fail") and (ev.get("Test") or ev.get("Action") == "build-fail"):
            failed.append(key)
    # A failing subtest also fails its parent; report only the innermost test.
    names = {(p, t) for p, t in failed}
    leaves = [(p, t) for p, t in failed if not any(q == p and u.startswith(t + "/") for q, u in names)]
    return [{"package": p, "test": t, "output": "".join(logs.get((p, t), []))} for p, t in leaves]


def select_packages(paths):
    """Return ./dir patterns for the Go packages among changed paths, outside PACKAGES.

    The acceptance package runs the whole acceptance suite and integration tests
    need a workspace, so both are left to aitools-full; tools/ is its own module.

    >>> select_packages(["libs/telemetry/logger.go", "cmd/aitools/x_test.go", "acceptance/a/script",
    ...                  "acceptance/acceptance_test.go", "tools/x.go", "README.md", "libs/x/testdata/a.go"])
    ['./libs/telemetry']
    """
    dirs = set()
    for path in paths:
        if not path.endswith(".go") or "/testdata/" in path:
            continue
        d = os.path.dirname(path)
        if d.split("/")[0] in ("acceptance", "integration", "tools") or any(
            (d + "/").startswith(p.removeprefix("./").removesuffix("...")) for p in PACKAGES
        ):
            continue
        dirs.add("./" + d)
    return sorted(dirs)


def changed_packages(ref):
    """Packages with Go files changed since the merge base with ref, committed or not."""
    diff = subprocess.run(
        ["git", "diff", "--name-only", "--merge-base", ref, "--", "."], capture_output=True, text=True
    )
    if diff.returncode != 0:
        print(f"aitools-quick: cannot diff against {ref}; checking the aitools packages only", file=sys.stderr)
        return []
    untracked = subprocess.run(["git", "ls-files", "--others", "--exclude-standard"], capture_output=True, text=True)
    paths = diff.stdout.split() + untracked.stdout.split()
    return [d for d in select_packages(paths) if os.path.isdir(d)]


def user_cache_dir():
    """The per-user cache directory, as Go's os.UserCacheDir picks it."""
    if sys.platform == "win32":
        return os.environ["LocalAppData"]
    if sys.platform == "darwin":
        return os.path.expanduser("~/Library/Caches")
    return os.environ.get("XDG_CACHE_HOME") or os.path.expanduser("~/.cache")


def hermetic_env(home):
    keys = ["GOCACHE", "GOENV", "GOMODCACHE", "GOPATH", "GOTOOLCHAIN"]
    out = subprocess.run(["go", "env", "-json", *keys], capture_output=True, text=True, check=True)
    pinned = json.loads(out.stdout)
    env = {k: v for k, v in os.environ.items() if k not in AGENT_ENV and not k.startswith("DATABRICKS_")}
    # Pin the Go caches and the user's go env file (GOPROXY, GOFLAGS and the like
    # set with `go env -w`) before moving HOME: otherwise every run would start
    # cold and the tests would run under a different Go configuration than lint
    # and build.
    env.update({k: v for k, v in pinned.items() if v})
    env["HOME"] = home
    env["USERPROFILE"] = home
    return env


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--json", metavar="PATH", help="also write a machine-readable report to PATH")
    parser.add_argument("--ref", default="main", help="ref whose merge base defines the changed packages")
    parser.add_argument(
        "--full", action="store_true", help="also run importer tests, aitools acceptance tests and deadcode"
    )
    args = parser.parse_args()
    name = "aitools-full" if args.full else "aitools-quick"

    # A fixed HOME that is not wiped between runs keeps the test cache valid:
    # the cache key covers the env vars and files the tests read, HOME included.
    # It lives in the user's cache directory, not the shared temp directory,
    # where another account could create it first.
    home = os.path.join(user_cache_dir(), "databricks-cli", "aitools-quick-home")
    os.makedirs(home, mode=0o700, exist_ok=True)

    packages = PACKAGES + changed_packages(args.ref)
    lint = [
        "go",
        "tool",
        "-modfile=tools/go.mod",
        "golangci-lint",
        "run",
        "--allow-parallel-runners",
        "-j=4",
        *packages,
    ]
    if args.full:
        steps = [
            ("lint", lint, None),
            ("test", ["go", "test", "-json", *packages, *IMPORTERS], hermetic_env(home)),
            ("acceptance", ["go", "test", "-json", "./acceptance", "-run", "|".join(ACCEPTANCE)], hermetic_env(home)),
            ("deadcode", ["uv", "run", "--script", "tools/check_deadcode.py"], None),
        ]
    else:
        # The quick tier only checks that the importers still compile; their
        # tests run in aitools-full.
        steps = [
            ("lint", lint, None),
            # A hung test fails in 2 minutes instead of go test's default 10.
            ("test", ["go", "test", "-json", "-timeout=2m", *packages], hermetic_env(home)),
            ("build", ["go", "build", *IMPORTERS], None),
        ]
    start = time.monotonic()
    with ThreadPoolExecutor(len(steps)) as pool:
        results = list(pool.map(lambda s: run(*s), steps))
    total = round(time.monotonic() - start, 2)

    for r in results:
        if r["step"] in ("test", "acceptance"):
            r["failures"] = test_failures(r["output"])
            # A failing build is reported through the compiler's own file:line output.
            r["findings"] = findings("".join(f["output"] for f in r["failures"]))
        else:
            r["findings"] = findings(r["output"])

    for r in results:
        print(f"{'ok  ' if r['ok'] else 'FAIL'} {r['step']:<10} {r['seconds']:6.2f}s  {r['cmd']}")
    for r in results:
        if r["ok"]:
            continue
        print(f"\n=== {r['step']} failed")
        if r.get("failures"):
            for f in r["failures"]:
                print(f"--- {f['package']} {f['test']}\n{f['output'].rstrip()}")
        else:
            print(r["output"].rstrip())
    ok = all(r["ok"] for r in results)
    print(f"\n{name}: {'PASS' if ok else 'FAIL'} in {total}s")

    if args.json:
        report = {"ok": ok, "seconds": total, "steps": [{k: v for k, v in r.items() if k != "output"} for r in results]}
        with open(args.json, "w") as f:
            json.dump(report, f, indent=2)
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
