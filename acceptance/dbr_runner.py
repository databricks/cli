# Databricks notebook source

import os
import subprocess
import sys
import tarfile
from pathlib import Path
from dbruntime.databricks_repl_context import get_context

def extract_cli_archive():
    src = dbutils.widgets.get("archive_path")
    if not src:
        print("Error: archive_path is not set", file=sys.stderr)
        sys.exit(1)

    # Every serverless instance gets a unique home directory
    # mounted on the local file system.
    home = Path.home()
    dst = home / "archive"

    os.makedirs(dst, exist_ok=True)

    with tarfile.open(src, "r:gz") as tar:
        tar.extractall(path=dst)

    print(f"Extracted {src} to {dst}")
    return dst

def main():
    archive_dir = extract_cli_archive()
    env = os.environ.copy()

    # Today all serverless instances are AMD. There are plans to also
    # have arm based instances in Q4. For now we can keep using the AMD
    # binaries without checking for the architecture.
    go_bin_dir = archive_dir  / "bin" / "amd64" / "go" / "bin"
    bin_dir = archive_dir / "bin" / "amd64"
    env["PATH"] = os.pathsep.join([str(go_bin_dir), str(bin_dir), env.get("PATH", "")])

    # Read the cloud environment from the job parameters.
    env["CLOUD_ENV"] = dbutils.widgets.get("cloud_env")
    assert env["CLOUD_ENV"] in ["aws", "azure", "gcp"]

    ctx = get_context()
    workspace_url = spark.conf.get("spark.databricks.workspaceUrl")

    # Configure auth for the acceptance tests.
    env["DATABRICKS_TOKEN"] = ctx.apiToken
    env["DATABRICKS_HOST"] = workspace_url

    # Change working directory to the root of the CLI repo.
    os.chdir(archive_dir / "cli")
    cmd = [
        "go", "test",
        "-timeout", "7200s",
        "-run", r"^TestAccept/workspace/jobs/create",
        "github.com/databricks/cli/acceptance"
    ]

    print("Running acceptance tests...")
    result = subprocess.run(cmd, env=env, check=False)
    print(result.stdout)
    print(result.stderr)
    assert result.returncode == 0, "Acceptance tests failed"

if __name__ == "__main__":
    main()
