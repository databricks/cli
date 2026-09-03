import collections
import ctypes
import ctypes.util
import os
import platform
import signal
import subprocess
import sys
import time

from databricks.sdk import WorkspaceClient
from dbruntime.databricks_repl_context import get_context

SSH_TUNNEL_BASENAME = "databricks_cli"

# How often the linger loop re-checks for detached processes still holding the run open.
LINGER_POLL_SECONDS = 15

# Exit statuses collected by the SIGCHLD subreaper handler, keyed by pid. The handler
# can reap the server subprocess before Popen.wait() does, in which case Popen would
# report exit code 0; this map preserves the real status.
reaped_statuses = {}

dbutils.widgets.text("version", "")
dbutils.widgets.text("secretScopeName", "")
dbutils.widgets.text("authorizedKeySecretName", "")
dbutils.widgets.text("maxClients", "10")
dbutils.widgets.text("shutdownDelay", "10m")
dbutils.widgets.text("sessionId", "")
dbutils.widgets.text("serverless", "false")
dbutils.widgets.text("usagePolicyId", "")
dbutils.widgets.text("keepDetachedForSeconds", "0")


def cleanup():
    # Terminate an SSH server left behind by a previous, hard-killed run. The pattern matches
    # the server's own argv rather than the CLI binary name alone, so detached work that
    # happens to run the CLI is not swept away with it.
    subprocess.run(["pkill", "-f", f"{SSH_TUNNEL_BASENAME}.*ssh server --cluster="], check=False)


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
                    reaped_statuses[pid] = status
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


def kill_server_group(server_pgid):
    """Terminate the SSH server's own process group.

    That group holds exactly what the tunnel started: the server and the sshd processes it
    spawned per connection. A process that deliberately left the group - which is what tmux,
    setsid and disown do - is not in it, so detached work survives this. Killing by parentage
    instead (pkill -P) would sweep those too, because PR_SET_CHILD_SUBREAPER makes this
    process adopt every orphan in the session.
    """
    try:
        os.killpg(server_pgid, signal.SIGTERM)
        print(f"Terminated SSH server process group {server_pgid}")
    except ProcessLookupError:
        print(f"SSH server process group {server_pgid} is already gone")


def detached_descendants(server_pgid):
    """Adopted children of this process that are outside the SSH server's process group.

    PR_SET_CHILD_SUBREAPER makes every orphan in the session reparent to this process, so
    work that detached itself - tmux, setsid, disown, a plain background command - resurfaces
    here as a direct child. The server's own sshd children stay in its group and are excluded.

    Reads /proc directly, mirroring detachedDescendants in internal/server/descendants.go.
    Asking ps for this process's children cannot work: ps is one of them, and subprocess.run
    leaves it in this process's group, so it matches its own query on every poll and the
    survivor list is never empty.
    """
    self_pid = os.getpid()
    survivors = []
    for entry in os.listdir("/proc"):
        if not entry.isdigit():
            continue
        try:
            with open(f"/proc/{entry}/stat") as stat_file:
                # Split on the last ')' rather than from the left: the comm field before it
                # can contain both spaces and parentheses. state, ppid and pgrp follow it.
                state, ppid, pgrp = stat_file.read().rsplit(")", 1)[1].split()[:3]
        except OSError:
            # The process exited while we were walking /proc.
            continue
        if ppid != str(self_pid) or pgrp == str(server_pgid):
            continue
        # Zombies are already dead; the SIGCHLD handler collects them.
        if state.startswith("Z"):
            continue
        survivors.append(entry)
    return sorted(survivors, key=int)


def wait_for_detached_descendants(server_pgid, timeout_seconds):
    """Hold the notebook open while detached work is still running.

    WSFS authorises an I/O by walking the live process tree for a registered ancestor, and
    this process is the registered one. Returning while detached work is still alive would
    reparent it to PID 1, outside the registered subtree, and silently strip its /Workspace
    and /Volumes access - trading a visible failure for an invisible one. Holding the run
    open instead keeps the cluster from auto-terminating, which is why it is opt-in and
    bounded by --keep-detached-for.
    """
    deadline = time.monotonic() + timeout_seconds
    while True:
        survivors = detached_descendants(server_pgid)
        if not survivors:
            print("No detached processes left, releasing the run", flush=True)
            return
        if time.monotonic() > deadline:
            print(
                f"Reached the --keep-detached-for limit of {timeout_seconds}s with "
                f"{len(survivors)} detached process(es) still running: {','.join(survivors)}. "
                "Releasing the run; they lose /Workspace and /Volumes access from here.",
                flush=True,
            )
            return
        print(
            f"Holding the run open for {len(survivors)} detached process(es): {','.join(survivors)}",
            flush=True,
        )
        time.sleep(LINGER_POLL_SECONDS)


def run_ssh_server():
    ctx = get_context()

    # Old DBRs require explicit WorkspaceClient arguments
    try:
        client = WorkspaceClient()
    except Exception:
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
    os.environ["DATABRICKS_REMOTE_ENV"] = "1"
    python_path = os.path.dirname(sys.executable)
    os.environ["PATH"] = f"{python_path}:{os.environ['PATH']}"
    if os.environ.get("VIRTUAL_ENV") is None:
        os.environ["VIRTUAL_ENV"] = sys.executable

    secrets_scope = dbutils.widgets.get("secretScopeName")
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
    session_id = dbutils.widgets.get("sessionId")
    if not session_id:
        raise RuntimeError("Session ID is required. Please provide it using the 'sessionId' widget.")
    serverless = dbutils.widgets.get("serverless")
    usage_policy_id = dbutils.widgets.get("usagePolicyId")
    keep_detached_for_seconds = int(dbutils.widgets.get("keepDetachedForSeconds") or 0)

    # Mark this process's WSFS command origin so workspace-file activity from the
    # remote SSH session is attributable
    try:
        with open("/Workspace/.proc/self/metadata/command_origin", "w") as command_origin_file:
            command_origin_file.write("RemoteSshServer")
    except OSError as e:
        print(f"Could not set WSFS command origin: {e}")

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

    server_args = [
        binary_path,
        "ssh",
        "server",
        f"--cluster={ctx.clusterId}",
        f"--session-id={session_id}",
        f"--serverless={serverless}",
        f"--secret-scope-name={secrets_scope}",
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
    ]

    # Recorded in the server's metadata.json so reconnects can match the usage policy.
    if usage_policy_id:
        server_args.append(f"--usage-policy-id={usage_policy_id}")

    # The server does not linger itself; it uses this to persist the mode for reconnects and
    # to warn about detached work it is about to leave behind when the mode is off.
    if keep_detached_for_seconds > 0:
        server_args.append(f"--keep-detached-for={keep_detached_for_seconds}s")

    # Tee the server output instead of inheriting stdout: the run-page logs remain the only
    # place to debug a RUNNING server, but on failure we attach the log tail to the exception
    # so "ssh connect" can print it (the Jobs run-output API has no stdout logs for notebook tasks).
    tail = collections.deque(maxlen=100)
    proc = subprocess.Popen(
        server_args,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        errors="replace",
        # Make the server a session and process group leader, so teardown can target exactly
        # the processes the tunnel started. See kill_server_group.
        start_new_session=True,
    )
    # The server leads the new group, so the group id is its pid. Recorded here because the
    # pid may already have been reaped by the time we tear the group down.
    server_pgid = proc.pid
    try:
        for line in proc.stdout:
            # flush so the run-page logs stay live while the server is running
            print(line, end="", flush=True)
            tail.append(line)
        returncode = proc.wait()
        # The SIGCHLD subreaper handler may have collected the server first; Popen reports that as 0.
        if proc.pid in reaped_statuses:
            returncode = os.waitstatus_to_exitcode(reaped_statuses[proc.pid])
        if returncode == -signal.SIGTERM:
            # A newer session's bootstrap terminates a server already running on this cluster
            # (see cleanup), which a reconnect asking for a different --keep-detached-for now
            # reaches on a normal path. That is a handover, not a failure of this run, so it
            # must not mark the run FAILED - the linger below still runs.
            print("SSH server was terminated, most likely by a newer session on this cluster", flush=True)
        elif returncode != 0:
            # The tail size matches maxRunFailureTraceBytes, the cap the client prints to the terminal.
            raise RuntimeError(f"SSH server exited with code {returncode}. Last server logs:\n" + "".join(tail)[-2000:])
    finally:
        # Always reap the server and the sshd children it spawned; they are the only things
        # in its process group. What happens to work that left that group depends on the mode:
        # keep it and hold the run open as its WSFS anchor, or sweep it as we always have.
        # Narrowing the sweep without lingering would leave survivors alive but cut off from
        # /Workspace, which is a worse failure than the one it fixes.
        kill_server_group(server_pgid)
        if keep_detached_for_seconds > 0:
            wait_for_detached_descendants(server_pgid, keep_detached_for_seconds)
        else:
            kill_all_children()


if __name__ == "__main__":
    cleanup()
    setup_subreaper()
    run_ssh_server()
