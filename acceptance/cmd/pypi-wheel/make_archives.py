#!/usr/bin/env python3
"""Fake goreleaser output: every platform archive holds the given CLI binary."""

import sys
import tarfile
import zipfile
from pathlib import Path

cli, out = Path(sys.argv[1]), Path(sys.argv[2])
out.mkdir()
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
# Only the file name is used: it carries the version.
(out / "databricks_cli_1.2.3_SHA256SUMS").touch()
