import fcntl
import hashlib
import json
import os
import sys
from pathlib import Path

operation, target = sys.argv[1:3]
target = Path(target)
target.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
with open(target.parent / "install.lock", "a") as lock:
    fcntl.flock(lock, fcntl.LOCK_EX)
    cached = target.is_file() and os.access(target, os.X_OK)
    if operation == "install":
        temporary = Path(sys.argv[3])
        try:
            with temporary.open("rb") as source:
                digest = (
                    hashlib.file_digest(source, "sha256").hexdigest()
                    if hasattr(hashlib, "file_digest")
                    else hashlib.sha256(source.read()).hexdigest()
                )
            if digest != sys.argv[4]:
                raise ValueError("Uploaded CLI executable checksum mismatch")
            if not cached:
                temporary.chmod(0o700)
                os.replace(temporary, target)
            cached = True
        finally:
            if temporary.exists():
                temporary.unlink()
    elif operation != "check":
        raise ValueError("Unknown executable cache operation")
    print(json.dumps({"cached": cached}))
