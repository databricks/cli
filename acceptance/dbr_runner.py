# Databricks notebook source
# This notebook is meant to be run from a job on a Databricks workspace.
# It is recommended to have the job cluster be a serverless cluster
# to match DABs in the workspace execution environment.
# The recommended flow to run this is:
# run: deco env run -i -n <env-name> -- go test -timeout 7200s -run TestAccept github.com/databricks/cli/acceptance -dbr
#    where <env-name> is the name of the environment to run the tests in. This will automatically
#    start a job to execute integration acceptance tests on a serverless cluster.

import os
import subprocess
import sys
import tarfile
import tempfile
from pathlib import Path
from dbruntime.databricks_repl_context import get_context


def extract_cli_archive():
    src = dbutils.widgets.get("archive_path")
    if not src:
        print("Error: archive_path is not set", file=sys.stderr)
        sys.exit(1)

    dst = Path(tempfile.mkdtemp(prefix="cli_archive"))

    with tarfile.open(src, "r:gz") as tar:
        tar.extractall(path=dst)

    print(f"Extracted {src} to {dst}")
    return dst


def main():
    archive_dir = extract_cli_archive()
    env = os.environ.copy()

    # Today all serverless instances are amd64. There are plans to also
    # have ARM based instances in Q4 FY26 but for now we can keep using the amd64
    # binaries without checking for the architecture.
    bin_dir = archive_dir / "bin" / "amd64"
    go_bin_dir = bin_dir / "go" / "bin"
    env["PATH"] = os.pathsep.join([str(go_bin_dir), str(bin_dir), env.get("PATH", "")])

    # Env vars used by the acceptance tests. These need to
    # be provided by the job parameters to the test runner here.
    envvars = [
        "CLOUD_ENV",
        "TEST_DEFAULT_CLUSTER_ID",
        "TEST_DEFAULT_WAREHOUSE_ID",
        "TEST_INSTANCE_POOL_ID",
        "TEST_METASTORE_ID",
    ]

    for envvar in envvars:
        env[envvar] = dbutils.widgets.get(envvar)
        assert env[envvar] is not None, f"Error: {envvar} is not set"

    ctx = get_context()
    workspace_url = spark.conf.get("spark.databricks.workspaceUrl")

    # Configure auth for the acceptance tests.
    env["DATABRICKS_TOKEN"] = ctx.apiToken
    env["DATABRICKS_HOST"] = workspace_url

    # Change working directory to the root of the CLI repo.
    os.chdir(archive_dir / "cli")
    cmd = ["go", "test", "-timeout", "7200s", "-run", r"^TestAccept", "github.com/databricks/cli/acceptance",  "-workspace-tmp-dir"]

    if dbutils.widgets.get("short") == "true":
        cmd.append("-short")

    print("Running acceptance tests...")
    result = subprocess.run(cmd, env=env, check=False)
    print(result.stdout, flush=True)
    print(result.stderr, flush=True)
    assert result.returncode == 0, "Acceptance tests failed"


if __name__ == "__main__":
    main()
