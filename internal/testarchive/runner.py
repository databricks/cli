import os
import subprocess
import sys
import tarfile
from pathlib import Path

def extract_cli_archive():
    # Read from DATABRICKS_DBR_CLI_ARCHIVE
    src = os.environ.get("DATABRICKS_DBR_CLI_ARCHIVE")
    if not src:
        print("Error: DATABRICKS_DBR_CLI_ARCHIVE is not set", file=sys.stderr)
        sys.exit(1)

    home = Path.home()
    dst = home / "cli"

    os.makedirs(dst, exist_ok=True)

    with tarfile.open(src, "r:gz") as tar:
        tar.extractall(path=dst)

    print(f"Extracted {src} to {dst}")

def main():
    home = Path.home()

    # TODO: Have better organization for these binaries.
    cli_dir = home / "cli"
    go_bin = home / "cli" / "testdata" / "amd64" / "go" / "bin"
    uv_bin = home / "cli" / "testdata" / "x86_64" / "uv-x86_64-unknown-linux-gnu"
    jq_bin = home / "cli" / "testdata" / "amd64"

    # Ensure the directories exist (optional checks)
    for p in [cli_dir, go_bin, uv_bin]:
        if not p.exists():
            print(f"Warning: {p} does not exist", file=sys.stderr)

    # Build updated PATH
    env = os.environ.copy()

    # Prepend to PATH so these are found first
    env["PATH"] = os.pathsep.join([str(go_bin), str(uv_bin), str(jq_bin), env.get("PATH", "")])

    # TODO: pass cloudenv as a job parameter. We can pass through the existing local env var from
    # the runner.
    env["CLOUD_ENV"] = "dbr"

    # Configure auth for the workspace:
    env["DATABRICKS_TOKEN"] = ctx.apiToken
    env["DATABRICKS_HOST"] = workspace_url

    os.makedirs(cli_dir, exist_ok=True)

    # Change working directory
    os.chdir(cli_dir)

    # Command equivalent to:
    # go test -timeout 300s -run ^TestAccept/workspace/jobs/create github.com/databricks/cli/acceptance

    # TODO: Make the output format compatible with gotestsum. The gh_report and parse scripts
    # should work with the output from these.
    cmd = [
        "go", "test",
        "-timeout", "300s",
        "-run", r"^TestAccept/selftest/record_cloud/basic", "github.com/databricks/cli/acceptance",
        "-dbr",
        "github.com/databricks/cli/acceptance",
    ]

    # Run and stream output
    try:
        result = subprocess.run(cmd, env=env, check=False)
        # sys.exit(result.returncode)
    except FileNotFoundError:
        print("Error: 'go' executable not found in PATH. Check that go_bin exists and PATH was set.", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
