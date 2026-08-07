#!/usr/bin/env python3
"""Set up a fault rule on the testserver for the current test token.

The rule is scoped to the current DATABRICKS_TOKEN so it only affects
the test that registers it, even when tests share a server.
"""

import argparse
import json
import os
import sys
import urllib.request

parser = argparse.ArgumentParser(description="Set up a fault rule on the testserver for the current test token.")
parser.add_argument(
    "pattern", help='HTTP method and path, supports trailing * wildcard, e.g. "PUT /api/2.0/permissions/pipelines/*"'
)
parser.add_argument("status_code", type=int, help="HTTP status code to return, e.g. 504")
parser.add_argument("offset", type=int, help="number of requests to let through before fault starts")
parser.add_argument("times", type=int, help="number of times to return the fault response")
parser.add_argument(
    "error_code",
    nargs="?",
    default="INJECTED",
    help="error_code for the response body, e.g. MAX_CHILD_NODE_SIZE_EXCEEDED",
)
parser.add_argument(
    "--body-contains",
    default="",
    metavar="SUBSTR",
    help="only fire when the request body contains SUBSTR. Needed to target a single file's "
    "/workspace/import upload, since every upload shares the same method+path and only "
    'differs by the multipart "path" form field.',
)
args = parser.parse_args()

host = os.environ.get("DATABRICKS_HOST", "")
token = os.environ.get("DATABRICKS_TOKEN", "")

if not host:
    print("DATABRICKS_HOST not set", file=sys.stderr)
    sys.exit(1)

body = json.dumps({"error_code": args.error_code, "message": "Fault injected by test."})

data = json.dumps(
    {
        "pattern": args.pattern,
        "status_code": args.status_code,
        "body": body,
        "offset": args.offset,
        "times": args.times,
        "body_contains": args.body_contains,
    }
).encode()

req = urllib.request.Request(
    f"{host}/__testserver/fault",
    data=data,
    headers={"Content-Type": "application/json", "Authorization": f"Bearer {token}"},
    method="POST",
)
urllib.request.urlopen(req)
