#!/usr/bin/env python3
import sys
import os
import json
import subprocess
import urllib.parse


args = " ".join(sys.argv[1:])
args = urllib.parse.quote_plus(args)
cwd = urllib.parse.quote_plus(os.getcwd())

url = os.environ["CMD_SERVER_URL"] + "/?args=" + args + "&cwd=" + cwd
out = subprocess.run(["curl", "-s", url], stdout=subprocess.PIPE, check=True)
result = json.loads(out.stdout)
sys.stderr.write(result["stderr"])
sys.stdout.write(result["stdout"])
exitcode = int(result["exitcode"])
sys.exit(exitcode)
