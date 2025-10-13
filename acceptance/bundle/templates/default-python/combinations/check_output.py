#!/usr/bin/env python3
import sys
import os
import re
import subprocess

SERVERLESS = os.environ["SERVERLESS"] == "yes"
INCLUDE_PYTHON = os.environ["PY"] == "yes"

CLOUD_ENV = os.environ.get("CLOUD_ENV")
if CLOUD_ENV and SERVERLESS and not os.environ.get("TEST_METASTORE_ID"):
    sys.exit(f"SKIP_TEST SERVERLESS=yes but TEST_METASTORE_ID is empty in this env {CLOUD_ENV=}")

BUILDING = "Building python_artifact"
UPLOADING_WHL = re.compile(r"^Uploading .*whl\.\.\.$", re.M)
STATE = "Updating deployment state"


def is_printable_line(line):
    # only shown when include_python=yes
    if line.startswith(BUILDING):
        return False

    # only shown when include_python=yes
    if UPLOADING_WHL.match(line):
        return False

    # not shown when all settings are equal to "no"
    if line.startswith(STATE):
        return False

    return True


p = subprocess.run(sys.argv[1:], stdout=subprocess.PIPE, stderr=subprocess.PIPE, encoding="utf-8")
try:
    assert p.returncode == 0, p.returncode
    assert p.stdout == ""
    if INCLUDE_PYTHON:
        assert BUILDING in p.stderr, BUILDING
        assert UPLOADING_WHL.search(p.stderr), UPLOADING_WHL
    else:
        assert BUILDING not in p.stderr, BUILDING
        assert not UPLOADING_WHL.search(p.stderr), UPLOADING_WHL

    for line in p.stderr.strip().split("\n"):
        if is_printable_line(line):
            print(line.strip())

except:
    print(f"STDOUT: {len(p.stdout)} chars")
    if p.stdout:
        print(p.stdout)
    print(f"STDERR: {len(p.stderr)} chars")
    if p.stderr:
        print(p.stderr)
    print(f"CODE: {p.returncode}", flush=True)
    raise
