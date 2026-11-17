#!/usr/bin/env python3
"""Fake goreleaser output for the CLI wheel builder.

Only the archive for the platform running the test holds the CLI binary. The
others hold a placeholder that cannot be executed, so running the installed
`databricks` fails if uv picks another platform's wheel.
"""

import io
import platform
import sys
import tarfile
import zipfile
from pathlib import Path

GOOS = {"Linux": "linux", "Darwin": "darwin", "Windows": "windows"}
GOARCH = {"x86_64": "amd64", "amd64": "amd64", "aarch64": "arm64", "arm64": "arm64"}

cli, out = Path(sys.argv[1]), Path(sys.argv[2])
host = (GOOS[platform.system()], GOARCH[platform.machine().lower()])
placeholder = b"placeholder for a binary of another platform\n"

out.mkdir()
for goos in ("linux", "darwin", "windows"):
    for goarch in ("amd64", "arm64"):
        data = cli.read_bytes() if (goos, goarch) == host else placeholder
        if goos == "windows":
            path = out / f"databricks_cli_1.2.3_{goos}_{goarch}.zip"
            with zipfile.ZipFile(path, "w") as z:
                z.writestr("databricks.exe", data)
        else:
            path = out / f"databricks_cli_1.2.3_{goos}_{goarch}.tar.gz"
            # Compression only costs test time: the builder reads the archive either way.
            with tarfile.open(path, "w:gz", compresslevel=0) as t:
                info = tarfile.TarInfo("databricks")
                info.size = len(data)
                info.mode = 0o755
                t.addfile(info, io.BytesIO(data))
# Only the file name is used: it carries the version.
(out / "databricks_cli_1.2.3_SHA256SUMS").touch()
