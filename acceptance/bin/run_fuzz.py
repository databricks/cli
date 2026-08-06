#!/usr/bin/env python3
"""
Seed loop for the invariant fuzzer. Runs one seed per iteration by calling the seed_body bash
function that acceptance/bundle/fuzz/script exports, which sources the invariant target picked by
FUZZ_TARGET, and classifies each outcome:

  deployed - the config deployed and the invariant held
  rejected - the CLI refused the config before deploying it; the common case, not a bug
  gap      - the config needs a route the testserver does not model
  hang     - the seed outlived FUZZ_SEED_TIMEOUT
  bug      - a panic, an internal error, a generator failure, or a config that deployed and then
             broke the invariant or failed a later command

Every seed adds a line to LOG.summary. A bug or a hang also writes a ready-to-run repro to
LOG.repro and exits non-zero. Nothing is written to stdout: the committed run asserts empty output.

FUZZ_TARGET and FUZZ_MODE come from the test.toml matrix; FUZZ_SEED_START, FUZZ_SEED_COUNT,
FUZZ_SEED_TIMEOUT and FUZZ_TIME_BUDGET are optional knobs the caller sets (see task test-fuzz).
FUZZ_CHECK_DRIFT is read only to name the oracle in the repro; script.prepare acts on it.
"""

import os
import re
import shutil
import signal
import subprocess
import sys
import time
from collections import Counter
from pathlib import Path

# Per-seed cap: a seed past this budget is stuck, not slow. Set FUZZ_SEED_TIMEOUT=0 to disable.
SEED_TIMEOUT = float(os.environ.get("FUZZ_SEED_TIMEOUT", "180"))

# Overall budget (seconds): stop starting seeds past it, so a slow but progressing variant exits
# cleanly instead of being force-killed at the 20m test.toml Timeout. 0 disables.
BUDGET = float(os.environ.get("FUZZ_TIME_BUDGET", "900"))

# Seconds between SIGQUIT and the SIGKILL backstop.
QUIT_GRACE = 10

# Log of the destroy in invariant_cleanup, which every target runs from an EXIT trap.
CLEANUP_LOG = "LOG.destroy"

TARGET = os.environ["FUZZ_TARGET"]
MODE = os.environ["FUZZ_MODE"]

# Which no-drift oracle script.prepare installed, and part of the repro because the two disagree:
# 0 is the plan-determinism diff, 1 the exact check that task test-fuzz defaults to.
CHECK_DRIFT = os.environ.get("FUZZ_CHECK_DRIFT", "0")

POSIX = os.name == "posix"

# Resolved, not a bare name: on Windows CreateProcess finds the System32 WSL stub first, which
# exits non-zero with no distribution installed and makes every seed read as rejected.
BASH = shutil.which("bash")


def read(path):
    """Log contents as bytes; a fuzzed config can put arbitrary bytes in there. Empty if absent."""
    return path.read_bytes() if path.exists() else b""


def concat_logs(seed_dir, skip=()):
    return b"".join(read(p) for p in sorted(seed_dir.glob("LOG.*")) if p.name not in skip)


def killpg(proc, sig):
    try:
        os.killpg(proc.pid, sig)
    except ProcessLookupError:
        # The seed can exit on its own between the timeout and the signal; it is still a hang.
        pass


def kill_seed(proc):
    if not POSIX:
        # Windows has neither SIGQUIT nor the process group below, so this is all it can do.
        proc.kill()
        return
    # SIGQUIT first for Go's goroutine dump, then SIGKILL as a backstop.
    killpg(proc, signal.SIGQUIT)
    try:
        proc.wait(timeout=QUIT_GRACE)
    except subprocess.TimeoutExpired:
        killpg(proc, signal.SIGKILL)


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


def oracle_verdict(seed_dir):
    """The no-drift oracle's own verdict, if it reached one. Empty if it never ran or was happy."""
    # Each oracle reports in a form only it produces, so a drift verdict survives a testserver gap.
    if b"Unexpected action=" in read(seed_dir / "LOG.check"):
        # verify_no_drift.py, the exact check shared with the curated invariant targets.
        return "planned a change after deploy"
    if read(seed_dir / "LOG.plan.determinism.diff").strip():
        # The plan-determinism diff script.prepare substitutes when FUZZ_CHECK_DRIFT is 0.
        return "planned differently on two consecutive runs"
    if read(seed_dir / "LOG.plan.failed").strip():
        # Same substitute, when the plan failed outright for a reason that is not a testserver gap.
        return "could not be planned after deploy"
    return ""


def classify(seed_dir):
    """Classify a seed that exited non-zero. Returns its kind and, for a failure, the reason."""
    # The generator only writes to stderr when it fails: our bug, not a rejected config.
    gen_err = read(seed_dir / "LOG.gen.err").strip()
    if gen_err:
        # Last line: a traceback's first one is always "Traceback (most recent call last):".
        last_line = gen_err.splitlines()[-1].decode(errors="replace")
        return "bug", f"could not be generated: {last_line}"

    # A panic or internal error anywhere is a bug even if the CLI then rejects the config.
    logs = concat_logs(seed_dir)
    if b"panic:" in logs or b"internal error" in logs:
        return "bug", "panicked or hit an internal error"

    # Before the gap marker: a seed can do both, and the drift verdict is the more specific.
    verdict = oracle_verdict(seed_dir)
    if verdict:
        return "bug", verdict

    # Marker from the catch-all stubs in fuzz/test.toml. A gap after the deploy is still a gap, so
    # this precedes INPUT_CONFIG_OK; the cleanup log is skipped, as it only runs after a failure.
    if b"TESTSERVER_GAP" in concat_logs(seed_dir, skip={CLEANUP_LOG}):
        return "gap", ""

    # Past the marker the CLI had accepted the config, so a failure here is not a rejection.
    if b"INPUT_CONFIG_OK" in read(seed_dir / "LOG.check"):
        return "bug", "failed after deploying; see the seed's LOG.* files"

    return "rejected", ""


def resource_type(seed_dir):
    """The resource type the seed's config declares, so a window shows which types it covered."""
    match = re.search(rb"^resources:\n  (\S+):", read(seed_dir / "LOG.config"), re.MULTILINE)
    return match.group(1).decode() if match else "unknown"


def record(kind, seed, seed_dir):
    """One machine-readable line per seed. To a file, not stdout, so empty output still holds."""
    with open("LOG.summary", "a") as f:
        f.write(f"{kind} seed={seed} target={TARGET} mode={MODE} type={resource_type(seed_dir)}\n")


def fail(seed, seed_dir, kind, reason, prefix=""):
    record(kind, seed, seed_dir)
    # To a file, because the harness rewrites env-var values in stdout. Target and mode go through
    # ENVFILTER: as EnvMatrix keys, plain env vars would be overridden and re-run every variant.
    Path("LOG.repro").write_text(
        f"fuzz: seed {seed} {reason}, reproduce with: {prefix}"
        f"ENVFILTER=FUZZ_TARGET={TARGET},FUZZ_MODE={MODE} FUZZ_SEED_START={seed} "
        f"FUZZ_SEED_COUNT=1 FUZZ_CHECK_DRIFT={CHECK_DRIFT} task test-fuzz\n"
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
            record("deployed", seed, seed_dir)
            continue

        # A seed that had to be killed hung, which is distinct from a drift bug.
        if killed:
            fail(seed, seed_dir, "hang", f"hung (>{SEED_TIMEOUT:g}s)", "FUZZ_SEED_TIMEOUT=0 ")

        kind, reason = classify(seed_dir)
        if reason:
            fail(seed, seed_dir, kind, reason)
        record(kind, seed, seed_dir)

    kinds = totals()

    # Nothing deploying is not a pass: a broken schema, generator or fixture looks exactly like the
    # CLI correctly rejecting random input. A single-seed replay is exempt.
    if count > 1 and not kinds["deployed"]:
        sys.exit("fuzz: no seed deployed; the schema, generator or fixtures are broken")


if __name__ == "__main__":
    main()
