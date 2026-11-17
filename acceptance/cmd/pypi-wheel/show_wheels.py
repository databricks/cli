#!/usr/bin/env python3
"""Print what each wheel in a directory installs, in a platform-independent form.

RECORD is not printed: its hashes cover the CLI binary, which differs per build.
"""

import sys
import zipfile
from pathlib import Path

wheels = sorted(Path(sys.argv[1]).glob("*.whl"))
for whl in wheels:
    print(f"== {whl.name}")
    with zipfile.ZipFile(whl) as z:
        for info in z.infolist():
            print(f"{info.external_attr >> 16:o} {info.filename}")
        wheel_file = next(n for n in z.namelist() if n.endswith(".dist-info/WHEEL"))
        print(z.read(wheel_file).decode(), end="")

with zipfile.ZipFile(wheels[0]) as z:
    metadata_file = next(n for n in z.namelist() if n.endswith(".dist-info/METADATA"))
    print("== METADATA (same in every wheel)")
    print(z.read(metadata_file).decode(), end="")
