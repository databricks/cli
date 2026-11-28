#!/usr/bin/env python3
"""Make the testserver send a request of its own when a request of the test arrives.

Usage: hook.py [--clock-offset-ms N] PATTERN OFFSET TIMES REQUEST [BODY]

  --clock-offset-ms  shift the time the handler of REQUEST sees by N milliseconds, e.g. -30000
                     for a resource created 30 seconds ago. Only handlers that take the time
                     from the request honor it (today: the vector search endpoint create).
  PATTERN     HTTP method and path of the request to wait for, as for fault.py:
              * matches within one path segment, ** spans /
  OFFSET      number of matching requests to let through before the hook runs
  TIMES       number of times to run the hook
  REQUEST     HTTP method and path of the request to send, e.g.
              "POST /api/2.0/vector-search/endpoints"
  BODY        optional body of the request to send

The server handles REQUEST after the matching request is handled (or failed by a fault
rule) and before its response is sent. So REQUEST lands between that request and the next
one the CLI sends, whatever the speed of the machine: use it for what another client does
in the middle of a command, instead of a background process and a sleep. REQUEST is sent
with the current DATABRICKS_TOKEN, is recorded like any other request, and fails the test
if it is answered with an error.

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

clock_offset_ms = 0
if "--clock-offset-ms" in args:
    i = args.index("--clock-offset-ms")
    clock_offset_ms = int(args[i + 1])
    del args[i : i + 2]

if len(args) not in (4, 5) or " " not in args[3]:
    print(
        f"usage: {sys.argv[0]} [--clock-offset-ms N] PATTERN OFFSET TIMES REQUEST [BODY]",
        file=sys.stderr,
    )
    sys.exit(1)

pattern, offset, times = args[0], int(args[1]), int(args[2])
method, path = args[3].split(" ", 1)
body = args[4] if len(args) == 5 else ""

data = json.dumps(
    {
        "pattern": pattern,
        "offset": offset,
        "times": times,
        "method": method,
        "path": path,
        "body": body,
        "clock_offset_ms": clock_offset_ms,
    }
).encode()

req = urllib.request.Request(
    f"{host}/__testserver/hook",
    data=data,
    headers={"Content-Type": "application/json", "Authorization": f"Bearer {token}"},
    method="POST",
)
urllib.request.urlopen(req)
