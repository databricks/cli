#!/usr/bin/env python3
"""
Seed loop for the invariant fuzzer. Invokes fuzz/seed.sh per seed and classifies each:

  deployed - deployed and the invariant held
  rejected - validate --strict refused the config before deploy
  gap      - needs a route the testserver does not model
  hang     - exceeded FUZZ_SEED_TIMEOUT
  bug      - panic, internal error, mutator failure, or a deploy that failed or drifted

Writes LOG.summary per seed; on bug/hang writes LOG.repro and exits non-zero.
Stdout stays empty (the committed run asserts that).

Knobs: FUZZ_TARGET (matrix), FUZZ_SEED_*, FUZZ_TIME_BUDGET, FUZZ_CHECK_DRIFT
(script.prepare acts on the latter).
"""

import json
import os
import shutil
import signal
import subprocess
import sys
import time
from collections import Counter
from pathlib import Path

# Stuck-seed cap; FUZZ_SEED_TIMEOUT=0 disables.
SEED_TIMEOUT = float(os.environ.get("FUZZ_SEED_TIMEOUT", "180"))

# Nightly/task stop; seed count is only a ceiling. 0 disables.
BUDGET = float(os.environ.get("FUZZ_TIME_BUDGET", "900"))

QUIT_GRACE = 10  # seconds between SIGQUIT and SIGKILL

CLEANUP_LOG = "LOG.destroy"  # EXIT-trap destroy; every target writes this

TARGET = os.environ["FUZZ_TARGET"]

# Repro knob: 0 = plan-determinism, 1 = exact no_drift (task test-fuzz default).
CHECK_DRIFT = os.environ.get("FUZZ_CHECK_DRIFT", "0")

POSIX = os.name == "posix"

# Resolved path: Windows CreateProcess otherwise picks System32\bash.exe (WSL stub).
BASH = shutil.which("bash")
if not BASH:
    sys.exit("fuzz: bash not found on PATH")


def read(path):
    """Bytes from a log; fuzzed configs can put arbitrary bytes there. Empty if absent."""
    return path.read_bytes() if path.exists() else b""


def concat_logs(seed_dir, skip=()):
    return b"".join(read(p) for p in sorted(seed_dir.glob("LOG.*")) if p.name not in skip)


def killpg(proc, sig):
    try:
        os.killpg(proc.pid, sig)
    except ProcessLookupError:
        # Exited between timeout and signal; still report as hang.
        pass


def kill_seed(proc):
    if not POSIX:
        proc.kill()
        return
    # SIGQUIT for Go's goroutine dump, then SIGKILL.
    killpg(proc, signal.SIGQUIT)
    try:
        proc.wait(timeout=QUIT_GRACE)
    except subprocess.TimeoutExpired:
        killpg(proc, signal.SIGKILL)


def run_seed(seed_dir, seed):
    """Exit code and whether the seed was killed for timeout."""
    seed_sh = Path(os.environ["TESTDIR"]) / "seed.sh"
    with open(seed_dir / "LOG.check", "wb") as log:
        proc = subprocess.Popen(
            [BASH, "-euo", "pipefail", str(seed_sh), str(seed_dir), str(seed)],
            stdout=log,
            stderr=subprocess.STDOUT,
            # Own process group so a hung CLI dies with the seed.
            start_new_session=POSIX,
        )
        try:
            return proc.wait(timeout=SEED_TIMEOUT or None), False
        except subprocess.TimeoutExpired:
            kill_seed(proc)
            return proc.wait(), True


def oracle_verdict(seed_dir):
    """Drift oracle wording if it fired; empty if it never ran or was happy."""
    # Checked before TESTSERVER_GAP: both can fire on one seed.
    if b"Unexpected action=" in read(seed_dir / "LOG.check"):
        return "planned a change after deploy"
    if read(seed_dir / "LOG.plan.determinism.diff").strip():
        return "planned differently on two consecutive runs"
    if read(seed_dir / "LOG.plan.failed").strip():
        return "could not be planned after deploy"
    return ""


def classify(seed_dir):
    """Kind and, for a failure, the reason."""
    gen_err = read(seed_dir / "LOG.gen.err").strip()
    if gen_err:
        last_line = gen_err.splitlines()[-1].decode(errors="replace")
        return "bug", f"could not be mutated: {last_line}"

    # Mutated config text must not count as a CLI panic/gap.
    skip_input = {"LOG.config"}
    logs = concat_logs(seed_dir, skip=skip_input)
    if b"panic:" in logs or b"internal error" in logs:
        return "bug", "panicked or hit an internal error"

    verdict = oracle_verdict(seed_dir)
    if verdict:
        return "bug", verdict

    # Skip cleanup: destroy after failure must not mask the real cause.
    if b"TESTSERVER_GAP" in concat_logs(seed_dir, skip=skip_input | {CLEANUP_LOG}):
        return "gap", ""

    if b"INPUT_CONFIG_OK" in read(seed_dir / "LOG.check"):
        return "bug", "failed after deploying; see the seed's LOG.* files"

    # LOG.deploy* exists only once validate --strict passed, so a seed here is a
    # config the CLI accepted and then failed to deploy.
    if any(seed_dir.glob("LOG.deploy*")):
        return "bug", "deploy failed after validate --strict; see the seed's LOG.* files"

    return "rejected", ""


def resource_type(seed_dir):
    # Empty only when the mutator failed; classify() already reports that as a bug.
    raw = read(seed_dir / "LOG.config")
    if not raw:
        return "unknown"
    (rtype,) = json.loads(raw)["resources"]
    return rtype


def record(kind, seed, seed_dir):
    with open("LOG.summary", "a") as f:
        f.write(f"{kind} seed={seed} target={TARGET} type={resource_type(seed_dir)}\n")


def fail(seed, seed_dir, kind, reason, prefix=""):
    record(kind, seed, seed_dir)
    # Harness rewrites env values in stdout; ENVFILTER selects the matrix key.
    Path("LOG.repro").write_text(
        f"fuzz: seed {seed} {reason}, reproduce with: {prefix}"
        f"ENVFILTER=FUZZ_TARGET={TARGET} FUZZ_SEED_START={seed} "
        f"FUZZ_SEED_COUNT=1 FUZZ_CHECK_DRIFT={CHECK_DRIFT} ./task test-fuzz\n"
    )
    sys.exit(1)


def totals():
    summary = Path("LOG.summary")
    if not summary.exists():
        return Counter()

    kinds = Counter(line.split()[0] for line in summary.read_text().splitlines())
    with summary.open("a") as f:
        f.write("--- totals ---\n")
        for kind, n in sorted(kinds.items()):
            f.write(f"{n} {kind}\n")
    return kinds


def main():
    start = time.monotonic()
    seed_start = int(os.environ.get("FUZZ_SEED_START", "0"))
    # PR smoke default; nightly/task raise the ceiling and stop on FUZZ_TIME_BUDGET.
    count = int(os.environ.get("FUZZ_SEED_COUNT", "25"))

    for offset in range(count):
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

        if killed:
            fail(seed, seed_dir, "hang", f"hung (>{SEED_TIMEOUT:g}s)", "FUZZ_SEED_TIMEOUT=0 ")

        kind, reason = classify(seed_dir)
        if reason:
            fail(seed, seed_dir, kind, reason)
        record(kind, seed, seed_dir)

    kinds = totals()

    # Zero deploys is fine only when every seed is a gap. Single-seed repro is exempt.
    if count > 1 and not kinds:
        sys.exit("fuzz: no seeds ran")
    if count > 1 and not kinds["deployed"] and kinds["gap"] != sum(kinds.values()):
        sys.exit("fuzz: no seed deployed; the mutator or fixtures are broken")


if __name__ == "__main__":
    main()
