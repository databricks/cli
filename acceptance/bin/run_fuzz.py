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
             failed the invariant

Every seed adds a line to LOG.summary. A bug or a hang also writes a ready-to-run repro to
LOG.repro and exits non-zero. Nothing is written to stdout: the committed run asserts empty output.

FUZZ_TARGET and FUZZ_MODE come from the test.toml matrix; FUZZ_SEED_START, FUZZ_SEED_COUNT,
FUZZ_SEED_TIMEOUT and FUZZ_TIME_BUDGET are optional knobs the caller sets (see task test-fuzz).
FUZZ_CHECK_DRIFT is read only to name the oracle in the repro; script.prepare acts on it.
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
# progressing variant isn't force-killed at the per-script Timeout and read as a failure. 900s
# leaves margin under the 20m test.toml Timeout. Set FUZZ_TIME_BUDGET=0 to disable.
BUDGET = float(os.environ.get("FUZZ_TIME_BUDGET", "900"))

# Grace period between SIGQUIT and the SIGKILL backstop.
QUIT_GRACE = 10

TARGET = os.environ["FUZZ_TARGET"]
MODE = os.environ["FUZZ_MODE"]

# Which no-drift oracle script.prepare installed. Part of the repro because the two disagree: the
# committed run leaves this at 0 and gets the plan-determinism diff, while task test-fuzz defaults
# it to 1 and gets the exact check.
CHECK_DRIFT = os.environ.get("FUZZ_CHECK_DRIFT", "0")

POSIX = os.name == "posix"

# Resolved rather than passed as a bare name: on Windows, CreateProcess searches System32 first,
# where "bash" is the WSL launcher stub, which exits non-zero with no distribution installed and
# makes every seed read as rejected. shutil.which searches PATH, finding the bash we run under.
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


def oracle_verdict(seed_dir):
    """The no-drift oracle's own verdict, if it reached one. Empty if it never ran or was happy."""
    # Each oracle reports a violation in a form only it produces, so a seed that broke the
    # invariant is still recognisable when it also touched a route the testserver lacks.
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
    gen_err = read(seed_dir / "LOG.gen.err")
    if gen_err:
        first_line = gen_err.splitlines()[0].decode(errors="replace")
        return "bug", f"could not be generated: {first_line}"

    logs = concat_logs(seed_dir)

    # A panic or internal error anywhere is a bug even if the CLI then rejects the config.
    if b"panic:" in logs or b"internal error" in logs:
        return "bug", "panicked or hit an internal error"

    # Before the gap marker: the cleanup destroy runs on every seed, so an unmodeled delete route
    # puts that marker in the logs of seeds whose invariant genuinely failed.
    verdict = oracle_verdict(seed_dir)
    if verdict:
        return "bug", verdict

    # Marker body of the catch-all stubs in fuzz/test.toml. A gap seed has usually deployed first,
    # so this has to come before the INPUT_CONFIG_OK check below.
    if b"TESTSERVER_GAP" in logs:
        return "gap", ""

    # The oracle above names a drift failure; anything else here is a command that failed on a
    # config the CLI had accepted. Failing before the marker just means it was rejected.
    if b"INPUT_CONFIG_OK" in read(seed_dir / "LOG.check"):
        return "bug", "failed after deploying; see the seed's LOG.* files"

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
        f"FUZZ_SEED_COUNT=1 FUZZ_TARGET={TARGET} FUZZ_MODE={MODE} "
        f"FUZZ_CHECK_DRIFT={CHECK_DRIFT} task test-fuzz\n"
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
