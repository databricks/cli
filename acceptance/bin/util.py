import json
import os
import shlex
import subprocess
import sys
import time

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


def run_statements(statements):
    """Runs SQL statements on the test warehouse one after another, waiting for each to succeed."""
    cli = os.environ.get("CLI", "databricks")
    warehouse_id = os.environ["TEST_DEFAULT_WAREHOUSE_ID"]
    for statement in statements:
        payload = json.dumps({"warehouse_id": warehouse_id, "statement": statement, "wait_timeout": "50s"})
        response = run_json([cli, "api", "post", "/api/2.0/sql/statements/", "--json", payload])
        # A warehouse that is still starting outlasts wait_timeout.
        deadline = time.monotonic() + 600
        while response["status"]["state"] in ("PENDING", "RUNNING") and time.monotonic() < deadline:
            time.sleep(5)
            response = run_json([cli, "api", "get", f"/api/2.0/sql/statements/{response['statement_id']}"])
        if response["status"]["state"] != "SUCCEEDED":
            raise RunError(f"Statement {statement!r} ended in {response['status']}")
