import base64
import json
import os
import re
import signal
import socket
import subprocess
import tempfile
import time
from pathlib import Path

cli = os.environ["CLI"]
environment = {
    key: value
    for key, value in os.environ.items()
    if not key.startswith(("DATABRICKS_", "ARM_", "AZURE_", "AWS_", "GOOGLE_"))
}
with tempfile.TemporaryDirectory() as temporary:
    directory = Path(temporary)
    environment["HOME"] = temporary
    environment["DATABRICKS_CONFIG_FILE"] = str(directory / "missing-config")
    child_pid = directory / "child.pid"
    sshd = directory / "sshd"
    # A long-running child exercises cancellation without requiring system sshd.
    sshd.write_text(
        '#!/bin/sh\nif [ "$1" = "-t" ]; then exit 0; fi\n'
        f'echo $$ > "{child_pid}.tmp"\n'
        f'mv "{child_pid}.tmp" "{child_pid}"\n'
        "exec sleep 300\n"
    )
    sshd.chmod(0o700)
    help_result = subprocess.run(
        [cli, "sandbox", "--help"], env=environment, capture_output=True, text=True, check=True
    )
    assert not re.search(r"^\s+server\s", help_result.stdout, re.MULTILINE)
    print("Server remains hidden from sandbox help")
    process = subprocess.Popen(
        [cli, "sandbox", "server", "--port", "0", "--sshd", str(sshd), "--sshd-config", "prepared-config"],
        env=environment,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        start_new_session=True,
    )
    try:
        ready = json.loads(process.stdout.readline())
        assert 10000 <= ready["port"] <= 65535
        assert ready["token"]
        print("Server starts without workspace credentials")
        with socket.create_connection(("127.0.0.1", ready["port"]), timeout=5) as connection:
            key = base64.b64encode(os.urandom(16)).decode()
            connection.sendall(
                (
                    "GET /ssh?id=acceptance HTTP/1.1\r\nHost: localhost\r\n"
                    "Upgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\n"
                    f"Sec-WebSocket-Key: {key}\r\nX-Sandbox-SSH-Token: {ready['token']}\r\n\r\n"
                ).encode()
            )
            assert connection.recv(4096).startswith(b"HTTP/1.1 101")
            deadline = time.monotonic() + 5
            while not child_pid.exists():
                assert time.monotonic() < deadline, "sshd child did not start"
                time.sleep(0.01)
            pid = int(child_pid.read_text())
            os.kill(pid, 0)
            assert process.poll() is None, "server exited before SIGTERM"
            process.send_signal(signal.SIGTERM)
            assert process.wait(timeout=10) == 1  # context cancellation, rather than signal termination
            try:
                os.kill(pid, 0)
            except ProcessLookupError:
                pass
            else:
                raise AssertionError("sshd child survived server shutdown")
        print("SIGTERM stops the server and its SSH child")
    finally:
        if process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
        try:
            os.killpg(process.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
