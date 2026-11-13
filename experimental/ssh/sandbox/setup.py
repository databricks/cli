import fcntl
import hashlib
import json
import os
import pwd
import shutil
import subprocess
import sys
import tempfile
from datetime import datetime, timedelta, timezone
from pathlib import Path


def atomic_write(path, content):
    fd, temporary = tempfile.mkstemp(dir=path.parent)
    try:
        with os.fdopen(fd, "w") as output:
            output.write(content)
        os.replace(temporary, path)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)


request = json.loads(sys.argv[1])
sshd = shutil.which("sshd") or "/usr/sbin/sshd"
if not os.path.isfile(sshd):
    raise RuntimeError("Install openssh-server in the sandbox before connecting")
user = pwd.getpwuid(os.getuid()).pw_name
home = Path.home()
if home.stat().st_mode & 0o022:
    raise RuntimeError("SSH requires HOME not to be writable by group or other users")
parent = home / ".databricks"
parent.mkdir(mode=0o700, exist_ok=True)
# StrictModes checks every directory above authorized_keys, including existing ones.
parent.chmod(parent.stat().st_mode & ~0o022)
base = home / ".databricks" / "sandbox-ssh"
base.mkdir(mode=0o700, exist_ok=True)
base.chmod(0o700)
key = request["public_key"].strip()
if len(key.split()) != 2 or not key.startswith("ssh-ed25519 "):
    raise ValueError("Expected one Ed25519 public key")
key_dir = base / hashlib.sha256(key.encode()).hexdigest()
key_dir.mkdir(mode=0o700, exist_ok=True)
key_dir.chmod(0o700)
# Each client key has its own file, so renewal cannot discard another caller's entry.
authorized = key_dir / "authorized_keys"
# The server runs with TZ=UTC. Older OpenSSH releases reject a trailing Z.
expiry = (datetime.now(timezone.utc) + timedelta(hours=24)).strftime("%Y%m%d%H%M%S")
atomic_write(authorized, 'expiry-time="' + expiry + '" ' + key + "\n")
# Server keys belong to this persistent HOME, and are never returned to the client.
# Serialize generation so simultaneous connections share the same server identity.
with open(base / "host-key.lock", "a") as lock:
    fcntl.flock(lock, fcntl.LOCK_EX)
    host_key = base / "host_ed25519"
    if not host_key.exists():
        subprocess.run(["ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", str(host_key)], check=True)
    host_public = subprocess.check_output(["ssh-keygen", "-y", "-f", str(host_key)], universal_newlines=True).strip()
session = Path(tempfile.mkdtemp(prefix="session-", dir=base))
config = session / "sshd_config"


def quote(value):
    value = str(value)
    if any(c in value for c in ('"', "\n", "\r", "\\")):
        raise ValueError("Unsupported character in SSH configuration path")
    return '"' + value + '"'


atomic_write(
    config,
    "\n".join(
        [
            "HostKey " + quote(host_key),
            "AuthorizedKeysFile " + quote(authorized),
            "PubkeyAuthentication yes",
            "PasswordAuthentication no",
            "KbdInteractiveAuthentication no",
            # OpenSSH requires root for PAM; this server runs as sandbox-agent.
            "UsePAM no",
            "AllowUsers " + user,
            "AllowTcpForwarding yes",
            "ClientAliveInterval 30",
            "Subsystem sftp internal-sftp",
            "PidFile " + quote(session / "sshd.pid"),
        ]
    )
    + "\n",
)
print(
    json.dumps(
        {
            "home": str(home),
            "config": str(config),
            "sshd": sshd,
            "user": user,
            "host_key": host_public,
            "architecture": os.uname().machine,
            "session_dir": str(session),
        }
    )
)
