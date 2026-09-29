#!/usr/bin/env python3
"""
Reconstruct a terraform deployment for the migrate tests without the (removed) terraform engine.

Given a checked-in terraform.tfstate and the resource-mutating requests the terraform deploy made
(both captured once from a real terraform run), replay the requests against the backend so it ends
up in the exact terraform-shaped state, then write the tfstate locally with each recorded resource
id rewritten to the one the backend just minted. The migrate step then reads a real terraform state
backed by real (terraform-shaped) resources.

The requests carry the recorded ids; the backend mints fresh ones on replay. We learn each mapping
from the create responses (matching a create to its tfstate resource by an identifying name) and
rewrite it into every later request and into the tfstate, so cross-resource references stay consistent.

Usage: replay_tfstate.py REQUESTS.json TFSTATE.json [-t TARGET]
"""

import argparse
import json
import os
import re
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request

import print_state

CLI = os.environ["CLI"]
HOST = os.environ.get("DATABRICKS_HOST", "").rstrip("/")
UNIQUE_NAME = os.environ.get("UNIQUE_NAME", "")

# Talk to the workspace directly rather than via `databricks api`: the CLI renders responses through
# Go's `any`, which decodes JSON numbers as float64 and silently mangles the low digits of a freshly
# minted id (they exceed 2^53), so the id we'd record would not match the one the backend stored.
# Python's json keeps ids exact. Bypass any proxy: these tests only hit the workspace directly.
_opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))


# The fake server routes by the PAT in DATABRICKS_TOKEN. These fixtures are captured against it, so
# the migrate tests that replay them are local-only (see their test.toml) - there is no cloud path.
TOKEN = os.environ.get("DATABRICKS_TOKEN", "")

# Per resource: which response field carries the new id. Terraform's tfstate stores the same value
# as the resource's `id`, keyed here by the create path so we can match a create to its tfstate entry.
ID_FIELDS = ["job_id", "pipeline_id", "dashboard_id", "experiment_id", "cluster_id", "full_name", "id", "name"]

# A create request is matched to its tfstate resource by a human-supplied identifying field; different
# resource types name it differently (jobs/pipelines use `name`, dashboards use `display_name`,
# clusters use `cluster_name`, …).
NAME_FIELDS = ["name", "display_name", "full_name", "cluster_name"]

# A backend-minted value that another resource may reference (e.g. a volume's storage_location, which
# a pipeline tags). We remap these recorded->minted just like ids: a UUID, an s3-style path (which
# embeds a UUID), or a long numeric id. Timestamps (13-digit millis) are shorter and excluded.
BACKEND_VALUE = re.compile(r"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}")


def is_backend_value(v):
    s = str(v)
    return bool(BACKEND_VALUE.search(s)) or s.startswith("s3://") or (s.isdigit() and len(s) >= 15)


def apply_map(text, id_map):
    for old, new in id_map.items():
        text = text.replace(old, new)
    return text


def run_cli(*args):
    r = subprocess.run([CLI, *args], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, encoding="utf-8")
    if r.returncode != 0:
        sys.exit(f"replay_tfstate.py: {' '.join(args)} failed:\n{r.stdout}")


def api(method, path, body):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(HOST + path, data=data, method=method.upper())
    req.add_header("Authorization", "Bearer " + TOKEN)
    # Stable UA so tests that record the replayed requests don't pin the Python version.
    req.add_header("User-Agent", "replay_tfstate.py")
    if data is not None:
        req.add_header("Content-Type", "application/json")
    try:
        with _opener.open(req) as resp:
            text = resp.read().decode()
    except urllib.error.HTTPError as e:
        sys.exit(f"replay_tfstate.py: {method} {path} failed: HTTP {e.code}\n{e.read().decode()}")
    return json.loads(text) if text.strip() else {}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("requests")
    parser.add_argument("tfstate")
    parser.add_argument("-t", "--target", default="default")
    args = parser.parse_args()

    requests_raw = open(args.requests).read()
    tfstate_raw = open(args.tfstate).read()

    # Restore this run's values into the frozen fixtures. UNIQUE_NAME goes into both (resource
    # names must match the current config); the host goes into the requests we issue, but the
    # tfstate keeps [DATABRICKS_URL] so print_state's output matches the goldens.
    if UNIQUE_NAME:
        requests_raw = requests_raw.replace("[UNIQUE_NAME]", UNIQUE_NAME)
        tfstate_raw = tfstate_raw.replace("[UNIQUE_NAME]", UNIQUE_NAME)
    if HOST:
        requests_raw = requests_raw.replace("[DATABRICKS_URL]", HOST)

    requests = json.loads(requests_raw)
    tfstate = json.loads(tfstate_raw)

    # The fixtures stay on disk (a later deploy skips them via the .gitignore script.prepare writes),
    # so replay is non-destructive and a single fixture can be replayed more than once.

    # identifying name -> recorded resource attributes, from the tfstate managed resources.
    name_to_attrs = {}
    for r in tfstate["resources"]:
        if r.get("mode") != "managed":
            continue
        attrs = r["instances"][0]["attributes"]
        if "id" not in attrs:
            continue
        for nf in NAME_FIELDS:
            if attrs.get(nf):
                name_to_attrs[str(attrs[nf])] = attrs

    id_map = {}
    for req in requests:
        path = apply_map(req["path"], id_map)
        # Reattach recorded query params (the backend reads some required ids from there, e.g.
        # postgres `?project_id=...`); id-remap the values in case one references a minted id.
        q = req.get("q")
        if q:
            items = [
                (k, apply_map(str(val), id_map))
                for k, vals in q.items()
                for val in (vals if isinstance(vals, list) else [vals])
            ]
            path += "?" + urllib.parse.urlencode(items)
        body = req.get("body")
        if body is not None:
            body = json.loads(apply_map(json.dumps(body), id_map))
        resp = api(req["method"], path, body)

        # A create (POST that returns a new id) whose identifying name matches a tfstate resource:
        # map its recorded id to the freshly minted one.
        create_name = None
        if isinstance(body, dict):
            create_name = next((str(body[nf]) for nf in NAME_FIELDS if body.get(nf)), None)
        if req["method"] == "POST" and create_name in name_to_attrs:
            attrs = name_to_attrs[create_name]
            # When the resource's id is the identifying name we sent (registered models, serving
            # endpoints, …), it is client-provided and already stable, so there is nothing to remap;
            # extracting a backend id here would either fail (nested response) or map the name to a
            # separate uuid the response also carries. Only remap a genuinely backend-minted id.
            if str(attrs["id"]) != create_name:
                new_id = next((str(resp[f]) for f in ID_FIELDS if f in resp), None)
                if new_id is None:
                    sys.exit(f"replay_tfstate.py: no id in response to {req['path']}: {resp}")
                id_map[str(attrs["id"])] = new_id
            # Also remap other backend-minted values a later resource may reference (e.g. a volume's
            # storage_location that a pipeline tags), so those requests and the tfstate stay consistent.
            for k, v in resp.items():
                old = attrs.get(k)
                if old is not None and str(old) != str(v) and is_backend_value(old):
                    id_map[str(old)] = str(v)

    out_dir = os.path.join(".databricks", "bundle", args.target, "terraform")
    os.makedirs(out_dir, exist_ok=True)
    local_path = os.path.join(out_dir, "terraform.tfstate")
    with open(local_path, "w") as f:
        f.write(apply_map(tfstate_raw, id_map))

    # A real terraform deploy leaves its state both on disk and in the workspace; push the remote
    # copy too so a later migrate has a remote state to back up and tests that inspect the remote
    # state dir see it. workspace import does not create parents, so mkdirs first.
    state_dir = print_state.get_remote_state_path(args.target)
    run_cli("workspace", "mkdirs", state_dir)
    run_cli(
        "workspace", "import", f"{state_dir}/terraform.tfstate", "--file", local_path, "--format", "AUTO", "--overwrite"
    )


if __name__ == "__main__":
    main()
