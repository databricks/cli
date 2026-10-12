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

Usage: capture_tfstate.py [TARGET] [--out PREFIX]
  TARGET   bundle target to read state from; auto-detected from .databricks/bundle/* if omitted.
  --out    output path prefix; writes PREFIX.tfstate + PREFIX.requests.json (parents created).
           Defaults to $TESTDIR/terraform, i.e. terraform.tfstate + terraform-requests.json.
"""

import argparse
import glob
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
# telemetry, scim, deletes) is deploy plumbing the migrate test does not need reconstructed.
# `workspace/mkdirs` is kept, though: a resource can require a parent directory (a dashboard's
# parent_path), which the migrate test's own deploy would not recreate before replay uses it.
SKIP = re.compile(r"/api/2\.0/workspace-files|/api/2\.0/workspace/(?!mkdirs)|/telemetry|/scim|/\.well-known|/state/")


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


def detect_target(explicit):
    if explicit:
        return explicit
    # The invariant configs declare no target, so find the one the deploy actually wrote.
    dirs = [d for d in glob.glob(".databricks/bundle/*") if os.path.exists(f"{d}/terraform/terraform.tfstate")]
    if len(dirs) != 1:
        sys.exit(f"capture_tfstate.py: expected exactly one deployed target, found {dirs}")
    return os.path.basename(dirs[0])


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("target", nargs="?")
    parser.add_argument("--out")
    args = parser.parse_args()

    target = detect_target(args.target)
    tfstate_raw = open(f".databricks/bundle/{target}/terraform/terraform.tfstate").read()
    requests = read_json_many(open(os.environ["OUT_REQUESTS"]).read())

    # Keep `q` (query params, recorded separately by the harness): some creates carry a required id
    # there, e.g. postgres `?project_id=...`, which the backend reads from the query, not the body.
    kept = [
        {"method": r["method"], "path": r["path"], **({"q": r["q"]} if r.get("q") else {}), "body": r.get("body")}
        for r in requests
        if r.get("method") in ("POST", "PUT", "PATCH") and not SKIP.search(r.get("path", ""))
    ]

    if args.out:
        tfstate_path = args.out + ".tfstate"
        requests_path = args.out + ".requests.json"
        os.makedirs(os.path.dirname(tfstate_path) or ".", exist_ok=True)
    else:
        tfstate_path = os.path.join(TESTDIR, "terraform.tfstate")
        requests_path = os.path.join(TESTDIR, "terraform-requests.json")

    with open(tfstate_path, "w") as f:
        f.write(norm(tfstate_raw))
    with open(requests_path, "w") as f:
        f.write(norm(json.dumps(kept, indent=2)) + "\n")


if __name__ == "__main__":
    main()
