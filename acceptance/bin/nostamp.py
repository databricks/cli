#!/usr/bin/env python3
"""Read JSON on stdin, write it back with the DMS deployment stamp removed.

Deployment history recording (DATABRICKS_BUNDLE_RECORD_DEPLOYMENT_HISTORY=true; see
bundle/test.toml) adds a deployment stamp to plans, states, and resource payloads. Pipe any of
those through this so an acceptance golden compares equal whether or not recording is on. Output
is a 2-space indent, keys in input order, <>& left unescaped, and integers at full precision.
Tests under bundle/dms assert the stamp itself and must not use it.

Removed in three shapes plus the plan header:
  1. deployment_id/version_id nested in a deployment block, recognized by the
     neighbouring "kind" and "metadata_file_path", which are kept; version_id
     == "" is kept (a Terraform state dump carries that for an unstamped job).
  2. changes entries keyed "deployment.deployment_id" / "deployment.version_id".
  3. a changes object emptied by (2), or already empty.
  4. the recorded plan header fields deployment_id, next_version_id, last_version_id.
"""

import argparse
import json
import sys

_STAMP_KEYS = ("deployment_id", "version_id")
_CHANGE_KEYS = ("deployment.deployment_id", "deployment.version_id")
_PLAN_HEADER_KEYS = ("deployment_id", "next_version_id", "last_version_id")


def scrub(node):
    if isinstance(node, dict):
        # 1. A deployment block is the only object carrying both of these.
        if "kind" in node and "metadata_file_path" in node:
            for k in _STAMP_KEYS:
                if node.get(k, "") != "":
                    node.pop(k, None)
        out = {}
        for k, v in node.items():
            if k == "changes" and isinstance(v, dict):
                v = {ck: scrub(cv) for ck, cv in v.items() if ck not in _CHANGE_KEYS}
                if not v:
                    continue  # 3. drop a changes object left (or already) empty
                out[k] = v
            else:
                out[k] = scrub(v)
        return out
    if isinstance(node, list):
        return [scrub(x) for x in node]
    return node


def render(data, indent):
    data = scrub(data)
    if isinstance(data, dict):
        for k in _PLAN_HEADER_KEYS:
            data.pop(k, None)  # 4. plan header, present only at the root
    return json.dumps(data, indent=indent, ensure_ascii=False, separators=(",", ": "))


def main():
    parser = argparse.ArgumentParser()
    # A state dump is printed with a single-space indent; plans use two.
    parser.add_argument("--indent", type=int, default=2)
    args = parser.parse_args()

    # Input is one JSON value (a plan or state dump) or a whitespace-separated stream of them (each
    # request emitted by a print_requests filter) - jq accepted both here, so this must too.
    text = sys.stdin.read()
    decoder = json.JSONDecoder()
    idx, n = 0, len(text)
    while idx < n:
        while idx < n and text[idx].isspace():
            idx += 1
        if idx >= n:
            break
        data, idx = decoder.raw_decode(text, idx)
        sys.stdout.write(render(data, args.indent) + "\n")


if __name__ == "__main__":
    main()
