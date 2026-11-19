import ctypes
import fcntl
import os
import signal
import subprocess
import sys
import time
from pathlib import Path


def stop(signum, _frame):
    raise SystemExit(128 + signum)


def reap_children():
    # SSH commands create their own sessions. Subreaping keeps their orphans
    # under this invocation so cleanup cannot reach another caller's processes.
    deadline = time.monotonic() + 5
    while True:
        try:
            while os.waitpid(-1, os.WNOHANG)[0]:
                pass
        except ChildProcessError:
            return
        for child in Path(f"/proc/self/task/{os.getpid()}/children").read_text().split():
            try:
                os.kill(int(child), signal.SIGKILL)
            except ProcessLookupError:
                pass
        if time.monotonic() >= deadline:
            raise RuntimeError("timed out stopping sandbox SSH child processes")
        time.sleep(0.01)


with open(os.path.join(sys.argv[1], "server.lock"), "a") as lock:
    fcntl.flock(lock, fcntl.LOCK_EX)
    libc = ctypes.CDLL(None, use_errno=True)
    # PR_SET_CHILD_SUBREAPER, also used by the compute SSH launcher.
    if libc.prctl(36, 1, 0, 0, 0) != 0:
        raise OSError(ctypes.get_errno(), "could not configure sandbox SSH child cleanup")
    signals = (signal.SIGTERM, signal.SIGHUP, signal.SIGINT)
    for signum in signals:
        signal.signal(signum, stop)
    server = None
    try:
        server = subprocess.Popen(sys.argv[2:], start_new_session=True)
        code = server.wait()
    finally:
        for signum in signals:
            signal.signal(signum, signal.SIG_IGN)
        if server is not None and server.poll() is None:
            try:
                os.killpg(server.pid, signal.SIGTERM)
            except ProcessLookupError:
                pass
            try:
                server.wait(timeout=2)
            except subprocess.TimeoutExpired:
                server.kill()
                server.wait()
        reap_children()
    sys.exit(code if code >= 0 else 128 - code)
