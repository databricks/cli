#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""
Re-measure `./task aitools-quick`: its wall time after an edit, and how many of
the defects the full tier (`./task aitools-full`) catches it also catches.

    time    Simulate an edit (append an unexported function to one production
            file per aitools package), run aitools-quick, restore the file.
            Reports p50/p95 wall time; --budget fails the run when p50 is over it.
    detect  Apply source mutants (flipped comparisons and booleans, swallowed
            errors, removed calls, changed strings) one at a time, run the
            quick tier on each and the full tier on the ones it misses, and
            report the quick tier's share of the full tier's kills. Slow: about
            a minute per mutant.

Both modes edit files, so they run in a temporary git worktree of HEAD with the
checkout's uncommitted and untracked files copied in; the checkout itself is
never written to.

Usage:
    ./task aitools-quick-bench -- time --rounds 10 --budget 10
    ./task aitools-quick-bench -- time --cold
    ./task aitools-quick-bench -- detect --sample 20 --seed 1
"""

import argparse
import json
import os
import random
import re
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import uuid

QUICK = ["uv", "run", "--script", "tools/aitools_quick.py"]
FULL = [*QUICK, "--full"]

# One production file per aitools package: an edit there invalidates that
# package and everything that imports it.
EDIT_TARGETS = [
    "cmd/aitools/version.go",
    "libs/aitools/agents/agents.go",
    "libs/aitools/autoinstall/state.go",
    "libs/aitools/installer/state.go",
    "libs/aitools/mcp/state.go",
    "libs/aitools/mcp/agentconfig/writer.go",
    "libs/aitools/mcp/proxy/run.go",
    "libs/aitools/mcpconfig/spec.go",
    "experimental/aitools/cmd/aitools.go",
    "experimental/aitools/lib/middlewares/warehouse.go",
    "experimental/aitools/lib/session/session.go",
]

MUTANT_DIRS = ["cmd/aitools", "libs/aitools", "experimental/aitools"]


def percentile(values, p):
    """Nearest-rank percentile.

    >>> percentile([5, 1, 3, 2, 4], 50)
    3
    >>> percentile([5, 1, 3, 2, 4], 95)
    5
    """
    ordered = sorted(values)
    rank = max(1, -(-p * len(ordered) // 100))
    return ordered[rank - 1]


def run_quick(env=None, steps=None):
    """Run aitools-quick once; when steps is a dict, add each step's seconds to it."""
    with tempfile.TemporaryDirectory() as tmp:
        report = os.path.join(tmp, "report.json")
        start = time.monotonic()
        proc = subprocess.run([*QUICK, "--json", report], capture_output=True, text=True, env=env)
        seconds = time.monotonic() - start
        if steps is not None and os.path.exists(report):
            with open(report) as f:
                for step in json.load(f)["steps"]:
                    steps.setdefault(step["step"], []).append(step["seconds"])
    return seconds, proc.returncode == 0, proc.stdout + proc.stderr


def simulate_edit(path):
    """Append code that changes the package's compiled output, as a body edit would.

    The value differs on every call and a global store in init() can't be
    optimized away, so the package and every test binary that links it are
    rebuilt and re-run. The package's export data does not change, so
    importers are relinked but not recompiled. The self-read keeps the unused
    linter quiet.
    """
    with open(path, "a") as f:
        f.write(
            "\nvar aitoolsQuickBenchSink string\n\n"
            f'func init() {{\n\taitoolsQuickBenchSink = "{uuid.uuid4().hex}"\n\t_ = aitoolsQuickBenchSink\n}}\n'
        )


def cmd_time(args):
    env = None
    cache = None
    if args.cold:
        cache = tempfile.mkdtemp(prefix="aitools-quick-cold-")
        env = dict(os.environ, GOCACHE=os.path.join(cache, "go"), GOLANGCI_LINT_CACHE=os.path.join(cache, "lint"))
    try:
        # Warm-up: the first run after a checkout or a cache wipe is the cold one.
        seconds, ok, out = run_quick(env)
        print(f"{'cold' if args.cold else 'warm-up'} run: {seconds:.2f}s ok={ok}")
        if not ok:
            print(out)
            return 1
        if args.cold:
            return 0
        times = {t: [] for t in EDIT_TARGETS}
        steps = {}
        for i in range(args.rounds):
            for target in EDIT_TARGETS:
                with open(target) as f:
                    original = f.read()
                try:
                    simulate_edit(target)
                    seconds, ok, out = run_quick(steps=steps)
                finally:
                    with open(target, "w") as f:
                        f.write(original)
                if not ok:
                    print(f"aitools-quick failed after editing {target}:\n{out}")
                    return 1
                times[target].append(seconds)
                print(f"round {i + 1} {target}: {seconds:.2f}s", flush=True)
    finally:
        if cache:
            shutil.rmtree(cache, ignore_errors=True)

    every = [s for v in times.values() for s in v]
    print(f"\n{'edited file':<52} {'p50':>6} {'p95':>6} {'max':>6}")
    for target, v in times.items():
        print(f"{target:<52} {percentile(v, 50):6.2f} {percentile(v, 95):6.2f} {max(v):6.2f}")
    p50, p95 = percentile(every, 50), percentile(every, 95)
    print(f"{'all':<52} {p50:6.2f} {p95:6.2f} {max(every):6.2f}  (n={len(every)})")
    print(f"\n{'step (runs in parallel)':<52} {'p50':>6} {'p95':>6}")
    for step, v in steps.items():
        print(f"{step:<52} {percentile(v, 50):6.2f} {percentile(v, 95):6.2f}")
    if args.json:
        with open(args.json, "w") as f:
            json.dump({"p50": p50, "p95": p95, "runs": times, "steps": steps}, f, indent=2)
    if args.budget and p50 > args.budget:
        print(f"FAIL: p50 {p50:.2f}s is over the {args.budget}s budget")
        return 1
    return 0


def mask_strings(line):
    """Blank out string literals and trailing comments so operators inside them are skipped.

    >>> mask_strings('a == "x == y" // b == c')
    'a == "______"'
    """
    out = re.sub(r'"(?:[^"\\]|\\.)*"', lambda m: '"' + "_" * (len(m.group()) - 2) + '"', line)
    out = re.sub(r"`[^`]*`", lambda m: "`" + "_" * (len(m.group()) - 2) + "`", out)
    i = out.find("//")
    return out if i < 0 else out[:i].rstrip()


# (name, pattern on the masked line, replacement for the matched text)
OPERATORS = [
    ("flip ==", r"==", "!="),
    ("flip !=", r"!=", "=="),
    ("flip &&", r"&&", "||"),
    ("flip ||", r"\|\|", "&&"),
    ("flip true", r"\btrue\b", "false"),
    ("flip false", r"\bfalse\b", "true"),
    ("swallow error", r"\berr$", "nil"),
]


def mutants(seed, sample):
    found = []
    for root in MUTANT_DIRS:
        for dirpath, dirnames, files in os.walk(root):
            # Walk in a fixed order so a seed picks the same mutants on every machine.
            dirnames.sort()
            for name in sorted(files):
                if not name.endswith(".go") or name.endswith("_test.go"):
                    continue
                path = os.path.join(dirpath, name)
                with open(path) as f:
                    lines = f.read().split("\n")
                in_import = False
                for n, line in enumerate(lines):
                    masked = mask_strings(line)
                    stripped = masked.strip()
                    # Mutants of import paths never compile.
                    if stripped == "import (":
                        in_import = True
                    if in_import:
                        in_import = stripped != ")"
                        continue
                    if not stripped or stripped.startswith(("import", "package")):
                        continue
                    for op, pattern, repl in OPERATORS:
                        if op == "swallow error" and not stripped.startswith("return"):
                            continue
                        for m in re.finditer(pattern, masked):
                            found.append((path, n, m.start(), m.end(), repl, op))
                    # Drop a call statement such as cmd.AddCommand(x) or a flag registration.
                    if re.fullmatch(r"[\w.]+\(.*\)", stripped) and not stripped.startswith(("return", "defer", "go ")):
                        found.append((path, n, 0, len(line), "", "remove call"))
                    s = re.search(r'"[^"]+"', masked)
                    if s:
                        found.append((path, n, s.start() + 1, s.start() + 1, "x", "change string"))
    random.Random(seed).shuffle(found)
    return found[:sample]


def apply_mutant(path, n, start, end, repl):
    with open(path) as f:
        original = f.read()
    lines = original.split("\n")
    lines[n] = lines[n][:start] + repl + lines[n][end:]
    with open(path, "w") as f:
        f.write("\n".join(lines))
    return original


def cmd_detect(args):
    for name, cmd in (("aitools-quick", QUICK), ("aitools-full", FULL)):
        if subprocess.run(cmd, capture_output=True).returncode != 0:
            print(f"{name} fails on the unmodified tree; fix that first")
            return 1
    rows = []
    for path, n, start, end, repl, op in mutants(args.seed, args.sample):
        original = apply_mutant(path, n, start, end, repl)
        try:
            if (
                subprocess.run(
                    ["go", "build", "./cmd/aitools/...", "./libs/aitools/...", "./experimental/aitools/..."],
                    capture_output=True,
                ).returncode
                != 0
            ):
                continue  # Does not compile: both tiers catch it trivially.
            quick = subprocess.run(QUICK, capture_output=True).returncode != 0
            # The full tier runs everything the quick tier does, so it kills every mutant quick kills.
            full = quick or subprocess.run(FULL, capture_output=True).returncode != 0
        finally:
            with open(path, "w") as f:
                f.write(original)
        rows.append({"file": path, "line": n + 1, "op": op, "quick": quick, "full": full})
        print(
            f"{path}:{n + 1} {op:<14} quick={'killed' if quick else 'missed'} full={'killed' if full else 'missed'}",
            flush=True,
        )
    full_kills = [r for r in rows if r["full"]]
    both = [r for r in full_kills if r["quick"]]
    print(f"\ncompiling mutants: {len(rows)}; killed by full: {len(full_kills)}; also killed by quick: {len(both)}")
    if full_kills:
        print(f"detection: {100 * len(both) / len(full_kills):.0f}%")
    if args.json:
        with open(args.json, "w") as f:
            json.dump(rows, f, indent=2)
    return 0


def isolated_copy(root, copy):
    """Create a worktree of HEAD at copy with the uncommitted and untracked files of the checkout at root."""
    subprocess.run(["git", "-C", root, "worktree", "add", "--quiet", "--detach", copy, "HEAD"], check=True)
    # Pin the diff format: user config such as diff.noprefix or color.diff=always would break git apply.
    diff = subprocess.run(
        [
            "git",
            "-C",
            root,
            "diff",
            "--binary",
            "--no-color",
            "--no-ext-diff",
            "--src-prefix=a/",
            "--dst-prefix=b/",
            "HEAD",
        ],
        capture_output=True,
        check=True,
    ).stdout
    if diff:
        subprocess.run(["git", "-C", copy, "apply", "--binary"], input=diff, check=True)
    untracked = subprocess.run(
        ["git", "-C", root, "ls-files", "-z", "--others", "--exclude-standard"], capture_output=True, check=True
    )
    for path in untracked.stdout.decode().split("\0"):
        if path:
            os.makedirs(os.path.join(copy, os.path.dirname(path)), exist_ok=True)
            shutil.copy2(os.path.join(root, path), os.path.join(copy, path), follow_symlinks=False)


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="cmd", required=True)
    t = sub.add_parser("time", help="measure wall time after simulated edits")
    t.add_argument("--rounds", type=int, default=3)
    t.add_argument("--budget", type=float, help="fail when the p50 wall time is over this many seconds")
    t.add_argument("--cold", action="store_true", help="measure one run with empty Go and lint caches")
    t.add_argument("--json", metavar="PATH")
    d = sub.add_parser("detect", help="measure detection against the full tier")
    d.add_argument("--sample", type=int, default=20)
    d.add_argument("--seed", type=int, default=1)
    d.add_argument("--json", metavar="PATH")
    args = parser.parse_args()
    if args.cmd == "time" and args.rounds < 1:
        parser.error("--rounds must be at least 1")
    if args.cmd == "time" and args.cold and (args.json or args.budget):
        parser.error("--json and --budget do not apply to --cold, which measures a single run")
    if args.json:
        args.json = os.path.abspath(args.json)

    # Run the cleanup below on SIGTERM too, not only on Ctrl-C.
    signal.signal(signal.SIGTERM, lambda *_: sys.exit(143))
    root = subprocess.run(["git", "rev-parse", "--show-toplevel"], capture_output=True, text=True, check=True)
    root = root.stdout.strip()
    tmp = tempfile.mkdtemp(prefix="aitools-quick-bench-")
    copy = os.path.join(tmp, "src")
    try:
        isolated_copy(root, copy)
        os.chdir(copy)
        return cmd_time(args) if args.cmd == "time" else cmd_detect(args)
    finally:
        os.chdir(root)
        subprocess.run(["git", "worktree", "remove", "--force", copy], capture_output=True)
        shutil.rmtree(tmp, ignore_errors=True)


if __name__ == "__main__":
    sys.exit(main())
