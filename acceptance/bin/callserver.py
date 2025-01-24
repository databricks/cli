#!/usr/bin/env python3
import sys
import os
import json
import urllib.request
from urllib.parse import urlencode

env = {}
for key, value in os.environ.items():
    if len(value) > 10_000:
        sys.stderr.write(f"Dropping key={key} value len={len(value)}\n")
        continue
    env[key] = value

q = {
    "args": " ".join(sys.argv[1:]),
    "cwd": os.getcwd(),
    "env": json.dumps(env),
}

url = os.environ["CMD_SERVER_URL"] + "/?" + urlencode(q)
if len(url) > 100_000:
    sys.exit("url too large")

resp = urllib.request.urlopen(url)
assert resp.status == 200, (resp.status, resp.url, resp.headers)
result = json.load(resp)
sys.stderr.write(result["stderr"])
sys.stdout.write(result["stdout"])
exitcode = int(result["exitcode"])
sys.exit(exitcode)
