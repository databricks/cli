#!/usr/bin/env python3
"""Retry a command until it succeeds and its output matches expectations.

Usage: retry.py [--until SUBSTR] [--until-not SUBSTR] CMD [ARGS...]

Retries CMD up to 5 times (configurable via RETRY_MAX_ATTEMPTS env var),
sleeping RETRY_INTERVAL_MS milliseconds (default 500) between attempts.
An attempt is considered successful when the command exits with code 0 and:
  --until SUBSTR     SUBSTR appears in stdout
  --until-not SUBSTR SUBSTR does not appear in stdout

The condition is checked on every attempt, including the last. If a
--until/--until-not condition was given but never satisfied within
RETRY_MAX_ATTEMPTS, retry.py writes the last (stale) output to stderr and exits
non-zero, rather than passing the stale output through on stdout. When no
condition is given, the final attempt's output and exit code pass through
unchanged.
"""

import argparse
import os
import subprocess
import sys
import time


def main():
    parser = argparse.ArgumentParser(prog="retry.py")
    parser.add_argument("--until")
    parser.add_argument("--until-not")
    parser.add_argument("cmd", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    if not args.cmd:
        parser.error("no command given")
    until = args.until
    until_not = args.until_not
    argv = args.cmd

    interval = float(os.environ.get("RETRY_INTERVAL_MS", "500")) / 1000.0
    max_attempts = int(os.environ.get("RETRY_MAX_ATTEMPTS", "5"))

    def succeeded(result):
        return (
            result.returncode == 0
            and (until is None or until.encode() in result.stdout)
            and (until_not is None or until_not.encode() not in result.stdout)
        )

    ok = False
    result = None
    for attempt in range(max_attempts):
        if attempt > 0:
            time.sleep(interval)
        result = subprocess.run(argv, capture_output=True)
        if succeeded(result):
            ok = True
            break

    if not ok and (until is not None or until_not is not None):
        # A content condition was requested but never held within max_attempts. Emit the
        # stale output to stderr (never stdout) so callers capturing stdout don't silently
        # ingest stale data, and fail loudly with a non-zero exit so the flake is
        # attributed here rather than surfacing later as an unexplained output diff.
        sys.stderr.buffer.write(result.stdout)
        sys.stderr.buffer.write(result.stderr)
        sys.stderr.buffer.flush()
        print(f"retry: condition not met after {max_attempts} attempts", file=sys.stderr)
        sys.exit(result.returncode or 1)

    # Success, or no content condition was requested (retries only guarded the exit code).
    # Either way callers capture stdout (e.g. DASHBOARD=$(retry --until ...)), so pass the
    # final attempt's output and exit code through unchanged.
    sys.stdout.buffer.write(result.stdout)
    sys.stderr.buffer.write(result.stderr)
    sys.exit(result.returncode)


if __name__ == "__main__":
    main()
