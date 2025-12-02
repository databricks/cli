import sys
import subprocess
import json
import shlex


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
        raise RunError(f"{cmd} returned non-json: {ex}\n{result.stdout}")


def run(cmd):
    if VERBOSE:
        print("+ " + " ".join([shlex.quote(x) for x in cmd]), file=sys.stderr, flush=True)
    result = subprocess.run(cmd)
    if result.returncode != 0:
        raise RunError(f"{cmd} failed with code {result.returncode}")
    return result
