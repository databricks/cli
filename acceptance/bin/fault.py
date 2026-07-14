#!/usr/bin/env python3
"""Set up a fault rule on the testserver for the current test token.

Usage: fault.py PATTERN STATUS_CODE OFFSET TIMES

  PATTERN     HTTP method and path, supports trailing * wildcard,
              e.g. "PUT /api/2.0/permissions/pipelines/*"
  STATUS_CODE HTTP status code to return, e.g. 504
  OFFSET      number of requests to let through before fault starts
  TIMES       number of times to return the fault response

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

if len(sys.argv) != 5:
    print(f"usage: {sys.argv[0]} PATTERN STATUS_CODE OFFSET TIMES", file=sys.stderr)
    sys.exit(1)

pattern, status_code, offset, times = sys.argv[1], int(sys.argv[2]), int(sys.argv[3]), int(sys.argv[4])
body = '{"error_code":"INJECTED","message":"Fault injected by test."}'

data = json.dumps(
    {
        "pattern": pattern,
        "status_code": status_code,
        "body": body,
        "offset": offset,
        "times": times,
    }
).encode()

req = urllib.request.Request(
    f"{host}/__testserver/fault",
    data=data,
    headers={"Content-Type": "application/json", "Authorization": f"Bearer {token}"},
    method="POST",
)
urllib.request.urlopen(req)
