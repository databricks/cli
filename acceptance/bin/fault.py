#!/usr/bin/env python3
"""Set up a fault rule on the testserver for the current test token.

Usage: fault.py [--after-handler] [--body-contains SUBSTR] PATTERN STATUS_CODE OFFSET TIMES [ERROR_CODE]

  --after-handler  run the handler first; keep its effect and replace only the response
  --body-contains  only fire when the request body contains SUBSTR, e.g. to target one
                   file's /workspace/import upload (all uploads share that path)
  PATTERN     HTTP method and path; * matches within one path segment, ** spans /,
              e.g. "POST /api/2.0/bundle/deployments/*/versions/*/complete"
              or "PATCH /api/2.0/bundle/**"
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

body_contains = ""
if "--body-contains" in args:
    i = args.index("--body-contains")
    body_contains = args[i + 1]
    del args[i : i + 2]

if len(args) not in (4, 5):
    print(
        f"usage: {sys.argv[0]} [--after-handler] [--body-contains SUBSTR] PATTERN STATUS_CODE OFFSET TIMES [ERROR_CODE]",
        file=sys.stderr,
    )
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
        "body_contains": body_contains,
    }
).encode()

req = urllib.request.Request(
    f"{host}/__testserver/fault",
    data=data,
    headers={"Content-Type": "application/json", "Authorization": f"Bearer {token}"},
    method="POST",
)
urllib.request.urlopen(req)
