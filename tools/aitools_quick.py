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
against the merge base with origin/main, or main when there is no origin/main,
as tools/lintdiff.py computes it) are linted and tested too, so an edit to a shared package such as libs/telemetry runs that
package's tests. Beyond that there is no package selection: Go's build and test
caches re-run exactly the packages whose transitive inputs changed.

The unit tests run in a sanitized test environment: the host's coding-agent
variables and DATABRICKS_* variables are removed and HOME points at a dedicated
directory. This prevents agent-detection variables from changing aitools
behavior relative to CI. The acceptance tests run with the host environment:
the harness gives each test its own HOME and unsets the agent variables itself.

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

from lintdiff import changed_go_dirs, go_dirs

# hostenv's tests check that every aitools test package calls hostenv.Isolate.
PACKAGES = ["./cmd/aitools/...", "./libs/aitools/...", "./experimental/aitools/...", "./internal/testutil/hostenv/..."]

# Packages outside aitools that import it. cmd/root calls aitools from the root
# command's pre-run, so a change there can break every command. The fake agent
# CLIs of the acceptance tests write agent config through aitools. The
# test-exp-aitools task in Taskfile.yml runs a subset of IMPORTERS and ACCEPTANCE
# in CI. A new aitools importer goes in both places.
IMPORTERS = [
    ".",
    "./acceptance/internal/fakeagent",
    "./cmd",
    "./cmd/air",
    "./cmd/apps",
    "./cmd/experimental",
    "./cmd/root",
    "./cmd/version",
]

# Acceptance tests that run aitools: its own directories, plus the goldens that
# print aitools help or tips. The apps tests cover the `apps init` skills tip.
ACCEPTANCE = [
    "TestAccept/apps",
    "TestAccept/cmd/aitools",
    "TestAccept/experimental/aitools",
    "TestAccept/help$",
    "TestAccept/cmd/version",
    "TestAccept/quickstart",
    # All of air: besides its help goldens, image-push records the aitools install hint.
    "TestAccept/air",
]

# The agent variables aitools reads beyond the detection list in
# acceptance/acceptance_test.go: Claude Code's child-session marker, which gates
# the native hint, the agent session ids that scope the hint throttle, and the
# agent config-dir overrides.
AGENT_EXTRA_ENV = {
    "CLAUDE_CODE_CHILD_SESSION",
    "CLAUDE_CODE_SESSION_ID",
    "CLAUDE_CONFIG_DIR",
    "CODEX_HOME",
    "CODEX_THREAD_ID",
    "COPILOT_HOME",
    "GEMINI_CLI_HOME",
    "GOOSE_PATH_ROOT",
    "PI_CODING_AGENT_DIR",
    "XDG_CONFIG_HOME",
}

MODULE = "github.com/databricks/cli"

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


def agent_env(src):
    """Return the agent variables to remove: the ones the acceptance harness
    unsets (read from its source, src, so the two lists can't drift) plus
    AGENT_EXTRA_ENV.

    >>> src = '// Unset AI-agent detection env vars\\nfor _, v := range []string{\\n\\t"AGENT",\\n\\t"KIRO",\\n} {'
    >>> sorted(agent_env(src) - AGENT_EXTRA_ENV)
    ['AGENT', 'KIRO']
    """
    m = re.search(r"Unset AI-agent detection env vars.*?range \[\]string\{(?P<list>.*?)\}", src, re.DOTALL)
    if not m:
        raise SystemExit("aitools-quick: cannot find the agent variable list in acceptance/acceptance_test.go")
    return set(re.findall(r'"(\w+)"', m["list"])) | AGENT_EXTRA_ENV


def findings(output, pkg_dir=""):
    """Return file:line findings from compiler, vet, lint or test output.

    Test logs name only the file, so a bare file name is joined to pkg_dir.

    >>> findings("# pkg\\ncmd/aitools/x.go:3:5: undefined: y\\nok")
    [{'file': 'cmd/aitools/x.go', 'line': 3, 'col': 5, 'message': 'undefined: y'}]
    >>> findings("    hint_test.go:375: want x", "libs/aitools/agents")
    [{'file': 'libs/aitools/agents/hint_test.go', 'line': 375, 'col': None, 'message': 'want x'}]
    """
    result = []
    for line in output.splitlines():
        m = FINDING.match(line.strip())
        if m:
            col = int(m["col"]) if m["col"] else None
            file = m["file"] if "/" in m["file"] else os.path.join(pkg_dir, m["file"]).replace(os.sep, "/")
            result.append({"file": file, "line": int(m["line"]), "col": col, "message": m["msg"]})
    return result


def package_dir(pkg):
    """Return the repository directory of an import path.

    >>> package_dir("github.com/databricks/cli/libs/aitools/agents")
    'libs/aitools/agents'
    >>> package_dir("github.com/databricks/cli")
    ''
    """
    return pkg.removeprefix(MODULE).removeprefix("/")


def test_failures(output):
    """Return failing tests and their output from `go test -json` output.

    Build failures are reported as a pseudo-test named "(build)". A package that
    fails with no failing test (a timeout, a panic in init, a TestMain exit) is
    reported as a pseudo-test named "(package)" with the package's own output
    and that of its tests that never finished, where a timeout's panic lands.

    >>> out = '''{"Action":"output","Package":"p","Test":"TestOK","Output":"ok\\\\n"}
    ... {"Action":"pass","Package":"p","Test":"TestOK"}
    ... {"Action":"output","Package":"p","Test":"TestSlow","Output":"panic: test timed out\\\\n"}
    ... {"Action":"output","Package":"p","Output":"FAIL p\\\\n"}
    ... {"Action":"fail","Package":"p"}
    ... {"Action":"output","Package":"q","Test":"TestA","Output":"bad\\\\n"}
    ... {"Action":"fail","Package":"q","Test":"TestA"}
    ... {"Action":"fail","Package":"q"}'''
    >>> test_failures(out)
    [{'package': 'p', 'test': '(package)', 'output': 'panic: test timed out\\nFAIL p\\n'}, {'package': 'q', 'test': 'TestA', 'output': 'bad\\n'}]
    """
    logs = {}
    failed = []
    finished = set()
    for line in output.splitlines():
        try:
            ev = json.loads(line)
        except json.JSONDecodeError:
            continue
        action = ev.get("Action")
        if action in ("build-output", "build-fail"):
            key = (ev.get("ImportPath", ""), "(build)")
        else:
            key = (ev.get("Package", ""), ev.get("Test") or "(package)")
        if action in ("output", "build-output"):
            logs.setdefault(key, []).append(ev.get("Output", ""))
        elif action == "build-fail" or (action == "fail" and not ev.get("FailedBuild")):
            failed.append(key)
        if action in ("pass", "fail", "skip"):
            finished.add(key)
    # A failing subtest also fails its parent, and a failing test also fails its
    # package; report only the innermost test.
    names = {(p, t) for p, t in failed}
    leaves = [
        (p, t)
        for p, t in failed
        if not any(q == p and (u.startswith(t + "/") or (t == "(package)" and u != t)) for q, u in names)
    ]

    def output(p, t):
        if t != "(package)":
            return "".join(logs.get((p, t), []))
        keys = [k for k in logs if k[0] == p and k[1] != t and k not in finished] + [(p, t)]
        return "".join(line for k in keys for line in logs.get(k, []))

    return [{"package": p, "test": t, "output": output(p, t)} for p, t in leaves]


def select_packages(dirs):
    """Return the ./dir package patterns among dirs outside PACKAGES.

    The acceptance package runs the whole acceptance suite and integration tests
    need a workspace, so both are left to aitools-full; tools/ is its own module.

    >>> select_packages(["./libs/telemetry", "./cmd/aitools", "./acceptance", "./acceptance/internal",
    ...                  "./integration/cmd", "./tools/testmask"])
    ['./libs/telemetry']
    """
    return sorted(
        d
        for d in dirs
        if d.removeprefix("./").split("/")[0] not in ("acceptance", "integration", "tools")
        and not any((d + "/").startswith(p.removesuffix("...")) for p in PACKAGES)
    )


def default_ref():
    """origin/main when it exists: a local main is not moved by `git fetch origin main`."""
    found = subprocess.run(["git", "rev-parse", "--verify", "--quiet", "origin/main"], capture_output=True)
    return "origin/main" if found.returncode == 0 else "main"


def changed_packages(ref):
    """Packages with Go files changed since the merge base with ref, committed or not."""
    dirs = changed_go_dirs(ref)
    if dirs is None:
        print(f"aitools-quick: cannot diff against {ref}; checking the aitools packages only", file=sys.stderr)
        return []
    untracked = subprocess.run(["git", "ls-files", "--others", "--exclude-standard"], capture_output=True, text=True)
    return select_packages(set(dirs) | set(go_dirs(untracked.stdout.split(), True)))


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
    with open("acceptance/acceptance_test.go") as f:
        agent = agent_env(f.read())
    env = {k: v for k, v in os.environ.items() if k not in agent and not k.startswith("DATABRICKS_")}
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
    parser.add_argument(
        "--ref", help="ref whose merge base defines the changed packages (default: origin/main, or main without it)"
    )
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

    packages = PACKAGES + changed_packages(args.ref or default_ref())
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
        # The same timeout as the Taskfile's acceptance tasks.
        timeout = "-timeout=" + os.environ.get("LOCAL_TIMEOUT", "60m")
        steps = [
            ("lint", lint, None),
            ("test", ["go", "test", "-json", timeout, *packages, *IMPORTERS], hermetic_env(home)),
            # The harness isolates each test itself, and it locates uv's cache
            # and managed Pythons from the real HOME.
            ("acceptance", ["go", "test", "-json", timeout, "./acceptance", "-run", "|".join(ACCEPTANCE)], None),
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
            r["findings"] = [x for f in r["failures"] for x in findings(f["output"], package_dir(f["package"]))]
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
