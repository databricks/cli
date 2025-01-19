#!/usr/bin/env python3
import sys
import os
import json
import subprocess
from urllib.parse import quote_plus

env = {}
for key, value in os.environ.items():
    if "BUNDLE_VAR" in key or "DATABRICKS" in key:
        env[key] = value

q = [
    "args=" + quote_plus(" ".join(sys.argv[1:])),
    "cwd=" + quote_plus(os.getcwd()),
    "env=" + quote_plus(json.dumps(env)),
]

url = os.environ["CMD_SERVER_URL"] + "/?" + "&".join(q)
out = subprocess.run(["curl", "-s", url], stdout=subprocess.PIPE, check=True)
result = json.loads(out.stdout)
sys.stderr.write(result["stderr"])
sys.stdout.write(result["stdout"])
exitcode = int(result["exitcode"])
sys.exit(exitcode)
