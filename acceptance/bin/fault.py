#!/usr/bin/env python3
"""Set up a fault rule on the testserver for the current test token.

Usage: fault.py PATTERN STATUS_CODE OFFSET TIMES [DELAY_MS]

  PATTERN     HTTP method and path, supports trailing * wildcard,
              e.g. "PUT /api/2.0/permissions/pipelines/*"
  STATUS_CODE HTTP status code to return, e.g. 504. Use 0 together with
              DELAY_MS for a delay-only rule that sleeps and then lets the
              real handler run (a slow-but-successful response).
  OFFSET      number of requests to let through before fault starts
  TIMES       number of times to apply the rule
  DELAY_MS    optional milliseconds to sleep before responding (default 0)

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

if len(sys.argv) not in (5, 6):
    print(f"usage: {sys.argv[0]} PATTERN STATUS_CODE OFFSET TIMES [DELAY_MS]", file=sys.stderr)
    sys.exit(1)

pattern, status_code, offset, times = sys.argv[1], int(sys.argv[2]), int(sys.argv[3]), int(sys.argv[4])
delay_ms = int(sys.argv[5]) if len(sys.argv) == 6 else 0
# A delay-only rule (status_code 0) has no error body; the testserver falls
# through to the real handler after sleeping.
body = "" if status_code == 0 else '{"error_code":"INJECTED","message":"Fault injected by test."}'

data = json.dumps(
    {
        "pattern": pattern,
        "status_code": status_code,
        "body": body,
        "delay_ms": delay_ms,
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
