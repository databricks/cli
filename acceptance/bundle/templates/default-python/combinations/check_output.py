#!/usr/bin/env python3
import sys
import os
import subprocess

SERVERLESS = os.environ["S"] == "yes"
INCLUDE_PYTHON = os.environ["P"] == "yes"

CLOUD_ENV = os.environ.get("CLOUD_ENV")
if CLOUD_ENV and SERVERLESS and not os.environ.get("TEST_METASTORE_ID"):
    sys.exit(f"SKIP_TEST SERVERLESS=yes but TEST_METASTORE_ID is empty in this env {CLOUD_ENV=}")

BUILDING = "Building python_artifact"
UPLOADING = "Uploading dist/"
STATE = "Updating deployment state"


def is_printable_line(line):
    # only shown when include_python=yes
    if line.startswith(BUILDING):
        return False

    # only shown when include_python=yes
    if line.startswith(UPLOADING):
        return False

    # not shown when all settings are equal to "no"
    if line.startswith(STATE):
        return False

    return True


p = subprocess.run(sys.argv[1:], stdout=subprocess.PIPE, stderr=subprocess.PIPE, encoding="utf-8")
try:
    assert p.returncode == 0
    assert p.stdout == ""
    for line in p.stderr.strip().split("\n"):
        if is_printable_line(line):
            print(line.strip())

    if INCLUDE_PYTHON:
        assert BUILDING in p.stderr
        assert UPLOADING in p.stderr
    else:
        assert BUILDING not in p.stderr
        assert UPLOADING not in p.stderr

except:
    print(f"STDOUT: {len(p.stdout)} chars")
    if p.stdout:
        print(p.stdout)
    print(f"STDERR: {len(p.stderr)} chars")
    if p.stderr:
        print(p.stderr)
    print(f"CODE: {p.returncode}", flush=True)
    raise
