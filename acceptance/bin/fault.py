#!/usr/bin/env python3
"""Set up a fault rule on the testserver for the current test token.

Usage: fault.py [--after-handler] PATTERN STATUS_CODE OFFSET TIMES [ERROR_CODE]

  --after-handler  let the handler run and keep the state change it makes,
                   replacing only its response; models a response lost on the
                   way back from a request the backend already carried out
  PATTERN     HTTP method and path, supports trailing * wildcard,
              e.g. "PUT /api/2.0/permissions/pipelines/*"
  STATUS_CODE HTTP status code to return, e.g. 504
  OFFSET      number of requests to let through before fault starts
  TIMES       number of times to return the fault response
  ERROR_CODE  optional error_code for the response body, e.g.
              MAX_CHILD_NODE_SIZE_EXCEEDED (defaults to INJECTED)

The rule is scoped to the current DATABRICKS_TOKEN so it only affects
the test that registers it, even when tests share a server.
"""

import json
import os
import sys
import urllib.request

host = os.environ.get("DATABRICKS_HOST", "")
token = os.environ.get("DATABRICKS_TOKEN", "")

if not host:
    print("DATABRICKS_HOST not set", file=sys.stderr)
    sys.exit(1)

args = sys.argv[1:]
after_handler = "--after-handler" in args
if after_handler:
    args.remove("--after-handler")

if len(args) not in (4, 5):
    print(f"usage: {sys.argv[0]} [--after-handler] PATTERN STATUS_CODE OFFSET TIMES [ERROR_CODE]", file=sys.stderr)
    sys.exit(1)

pattern, status_code, offset, times = args[0], int(args[1]), int(args[2]), int(args[3])
error_code = args[4] if len(args) == 5 else "INJECTED"
body = json.dumps({"error_code": error_code, "message": "Fault injected by test."})

data = json.dumps(
    {
        "pattern": pattern,
        "status_code": status_code,
        "body": body,
        "offset": offset,
        "times": times,
        "after_handler": after_handler,
    }
).encode()

req = urllib.request.Request(
    f"{host}/__testserver/fault",
    data=data,
    headers={"Content-Type": "application/json", "Authorization": f"Bearer {token}"},
    method="POST",
)
urllib.request.urlopen(req)
