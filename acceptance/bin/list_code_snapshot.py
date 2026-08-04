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
import os
import subprocess
import sys
import tarfile

from print_requests import read_json_many


def code_source_paths(requests):
    """Every task's code_source_path from the jobs/create request(s)."""
    result = []
    for req in requests:
        body = req.get("body")
        if isinstance(body, dict) and req.get("path", "").endswith("/jobs/create"):
            for task in body.get("tasks", []):
                art = task.get("ai_runtime_task")
                if art and art.get("code_source_path"):
                    result.append(art["code_source_path"])
    return result


def print_entries(cli, env, remote):
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


def main():
    with open("out.requests.txt") as f:
        requests = read_json_many(f.read())

    paths = code_source_paths(requests)
    if not paths:
        sys.exit("no jobs/create request with code_source_path in out.requests.txt")

    cli = os.environ["CLI"]
    # MSYS_NO_PATHCONV stops Git Bash on Windows from rewriting the /Workspace path.
    env = {**os.environ, "MSYS_NO_PATHCONV": "1"}

    # code_source_path is an absolute workspace path (/Workspace/Users/.../files/...).
    # Sort so multi-task output is deterministic.
    for remote in sorted(paths):
        print_entries(cli, env, remote)


if __name__ == "__main__":
    main()
