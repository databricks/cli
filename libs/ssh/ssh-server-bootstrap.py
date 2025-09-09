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

SSH_TUNNEL_BASENAME = "databricks_cli_linux"

dbutils.widgets.text("version", "")
dbutils.widgets.text("secretsScope", "")
dbutils.widgets.text("publicKeySecretName", "")
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
                if pid == 0:
                    break
                print(f"Reaped child {pid} with status {status}")
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
    # Start the SSH tunnel
    ctx = get_context()
    client = WorkspaceClient()
    workspace_url = ctx.workspaceUrl or client.config.host or spark.conf.get("spark.databricks.workspaceUrl")
    user_name = client.current_user.me().user_name

    os.environ["DATABRICKS_HOST"] = workspace_url if workspace_url.startswith("https://") else f"https://{workspace_url}"
    os.environ["DATABRICKS_TOKEN"] = ctx.apiToken
    os.environ["DATABRICKS_CLUSTER_ID"] = ctx.clusterId
    os.environ["DATABRICKS_VIRTUAL_ENV"] = sys.executable
    python_path = os.path.dirname(sys.executable)
    os.environ["PATH"] = f"{python_path}:{os.environ['PATH']}"
    if os.environ.get("VIRTUAL_ENV") is None:
        os.environ["VIRTUAL_ENV"] = sys.executable

    secrets_scope = dbutils.widgets.get("secretsScope")
    if not secrets_scope:
        raise RuntimeError("Secrets scope is required. Please provide it using the 'secretsScope' widget.")

    public_key_secret_name = dbutils.widgets.get("publicKeySecretName")
    if not public_key_secret_name:
        raise RuntimeError("Public key secret name is required. Please provide it using the 'publicKeySecretName' widget.")

    public_key = dbutils.secrets.get(scope=secrets_scope, key=public_key_secret_name)
    if not public_key:
        raise RuntimeError(f"Public key secret '{public_key_secret_name}' in scope '{secrets_scope}' is empty.")

    os.environ["PUBLIC_SSH_KEY"] = public_key

    version = dbutils.widgets.get("version")
    if not version:
        raise RuntimeError("Version is required. Please provide it using the 'version' widget.")

    shutdown_delay = dbutils.widgets.get("shutdownDelay")
    max_clients = dbutils.widgets.get("maxClients")

    arch = platform.machine()
    if arch == "x86_64":
        cli_arch = "amd64"
    elif arch == "aarch64" or arch == "arm64":
        cli_arch = "arm64"
    else:
        raise RuntimeError(f"Unsupported architecture: {arch}")

    binary_path = f"/Workspace/Users/{user_name}/.databricks/ssh-tunnel/{version}/{SSH_TUNNEL_BASENAME}_{cli_arch}/databricks"
    try:
        subprocess.run(
            [
                binary_path,
                "ssh",
                "server",
                f"--cluster={ctx.clusterId}",
                f"--max-clients={max_clients}",
                f"--shutdown-delay={shutdown_delay}",
                f"--version={version}",
                # "info" has enough verbosity for debugging purposes, and "debug" log level prints too much (including secrets)
                "--log-level=info",
                # To get the server logs:
                # 1. Get a job run id from the "databricks ssh connect" output
                # 2. Run "databricks jobs get-run <id>" and open a run_page_url
                # TODO: file with log rotation
                "--log-file=stdout",
            ],
            check=True,
        )
    finally:
        kill_all_children()


if __name__ == "__main__":
    cleanup()
    setup_subreaper()
    setup_exit_handler()
    run_ssh_server()
