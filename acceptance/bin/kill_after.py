#!/usr/bin/env python3
"""Set up a kill rule on the testserver for the current test token.

Usage: kill_after.py PATTERN OFFSET TIMES

  PATTERN   HTTP method and path, e.g. "POST /api/2.2/jobs/create"
  OFFSET    number of requests to let through before killing starts
  TIMES     number of times to kill the caller

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

if len(sys.argv) != 4:
    print(f"usage: {sys.argv[0]} PATTERN OFFSET TIMES", file=sys.stderr)
    sys.exit(1)

pattern, offset, times = sys.argv[1], int(sys.argv[2]), int(sys.argv[3])

data = json.dumps({"pattern": pattern, "offset": offset, "times": times}).encode()
req = urllib.request.Request(
    f"{host}/__testserver/kill",
    data=data,
    headers={"Content-Type": "application/json", "Authorization": f"Bearer {token}"},
    method="POST",
)
urllib.request.urlopen(req)
