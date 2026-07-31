#!/usr/bin/env python3
"""
List the entries of the AI Runtime code snapshot tarball uploaded during deploy.

Reads out.requests.txt, takes code_source_path from the jobs/create request,
exports that workspace file via the CLI, and prints its sorted tar entries. Used
to assert which local files the snapshot includes (gitignore / sync rules).
"""

import gzip
import io
import json
import os
import subprocess
import sys
import tarfile

with open("out.requests.txt") as f:
    text = f.read()

# out.requests.txt is a stream of concatenated (pretty-printed) JSON objects.
decoder = json.JSONDecoder()
requests = []
pos = 0
while pos < len(text):
    while pos < len(text) and text[pos].isspace():
        pos += 1
    if pos >= len(text):
        break
    obj, pos = decoder.raw_decode(text, pos)
    requests.append(obj)

code_source_path = None
for req in requests:
    body = req.get("body")
    if isinstance(body, dict) and req.get("path", "").endswith("/jobs/create"):
        code_source_path = body["tasks"][0]["ai_runtime_task"]["code_source_path"]

if not code_source_path:
    sys.exit("no jobs/create request with code_source_path in out.requests.txt")

# code_source_path is recorded de-prefixed (/Users/...); export needs /Workspace.
remote = "/Workspace" + code_source_path
cli = os.environ["CLI"]
local = "code_snapshot.tar.gz"
# MSYS_NO_PATHCONV stops Git Bash on Windows from rewriting the /Workspace path.
env = {**os.environ, "MSYS_NO_PATHCONV": "1"}
subprocess.run(
    [cli, "workspace", "export", remote, "--format", "AUTO", "--file", local],
    check=True, env=env,
)
with open(local, "rb") as f:
    data = gzip.decompress(f.read())
os.remove(local)

with tarfile.open(fileobj=io.BytesIO(data)) as tar:
    for name in sorted(tar.getnames()):
        print(name)
