#!/usr/bin/env python3
"""
Capture a terraform deployment's artifacts as migrate-test fixtures. Run ONCE, right after a
`DATABRICKS_BUNDLE_ENGINE=terraform bundle deploy`, while the terraform engine is still present.

Writes into the test's SOURCE dir ($TESTDIR):
  - terraform.tfstate        the real terraform state
  - terraform-requests.json  the resource-mutating requests (with bodies) terraform issued

replay_tfstate.py later replays the requests to reconstruct a real terraform-shaped backend and
drops in the (id-remapped) tfstate, so the migrate test starts from a real terraform deployment
without running the (removed) terraform engine. The capture-time host is normalized to
[DATABRICKS_URL] so the fixtures do not embed a per-run port.

Usage: capture_tfstate.py [TARGET]   (TARGET defaults to "default")
"""

import json
import os
import re
import sys

TESTDIR = os.environ["TESTDIR"]
HOST = os.environ.get("DATABRICKS_HOST", "").rstrip("/")
# Per-run values baked into the captured state/requests are normalized to placeholders so the
# fixtures are frozen; replay_tfstate.py substitutes the current run's values back in.
UNIQUE_NAME = os.environ.get("UNIQUE_NAME", "")

# Requests that build backend resource state; everything else (file sync, state push, lock,
# telemetry, scim, mkdirs, deletes) is deploy plumbing the migrate test does not need reconstructed.
SKIP = re.compile(r"/api/2\.0/workspace(/|-files)|/telemetry|/scim|/\.well-known|/state/|/mkdirs")


def read_json_many(s):
    dec = json.JSONDecoder()
    out = []
    pos, n = 0, len(s)
    while pos < n:
        while pos < n and s[pos] in " \t\r\n":
            pos += 1
        if pos >= n:
            break
        obj, pos = dec.raw_decode(s, pos)
        out.append(obj)
    return out


def norm(text):
    if HOST:
        text = text.replace(HOST, "[DATABRICKS_URL]")
    # UNIQUE_NAME is a long random per-run token, so a plain replace is safe.
    if len(UNIQUE_NAME) >= 8:
        text = text.replace(UNIQUE_NAME, "[UNIQUE_NAME]")
    return text


def main():
    target = sys.argv[1] if len(sys.argv) > 1 else "default"
    tfstate_raw = open(f".databricks/bundle/{target}/terraform/terraform.tfstate").read()
    requests = read_json_many(open(os.environ["OUT_REQUESTS"]).read())

    kept = [
        {"method": r["method"], "path": r["path"], "body": r.get("body")}
        for r in requests
        if r.get("method") in ("POST", "PUT", "PATCH") and not SKIP.search(r.get("path", ""))
    ]

    with open(os.path.join(TESTDIR, "terraform.tfstate"), "w") as f:
        f.write(norm(tfstate_raw))
    with open(os.path.join(TESTDIR, "terraform-requests.json"), "w") as f:
        f.write(norm(json.dumps(kept, indent=2)) + "\n")


if __name__ == "__main__":
    main()
