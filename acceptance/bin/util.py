import json
import shlex
import subprocess
import sys

VERBOSE = False


class RunError(Exception):
    pass


def run_json(cmd):
    if VERBOSE:
        print("+ " + " ".join([shlex.quote(x) for x in cmd]), file=sys.stderr, flush=True)
    result = subprocess.run(cmd, stdout=subprocess.PIPE, encoding="utf-8")
    if VERBOSE and result.stdout:
        print(result.stdout, flush=True)
    if result.returncode != 0:
        raise RunError(f"{cmd} failed with code {result.returncode}\n{result.stdout}".strip())
    try:
        return json.loads(result.stdout)
    except Exception as ex:
        raise RunError(f"{cmd} returned non-json: {ex}\n{result.stdout}") from ex


def run(cmd):
    if VERBOSE:
        print("+ " + " ".join([shlex.quote(x) for x in cmd]), file=sys.stderr, flush=True)
    result = subprocess.run(cmd)
    if result.returncode != 0:
        raise RunError(f"{cmd} failed with code {result.returncode}")
    return result


def load_plan(path):
    # Empty or invalid output means `bundle plan` failed; exit cleanly with the reason
    # (no traceback) rather than raising, so the failure reads as a plain message.
    # Returns (data, raw).
    with open(path) as fobj:
        raw = fobj.read()
    if not raw.strip():
        sys.exit(f"{path}: empty plan output (bundle plan failed)")
    try:
        return json.loads(raw), raw
    except json.JSONDecodeError as e:
        sys.exit(f"{path}: invalid plan JSON: {e}\n{raw}")
