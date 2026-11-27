#!/usr/bin/env python3
"""
Run a command while holding the lock file LOCK, as an `aitools mcp add` or `remove` that is still running holds it.

Usage: with_lock.py LOCK COMMAND [ARG...]
"""

import os
import subprocess
import sys

lock, command = sys.argv[1], sys.argv[2:]
os.makedirs(os.path.dirname(lock), exist_ok=True)
with open(lock, "a+b") as f:
    if os.name == "nt":
        import msvcrt

        # The CLI locks the first byte of the file, so this locks the same byte.
        f.seek(0)
        msvcrt.locking(f.fileno(), msvcrt.LK_NBLCK, 1)
    else:
        import fcntl

        fcntl.flock(f, fcntl.LOCK_EX | fcntl.LOCK_NB)
    sys.exit(subprocess.call(command))
