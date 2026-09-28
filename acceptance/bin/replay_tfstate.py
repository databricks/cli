#!/usr/bin/env python3
"""
Reconstruct a terraform deployment for the migrate tests without the (removed) terraform engine.

Given a checked-in terraform.tfstate and the resource-mutating requests the terraform deploy made
(both captured once from a real terraform run), replay the requests against the backend so it ends
up in the exact terraform-shaped state, then write the tfstate locally with each recorded resource
id rewritten to the one the backend just minted. The migrate step then reads a real terraform state
backed by real (terraform-shaped) resources.

The requests carry the recorded ids; the backend mints fresh ones on replay. We learn each mapping
from the create responses (matching a create to its tfstate resource by name) and rewrite it into
every later request and into the tfstate, so cross-resource references stay consistent.

Usage: replay_tfstate.py REQUESTS.json TFSTATE.json [-t TARGET]
"""

import argparse
import json
import os
import re
import subprocess
import sys

CLI = os.environ["CLI"]
HOST = os.environ.get("DATABRICKS_HOST", "").rstrip("/")
UNIQUE_NAME = os.environ.get("UNIQUE_NAME", "")

# Per resource: which response field carries the new id. Terraform's tfstate stores the same value
# as the resource's `id`, keyed here by the create path so we can match a create to its tfstate entry.
ID_FIELDS = ["job_id", "pipeline_id", "dashboard_id", "experiment_id", "full_name", "id", "name"]

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


def api(method, path, body):
    cmd = [CLI, "api", method.lower(), path, "--output", "json"]
    if body is not None:
        cmd += ["--json", json.dumps(body)]
    r = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, encoding="utf-8")
    if r.returncode != 0:
        sys.exit(f"replay_tfstate.py: {method} {path} failed:\n{r.stdout}\n{r.stderr}")
    return json.loads(r.stdout) if r.stdout.strip() else {}


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

    # Drop the fixtures from the bundle root so a later deploy does not sync them as bundle files.
    os.remove(args.requests)
    os.remove(args.tfstate)

    # name -> recorded resource attributes, from the tfstate managed resources.
    name_to_attrs = {}
    for r in tfstate["resources"]:
        if r.get("mode") != "managed":
            continue
        attrs = r["instances"][0]["attributes"]
        if "name" in attrs and "id" in attrs:
            name_to_attrs[attrs["name"]] = attrs

    id_map = {}
    for req in requests:
        path = apply_map(req["path"], id_map)
        body = req.get("body")
        if body is not None:
            body = json.loads(apply_map(json.dumps(body), id_map))
        resp = api(req["method"], path, body)

        # A create (POST that returns a new id) whose config name matches a tfstate resource:
        # map its recorded id to the freshly minted one.
        if req["method"] == "POST" and isinstance(body, dict) and body.get("name") in name_to_attrs:
            attrs = name_to_attrs[body["name"]]
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
    with open(os.path.join(out_dir, "terraform.tfstate"), "w") as f:
        f.write(apply_map(tfstate_raw, id_map))


if __name__ == "__main__":
    main()
