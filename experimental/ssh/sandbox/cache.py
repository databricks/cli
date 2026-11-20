import fcntl
import hashlib
import json
import os
import re
import shutil
import sys
import time
from pathlib import Path

MAX_CACHED_RELEASES = 8
MAX_CACHE_AGE = 30 * 86400

operation, target, session = sys.argv[1:4]
target = Path(target)
session = Path(session)
releases = target.parent.parent
releases.mkdir(mode=0o700, parents=True, exist_ok=True)
# Keep this lock outside the pruned directories so concurrent starts share it.
with open(releases / "install.lock", "a") as lock:
    fcntl.flock(lock, fcntl.LOCK_EX)
    cached = target.is_file() and os.access(target, os.X_OK)
    if operation == "install":
        temporary = Path(sys.argv[4])
        try:
            with temporary.open("rb") as source:
                digest = (
                    hashlib.file_digest(source, "sha256").hexdigest()
                    if hasattr(hashlib, "file_digest")
                    else hashlib.sha256(source.read()).hexdigest()
                )
            if digest != sys.argv[5]:
                raise ValueError("Uploaded CLI executable checksum mismatch")
            if not cached:
                target.parent.mkdir(mode=0o700, exist_ok=True)
                temporary.chmod(0o700)
                os.replace(temporary, target)
            cached = True
        finally:
            if temporary.exists():
                temporary.unlink()
    elif operation != "check":
        raise ValueError("Unknown executable cache operation")
    if cached:
        # Pin before releasing the lock, including the gap before server exec.
        # Session cleanup already preserves directories with a live server lease.
        lease = session / target.parent.name
        if not lease.exists():
            os.link(target, lease)
        os.utime(target.parent, None)
    entries = [
        entry
        for entry in releases.iterdir()
        if re.fullmatch(r"(?:amd64|arm64)-[0-9a-f]{64}", entry.name) and not entry.is_symlink() and entry.is_dir()
    ]
    # Leased releases may exceed the budget until their sessions are cleaned up.
    retained = int(target.parent.exists())
    for entry in sorted(entries, key=lambda entry: entry.stat().st_mtime, reverse=True):
        executable = entry / "databricks"
        if entry == target.parent or (executable.exists() and executable.stat().st_nlink > 1):
            continue
        if entry.stat().st_mtime >= time.time() - MAX_CACHE_AGE and retained < MAX_CACHED_RELEASES:
            retained += 1
            continue
        shutil.rmtree(entry)
    print(json.dumps({"cached": cached}))
