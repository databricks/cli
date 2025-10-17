from dbruntime.databricks_repl_context import get_context
from databricks.sdk import WorkspaceClient
import os
import sys
import subprocess
import ctypes
import ctypes.util
import signal
import atexit
import platform
import time

SSH_TUNNEL_BASENAME = "databricks_cli"

dbutils.widgets.text("version", "")
dbutils.widgets.text("keysSecretScopeName", "")
dbutils.widgets.text("authorizedKeySecretName", "")
dbutils.widgets.text("maxClients", "10")
dbutils.widgets.text("shutdownDelay", "10m")


def cleanup():
    subprocess.run(["pkill", "-f", SSH_TUNNEL_BASENAME], check=False)


def setup_subreaper():
    # Mark itself as a child subreaper to handle orphaned processes,
    # preventing them from re-parenting to PID 1 and losing access to wsfs and dbfs.
    libc = ctypes.CDLL(ctypes.util.find_library("c"))
    PR_SET_CHILD_SUBREAPER = 36
    libc.prctl(PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0)

    def sigchld_handler(signum, frame):
        try:
            while True:
                # -1 means any child, WNOHANG means don't block
                pid, status = os.waitpid(-1, os.WNOHANG)
                if pid > 0:
                    print(f"Reaped child {pid} with status {status}")
                elif pid == 0:
                    print("No child has changed state")
                    break
                else:
                    print("Error while reaping child processes")
                    break
        except ChildProcessError:
            pass

    # Reap all dead children to prevent zombie processes.
    # We have to do it manually now, since we are a child subreaper.
    signal.signal(signal.SIGCHLD, sigchld_handler)


def kill_all_children():
    try:
        current_pid = os.getpid()
        while True:
            result = subprocess.run(["pgrep", "-P", str(current_pid)], capture_output=True, text=True, check=False)
            if result.returncode != 0 or not result.stdout.strip():
                break
            subprocess.run(["pkill", "-P", str(current_pid)], check=False)
            time.sleep(0.1)
        print("All descendant processes terminated")
    except Exception as e:
        print(f"Error while killing child processes: {e}")


def setup_exit_handler():
    # Register the cleanup function to be called when the script exits
    atexit.register(kill_all_children)


def run_ssh_server():
    ctx = get_context()

    # Old DBRs require explicit WorkspaceClient arguments
    try:
        client = WorkspaceClient()
    except Exception as e:
        client = WorkspaceClient(
            host=ctx.workspaceUrl or spark.conf.get("spark.databricks.workspaceUrl"), token=ctx.apiToken
        )

    workspace_url = ctx.workspaceUrl or client.config.host or spark.conf.get("spark.databricks.workspaceUrl")
    user_name = client.current_user.me().user_name

    os.environ["DATABRICKS_HOST"] = (
        workspace_url if workspace_url.startswith("https://") else f"https://{workspace_url}"
    )
    os.environ["DATABRICKS_TOKEN"] = ctx.apiToken
    os.environ["DATABRICKS_CLUSTER_ID"] = ctx.clusterId
    os.environ["DATABRICKS_VIRTUAL_ENV"] = sys.executable
    python_path = os.path.dirname(sys.executable)
    os.environ["PATH"] = f"{python_path}:{os.environ['PATH']}"
    if os.environ.get("VIRTUAL_ENV") is None:
        os.environ["VIRTUAL_ENV"] = sys.executable

    secrets_scope = dbutils.widgets.get("keysSecretScopeName")
    if not secrets_scope:
        raise RuntimeError("Secrets scope is required. Please provide it using the 'keysSecretScopeName' widget.")

    public_key_secret_name = dbutils.widgets.get("authorizedKeySecretName")
    if not public_key_secret_name:
        raise RuntimeError(
            "Public key secret name is required. Please provide it using the 'authorizedKeySecretName' widget."
        )

    version = dbutils.widgets.get("version")
    if not version:
        raise RuntimeError("Version is required. Please provide it using the 'version' widget.")

    shutdown_delay = dbutils.widgets.get("shutdownDelay")
    max_clients = dbutils.widgets.get("maxClients")

    arch = platform.machine()
    if arch == "x86_64":
        cli_arch = "linux_amd64"
    elif arch == "aarch64" or arch == "arm64":
        cli_arch = "linux_arm64"
    else:
        raise RuntimeError(f"Unsupported architecture: {arch}")

    if version.find("dev") != -1:
        cli_name = f"{SSH_TUNNEL_BASENAME}_{cli_arch}"
    else:
        cli_name = f"{SSH_TUNNEL_BASENAME}_{version}_{cli_arch}"

    binary_path = f"/Workspace/Users/{user_name}/.databricks/ssh-tunnel/{version}/{cli_name}/databricks"

    try:
        p = subprocess.run(
            [
                binary_path,
                "ssh",
                "server",
                f"--cluster={ctx.clusterId}",
                f"--keys-secret-scope-name={secrets_scope}",
                f"--authorized-key-secret-name={public_key_secret_name}",
                f"--max-clients={max_clients}",
                f"--shutdown-delay={shutdown_delay}",
                f"--version={version}",
                # "info" has enough verbosity for debugging purposes, and "debug" log level prints too much (including secrets)
                "--log-level=info",
                "--log-format=json",
                # To get the server logs:
                # 1. Get a job run id from the "databricks ssh connect" output
                # 2. Run "databricks jobs get-run <id>" and open a run_page_url
                # TODO: file with log rotation
                "--log-file=stdout",
            ],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=True,
        )
        kill_all_children()
        dbutils.notebook.exit(
            "stdout:\n"
            + p.stdout.decode(errors="replace")
            + "\n\nstderr:\n"
            + p.stderr.decode(errors="replace")
        )
    finally:
        kill_all_children()


if __name__ == "__main__":
    cleanup()
    setup_subreaper()
    setup_exit_handler()
    run_ssh_server()
