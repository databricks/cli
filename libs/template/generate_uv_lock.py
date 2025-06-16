#!/usr/bin/env python3
import sys
import os
import shutil
import subprocess
import tempfile
from contextlib import chdir  # Python 3.11+
from pathlib import Path


def generate_uv_lock(path):
    path = Path(path)
    data = path.read_text()
    data = data.replace("{{.project_name}}", "project_name")

    with tempfile.TemporaryDirectory() as tmpdir:
        tmpdir = Path(tmpdir)
        with open(tmpdir / "pyproject.toml", "w") as fobj:
            fobj.write(data)

        print(data)

        with chdir(tmpdir):
            subprocess.run(["uv", "sync", "--no-install-project", "--python", "3.11"], check=True)

        shutil.copyfile(tmpdir / "uv.lock", path.parent / "uv.lock")


def main():
    for arg in sys.argv[1:]:
        generate_uv_lock(arg)


if __name__ == "__main__":
    main()
