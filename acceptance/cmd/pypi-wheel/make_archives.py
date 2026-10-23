#!/usr/bin/env python3
"""Fake goreleaser output: every platform archive holds the given CLI binary."""

import hashlib
import shutil
import sys
import tarfile
import zipfile
from pathlib import Path

cli, out = Path(sys.argv[1]), Path(sys.argv[2])
out.mkdir()
sums = []
for goos in ("linux", "darwin", "windows"):
    for goarch in ("amd64", "arm64"):
        if goos == "windows":
            path = out / f"databricks_cli_1.2.3_{goos}_{goarch}.zip"
            with zipfile.ZipFile(path, "w") as z:
                z.write(cli, "databricks.exe")
        else:
            path = out / f"databricks_cli_1.2.3_{goos}_{goarch}.tar.gz"
            with tarfile.open(path, "w:gz") as t:
                t.add(cli, "databricks")
        sums.append(f"{hashlib.sha256(path.read_bytes()).hexdigest()}  {path.name}\n")
(out / "databricks_cli_1.2.3_SHA256SUMS").write_text("".join(sums))
