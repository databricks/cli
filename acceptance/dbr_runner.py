# Databricks notebook source

# This notebook runs CLI cloud acceptance tests on a DBR cluster.
# It is meant to be submitted as a job task from the TestDbrAcceptance* tests.
#
# The notebook expects the following parameters:
# - archive_path: Path to the archive.tar.gz file in the workspace
# - cloud_env: Cloud environment (e.g., "aws", "azure", "gcp")
# - test_filter: Optional regex filter for test names (e.g., "bundle/generate")
# - test_default_warehouse_id: Default SQL warehouse ID
# - test_default_cluster_id: Default cluster ID
# - test_instance_pool_id: Instance pool ID
# - test_metastore_id: Unity Catalog metastore ID
# - test_sp_application_id: Service principal application ID

import os
import platform
import subprocess
import sys
import tarfile
from datetime import datetime
from pathlib import Path

# COMMAND ----------


def get_workspace_client():
    """Get the workspace client using dbruntime context."""
    from databricks.sdk import WorkspaceClient

    return WorkspaceClient()


def get_auth_token():
    """Get the authentication token from the notebook context."""
    from dbruntime.databricks_repl_context import get_context

    ctx = get_context()
    return ctx.apiToken


def get_workspace_url():
    """Get the workspace URL from spark config."""
    url = spark.conf.get("spark.databricks.workspaceUrl")
    if not url.startswith("https://"):
        url = "https://" + url
    return url


def get_current_user_email() -> str:
    """Get the current user's email from the workspace client."""
    w = get_workspace_client()
    return w.current_user.me().user_name


def get_debug_log_path() -> Path:
    """Get a stable path for debug logs under the user's home directory."""
    import uuid

    unique_id = uuid.uuid4().hex[:8]
    log_dir = Path.home() / "dbr_test_logs"
    log_dir.mkdir(parents=True, exist_ok=True)
    return log_dir / f"cloud_{unique_id}.log"


def copy_debug_log_to_workspace(local_log_path: Path) -> tuple[str, str]:
    """Copy debug log from driver filesystem to workspace and return (workspace_path, url)."""
    import uuid

    user_email = get_current_user_email()
    unique_id = uuid.uuid4().hex[:8]
    timestamp = datetime.now().strftime("%Y-%m-%d_%H-%M-%S")

    # Workspace FUSE path
    workspace_dir = f"/Workspace/Users/{user_email}/dbr_acceptance_tests"
    workspace_path = f"{workspace_dir}/debug-cloud-{timestamp}-{unique_id}.log"

    # Create directory and copy file
    os.makedirs(workspace_dir, exist_ok=True)

    with open(local_log_path, "r") as src:
        with open(workspace_path, "w") as dst:
            dst.write(src.read())

    # Build URL
    workspace_url = get_workspace_url()
    # Remove /Workspace prefix for the URL fragment
    url_path = workspace_path[len("/Workspace") :]
    debug_url = f"{workspace_url}#files{url_path}"

    return workspace_path, debug_url


# COMMAND ----------


def extract_archive(archive_path: str) -> Path:
    """Extract the archive to the home directory."""
    import uuid

    home = Path.home()
    unique_id = uuid.uuid4().hex[:8]
    extract_dir = home / f"acceptance_test_{unique_id}"

    extract_dir.mkdir(parents=True, exist_ok=True)

    print(f"Extracting archive from {archive_path} to {extract_dir}")

    # The archive_path may be a FUSE path (with /Workspace prefix) or an API path.
    # The workspace API expects paths without the /Workspace prefix.
    api_path = archive_path
    if api_path.startswith("/Workspace"):
        api_path = api_path[len("/Workspace") :]

    w = get_workspace_client()

    # Download the archive to a temp location (use unique name to avoid conflicts)
    temp_archive = home / f"archive_{unique_id}.tar.gz"
    with open(temp_archive, "wb") as f:
        resp = w.workspace.download(api_path)
        f.write(resp.read())

    # Extract the archive
    with tarfile.open(temp_archive, "r:gz") as tar:
        tar.extractall(path=extract_dir)

    # Clean up temp archive (use missing_ok for FUSE filesystem quirks)
    temp_archive.unlink(missing_ok=True)

    print(f"Archive extracted to {extract_dir}")
    return extract_dir


# COMMAND ----------


def setup_environment(extract_dir: Path) -> dict:
    """Set up the environment variables for running tests."""
    env = os.environ.copy()

    # Determine architecture
    machine = platform.machine().lower()
    if machine in ("x86_64", "amd64"):
        arch = "amd64"
    elif machine in ("aarch64", "arm64"):
        arch = "arm64"
    else:
        raise ValueError(f"Unsupported architecture: {machine}")

    print(f"Detected architecture: {arch}")

    # Set up PATH to include our binaries
    bin_dir = extract_dir / "bin" / arch
    go_bin_dir = bin_dir / "go" / "bin"

    path_entries = [
        str(go_bin_dir),
        str(bin_dir),
        env.get("PATH", ""),
    ]
    env["PATH"] = os.pathsep.join(path_entries)

    # Set GOROOT for the Go installation
    env["GOROOT"] = str(bin_dir / "go")

    # Set up authentication
    env["DATABRICKS_HOST"] = get_workspace_url()
    env["DATABRICKS_TOKEN"] = get_auth_token()

    # Disable telemetry for tests. Just for a marginally faster test execution.
    env["DATABRICKS_CLI_TELEMETRY_DISABLED"] = "true"

    return env


# COMMAND ----------


class TestResult:
    """Holds the result of running tests."""

    def __init__(self, return_code: int, stdout: str, stderr: str, debug_log_path: Path, debug_log_url: str):
        self.return_code = return_code
        self.stdout = stdout
        self.stderr = stderr
        self.debug_log_path = debug_log_path
        self.debug_log_url = debug_log_url


def run_tests(
    extract_dir: Path,
    env: dict,
    test_filter: str = "",
    cloud_env: str = "",
    test_default_warehouse_id: str = "",
    test_default_cluster_id: str = "",
    test_instance_pool_id: str = "",
    test_metastore_id: str = "",
    test_user_email: str = "",
    test_sp_application_id: str = "",
) -> TestResult:
    """Run CLI cloud acceptance tests."""
    cli_dir = extract_dir / "cli"

    # Create debug log file
    debug_log_path = get_debug_log_path()

    cmd = [
        "go",
        "test",
        "./acceptance",
        "-timeout",
        "14400s",
        "-v",
        "-workspace-tmp-dir",
    ]

    if test_filter:
        cmd.extend(["-run", f"^TestAccept/{test_filter}"])
    else:
        cmd.extend(["-run", "^TestAccept"])

    # Cloud tests: run with CLOUD_ENV set and workspace access
    env["CLOUD_ENV"] = cloud_env
    # Only tests using direct deployment are run on DBR.
    # Terraform based tests are out of scope for DBR.
    env["ENVFILTER"] = "DATABRICKS_BUNDLE_ENGINE=direct"

    if test_default_warehouse_id:
        env["TEST_DEFAULT_WAREHOUSE_ID"] = test_default_warehouse_id
    if test_default_cluster_id:
        env["TEST_DEFAULT_CLUSTER_ID"] = test_default_cluster_id
    if test_instance_pool_id:
        env["TEST_INSTANCE_POOL_ID"] = test_instance_pool_id
    if test_metastore_id:
        env["TEST_METASTORE_ID"] = test_metastore_id
    if test_user_email:
        env["TEST_USER_EMAIL"] = test_user_email
    if test_sp_application_id:
        env["TEST_SP_APPLICATION_ID"] = test_sp_application_id

    # Write header to debug log
    with open(debug_log_path, "w") as log_file:
        log_file.write(f"Command: {' '.join(cmd)}\n")
        log_file.write(f"Working directory: {cli_dir}\n")
        log_file.write(f"CLOUD_ENV: {cloud_env}\n")
        log_file.write(f"Test filter: {test_filter or '(all tests)'}\n")
        log_file.write(f"PATH: {env.get('PATH', '')[:200]}...\n")
        log_file.write(f"GOROOT: {env.get('GOROOT', '')}\n")
        log_file.write("\n" + "=" * 60 + "\n")
        log_file.write("TEST OUTPUT:\n")
        log_file.write("=" * 60 + "\n")
    print(f"Running command: {' '.join(cmd)}")
    print(f"Working directory: {cli_dir}")
    print(f"CLOUD_ENV: {cloud_env}")
    print(f"Test filter: {test_filter or '(all tests)'}")
    print(f"Go version: ", end="", flush=True)

    # Print Go version for debugging
    subprocess.run(["go", "version"], cwd=cli_dir, env=env)

    print(f"PATH: {env.get('PATH', '')[:200]}...")
    print(f"GOROOT: {env.get('GOROOT', '')}")

    print("\n" + "=" * 60)
    print("TEST OUTPUT (streaming):")
    print("=" * 60, flush=True)

    # Run tests with streaming output using Popen.
    # Merge stderr into stdout for simpler streaming.
    process = subprocess.Popen(
        cmd,
        cwd=cli_dir,
        env=env,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        bufsize=1,  # Line buffered
    )

    # Collect output while streaming it and write to debug log
    output_lines = []
    with open(debug_log_path, "a") as log_file:
        for line in process.stdout:
            print(line, end="", flush=True)
            output_lines.append(line)
            log_file.write(line)
            log_file.flush()

    process.wait()
    stdout = "".join(output_lines)

    # Write footer to debug log
    with open(debug_log_path, "a") as log_file:
        log_file.write("\n" + "=" * 60 + "\n")
        log_file.write(f"Tests finished with return code: {process.returncode}\n")
        log_file.write("=" * 60 + "\n")

    # Copy debug log to workspace for persistent access
    _, debug_log_url = copy_debug_log_to_workspace(debug_log_path)
    print(f"\nDebug log URL: {debug_log_url}")

    print("\n" + "=" * 60)
    print(f"Tests finished with return code: {process.returncode}")
    print("=" * 60)

    return TestResult(process.returncode, stdout, "", debug_log_path, debug_log_url)


# COMMAND ----------


def main():
    """Main entry point for the notebook."""
    # Get parameters from widgets
    dbutils.widgets.text("archive_path", "")
    dbutils.widgets.text("cloud_env", "")
    dbutils.widgets.text("test_filter", "")
    dbutils.widgets.text("test_default_warehouse_id", "")
    dbutils.widgets.text("test_default_cluster_id", "")
    dbutils.widgets.text("test_instance_pool_id", "")
    dbutils.widgets.text("test_metastore_id", "")
    dbutils.widgets.text("test_user_email", "")
    dbutils.widgets.text("test_sp_application_id", "")

    archive_path = dbutils.widgets.get("archive_path")
    cloud_env = dbutils.widgets.get("cloud_env")
    test_filter = dbutils.widgets.get("test_filter")
    test_default_warehouse_id = dbutils.widgets.get("test_default_warehouse_id")
    test_default_cluster_id = dbutils.widgets.get("test_default_cluster_id")
    test_instance_pool_id = dbutils.widgets.get("test_instance_pool_id")
    test_metastore_id = dbutils.widgets.get("test_metastore_id")
    test_user_email = dbutils.widgets.get("test_user_email")
    test_sp_application_id = dbutils.widgets.get("test_sp_application_id")

    if not archive_path:
        raise ValueError("archive_path parameter is required")
    if not cloud_env:
        raise ValueError("cloud_env parameter is required")

    print("=" * 60)
    print("DBR Cloud Test Runner")
    print("=" * 60)
    print(f"Archive path: {archive_path}")
    print(f"Cloud environment: {cloud_env}")
    print(f"Test filter: {test_filter or '(none)'}")
    print("=" * 60)

    # Extract the archive
    extract_dir = extract_archive(archive_path)

    # Set up the environment
    env = setup_environment(extract_dir)

    # Run the tests
    result = run_tests(
        extract_dir=extract_dir,
        env=env,
        test_filter=test_filter,
        cloud_env=cloud_env,
        test_default_warehouse_id=test_default_warehouse_id,
        test_default_cluster_id=test_default_cluster_id,
        test_instance_pool_id=test_instance_pool_id,
        test_metastore_id=test_metastore_id,
        test_user_email=test_user_email,
        test_sp_application_id=test_sp_application_id,
    )

    print("=" * 60)
    print(f"Tests completed with return code: {result.return_code}")
    print("=" * 60)

    if result.return_code != 0:
        # Print debug log location first for easy access
        print("\n" + "=" * 60)
        print("DEBUG LOG LOCATION:")
        print(f"  {result.debug_log_url}")
        print("=" * 60 + "\n")

        # Include relevant output in the exception for debugging
        stdout_preview = result.stdout[-100000:] if result.stdout else "(no stdout)"
        stderr_preview = result.stderr[-100000:] if result.stderr else "(no stderr)"
        error_msg = f"""Cloud tests failed with return code {result.return_code}

Debug log: {result.debug_log_url}

=== STDOUT (last 100000 chars) ===
{stdout_preview}

=== STDERR (last 100000 chars) ===
{stderr_preview}
"""
        raise Exception(error_msg)


# COMMAND ----------

if __name__ == "__main__":
    main()
