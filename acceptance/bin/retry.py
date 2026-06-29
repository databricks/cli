#!/usr/bin/env python3
"""Retry a command until it succeeds and its output matches expectations.

Usage: retry.py [--until SUBSTR] [--until-not SUBSTR] CMD [ARGS...]

Retries CMD up to 5 times (configurable via RETRY_MAX_ATTEMPTS env var),
sleeping RETRY_INTERVAL_MS milliseconds (default 500) between attempts.
An attempt is considered successful when the command exits with code 0 and:
  --until SUBSTR     SUBSTR appears in stdout
  --until-not SUBSTR SUBSTR does not appear in stdout
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

    result = subprocess.run(argv, capture_output=True)
    for _ in range(1, max_attempts):
        success = (
            result.returncode == 0
            and (until is None or until.encode() in result.stdout)
            and (until_not is None or until_not.encode() not in result.stdout)
        )
        if success:
            break
        time.sleep(interval)
        result = subprocess.run(argv, capture_output=True)

    sys.stdout.buffer.write(result.stdout)
    sys.stderr.buffer.write(result.stderr)
    sys.exit(result.returncode)


if __name__ == "__main__":
    main()
