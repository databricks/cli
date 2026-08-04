#!/usr/bin/env python3
"""
Seed loop for the invariant fuzzer. Runs one seed per iteration by calling the seed_body bash
function that acceptance/bundle/fuzz/script exports, which sources the invariant target picked by
FUZZ_TARGET, and classifies each outcome:

  deployed - the config deployed and the invariant held
  rejected - the CLI refused the config before deploying it; the common case, not a bug
  gap      - the config needs a route the testserver does not model
  hang     - the seed outlived FUZZ_SEED_TIMEOUT
  bug      - a panic, an internal error, a generator failure, or a broken invariant

Every seed adds a line to LOG.summary. A bug or a hang also writes a ready-to-run repro to
LOG.repro and exits non-zero. Nothing is written to stdout: the committed run asserts empty output.

FUZZ_TARGET and FUZZ_MODE come from the test.toml matrix; FUZZ_SEED_START, FUZZ_SEED_COUNT,
FUZZ_SEED_TIMEOUT and FUZZ_TIME_BUDGET are optional knobs the caller sets (see task test-fuzz).
"""

import os
import shutil
import signal
import subprocess
import sys
import time
from collections import Counter
from pathlib import Path

# Per-seed cap: a seed past this budget is stuck, not slow. Set FUZZ_SEED_TIMEOUT=0 to disable.
SEED_TIMEOUT = float(os.environ.get("FUZZ_SEED_TIMEOUT", "180"))

# Overall budget (seconds): stop starting new seeds past it and exit cleanly, so a slow-but-
# progressing variant isn't force-killed at the per-script Timeout and read as a failure. Measured
# from this script rather than from fuzz/script, so it excludes the one-off schema dump. 900s
# leaves margin under the 20m test.toml Timeout. Set FUZZ_TIME_BUDGET=0 to disable.
BUDGET = float(os.environ.get("FUZZ_TIME_BUDGET", "900"))

# Grace period between the SIGQUIT that asks Go for a goroutine dump and the SIGKILL backstop.
QUIT_GRACE = 10

TARGET = os.environ["FUZZ_TARGET"]
MODE = os.environ["FUZZ_MODE"]

POSIX = os.name == "posix"

# Resolved rather than passed as a bare name: on Windows, CreateProcess searches System32 before
# PATH, so "bash" there is the WSL launcher stub, which exits non-zero with no distribution
# installed and every seed reads as rejected. shutil.which searches PATH only, so it finds the same
# bash the harness runs this script under.
BASH = shutil.which("bash")


def read(path):
    """Log contents as bytes; a fuzzed config can put arbitrary bytes in there. Empty if absent."""
    return path.read_bytes() if path.exists() else b""


def concat_logs(seed_dir):
    return b"".join(read(p) for p in sorted(seed_dir.glob("LOG.*")))


def kill_seed(proc):
    if not POSIX:
        # Windows has neither SIGQUIT nor the process group below, so this is all it can do.
        proc.kill()
        return
    # SIGQUIT first for Go's goroutine dump, then SIGKILL as a backstop.
    os.killpg(proc.pid, signal.SIGQUIT)
    try:
        proc.wait(timeout=QUIT_GRACE)
    except subprocess.TimeoutExpired:
        os.killpg(proc.pid, signal.SIGKILL)


def run_seed(seed_dir, seed):
    """Run one seed in a fresh bash. Returns its exit code and whether it had to be killed."""
    with open(seed_dir / "LOG.check", "wb") as log:
        proc = subprocess.Popen(
            [BASH, "-euo", "pipefail", "-c", 'seed_body "$@"', "_", str(seed_dir), str(seed)],
            stdout=log,
            stderr=subprocess.STDOUT,
            # Own process group, so killing a hung seed also takes down the CLI it is waiting on.
            start_new_session=POSIX,
        )
        try:
            return proc.wait(timeout=SEED_TIMEOUT or None), False
        except subprocess.TimeoutExpired:
            kill_seed(proc)
            return proc.wait(), True


def classify(seed_dir):
    """Classify a seed that exited non-zero. Returns its kind and, for a failure, the reason."""
    # The generator only writes to stderr when it fails: our bug, not a rejected config.
    gen_err = read(seed_dir / "LOG.gen.err")
    if gen_err:
        first_line = gen_err.splitlines()[0].decode(errors="replace")
        return "bug", f"could not be generated: {first_line}"

    logs = concat_logs(seed_dir)

    # A panic or internal error anywhere is a bug even if the CLI then rejects the config.
    if b"panic:" in logs or b"internal error" in logs:
        return "bug", "panicked or hit an internal error"

    # Marker body of the catch-all stubs in fuzz/test.toml: the route is one the testserver does
    # not model. Checked before the marker below, else an unmodeled route reached during plan or
    # destroy reads as a drift bug.
    if b"TESTSERVER_GAP" in logs:
        return "gap", ""

    # Failing after INPUT_CONFIG_OK means the config deployed but drifted (or destroy failed);
    # failing before it with no panic just means the config was rejected.
    if b"INPUT_CONFIG_OK" in read(seed_dir / "LOG.check"):
        return "bug", "broke the invariant"

    return "rejected", ""


def record(kind, seed):
    """One machine-readable line per seed. To a file, not stdout, so empty output still holds."""
    with open("LOG.summary", "a") as f:
        f.write(f"{kind} seed={seed} target={TARGET} mode={MODE}\n")


def fail(seed, kind, reason, prefix=""):
    record(kind, seed)
    # The repro goes to a file because the harness rewrites env-var values in stdout.
    Path("LOG.repro").write_text(
        f"fuzz: seed {seed} {reason}, reproduce with: {prefix}FUZZ_SEED_START={seed} "
        f"FUZZ_SEED_COUNT=1 FUZZ_TARGET={TARGET} FUZZ_MODE={MODE} task test-fuzz\n"
    )
    sys.exit(1)


def totals():
    """Per-variant tally for triage. Reached only on a clean run; a bug or hang exits above."""
    summary = Path("LOG.summary")
    if not summary.exists():
        return Counter()

    # Count before appending the header, else it would count itself.
    kinds = Counter(line.split()[0] for line in summary.read_text().splitlines())
    with summary.open("a") as f:
        f.write("--- totals ---\n")
        for kind, n in sorted(kinds.items()):
            f.write(f"{n} {kind}\n")
    return kinds


def main():
    start = time.monotonic()
    seed_start = int(os.environ.get("FUZZ_SEED_START", "0"))
    count = int(os.environ.get("FUZZ_SEED_COUNT", "5"))

    for offset in range(count):
        # A clean stop, not a failure, so log to a file.
        if BUDGET and time.monotonic() - start >= BUDGET:
            Path("LOG.budget").write_text(
                f"fuzz: stopping after {offset}/{count} seeds; hit FUZZ_TIME_BUDGET={BUDGET:g}s\n"
            )
            break

        seed = seed_start + offset
        seed_dir = Path(f"seed-{seed}")
        seed_dir.mkdir(exist_ok=True)

        returncode, killed = run_seed(seed_dir, seed)
        if returncode == 0:
            record("deployed", seed)
            continue

        # A seed that had to be killed hung, which is distinct from a drift bug.
        if killed:
            fail(seed, "hang", f"hung (>{SEED_TIMEOUT:g}s)", "FUZZ_SEED_TIMEOUT=0 ")

        kind, reason = classify(seed_dir)
        if reason:
            fail(seed, kind, reason)
        record(kind, seed)

    kinds = totals()

    # Nothing deploying is not a pass: it means the schema, generator or fixtures are broken, which
    # otherwise looks just like the CLI correctly rejecting random input. A single-seed replay is
    # exempt, where one rejected config is a normal outcome.
    if count > 1 and not kinds["deployed"]:
        sys.exit("fuzz: no seed deployed; the schema, generator or fixtures are broken")


if __name__ == "__main__":
    main()
