#!/usr/bin/env python3
"""
List the entries of each AI Runtime code snapshot tarball uploaded during deploy.

Reads out.requests.txt, takes every ai_runtime_task.code_source_path from the
jobs/create request, exports each workspace archive via the CLI, and prints its
sorted tar entries (grouped per archive). Used to assert which local files each
snapshot includes (gitignore / sync rules), across one or more tasks.
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

# Collect every task's code_source_path from the jobs/create request(s).
code_source_paths = []
for req in requests:
    body = req.get("body")
    if isinstance(body, dict) and req.get("path", "").endswith("/jobs/create"):
        for task in body.get("tasks", []):
            art = task.get("ai_runtime_task")
            if art and art.get("code_source_path"):
                code_source_paths.append(art["code_source_path"])

if not code_source_paths:
    sys.exit("no jobs/create request with code_source_path in out.requests.txt")

cli = os.environ["CLI"]
# MSYS_NO_PATHCONV stops Git Bash on Windows from rewriting the /Workspace path.
env = {**os.environ, "MSYS_NO_PATHCONV": "1"}

# code_source_path is an absolute workspace path (/Workspace/Users/.../files/...).
# Sort so multi-task output is deterministic.
for remote in sorted(code_source_paths):
    local = "code_snapshot.tar.gz"
    subprocess.run(
        [cli, "workspace", "export", remote, "--format", "AUTO", "--file", local],
        check=True,
        env=env,
    )
    with open(local, "rb") as f:
        data = gzip.decompress(f.read())
    os.remove(local)

    # Print the archive's sync-relative name (hash tokenized by test.toml repls) so
    # multi-archive output is legible.
    print(f"# {remote.split('/files/', 1)[-1]}")
    with tarfile.open(fileobj=io.BytesIO(data)) as tar:
        for name in sorted(tar.getnames()):
            print(name)
