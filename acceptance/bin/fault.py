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

_DEFAULT_BODIES = {
    408: '{"error_code":"REQUEST_TIMEOUT","message":"Request timed out."}',
    500: '{"error_code":"INTERNAL_ERROR","message":"Internal server error."}',
    502: '{"error_code":"BAD_GATEWAY","message":"Bad gateway."}',
    503: '{"error_code":"SERVICE_UNAVAILABLE","message":"Service unavailable."}',
    504: '{"error_code":"TEMPORARILY_UNAVAILABLE","message":"The service is taking too long to process your request."}',
}

host = os.environ.get("DATABRICKS_HOST", "")
token = os.environ.get("DATABRICKS_TOKEN", "")

if not host:
    print("DATABRICKS_HOST not set", file=sys.stderr)
    sys.exit(1)

if len(sys.argv) != 5:
    print(f"usage: {sys.argv[0]} PATTERN STATUS_CODE OFFSET TIMES", file=sys.stderr)
    sys.exit(1)

pattern, status_code, offset, times = sys.argv[1], int(sys.argv[2]), int(sys.argv[3]), int(sys.argv[4])
body = _DEFAULT_BODIES.get(status_code, '{"error_code":"ERROR","message":"Fault injected by test."}')

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
