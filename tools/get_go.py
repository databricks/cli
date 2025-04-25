#!/usr/bin/env python3
import platform
import os
import time
import re
import shutil
import sys
import tarfile
import zipfile
from pathlib import Path
from urllib.request import urlretrieve


def get_go_version_from_mod(file_path):
    with open(file_path) as f:
        content = f.read()
    match = re.search(r"^toolchain\s+go(\d+\.\d+(\.\d+)?)", content, re.MULTILINE)
    if match:
        return match.group(1)


def detect_os_arch():
    system = platform.system().lower()
    machine = platform.machine().lower()
    arch = {"x86_64": "amd64", "amd64": "amd64", "aarch64": "arm64", "arm64": "arm64"}.get(machine, machine)
    print(f"Platform: {system=} {arch=} {machine=}")
    return system, arch


def download_and_extract(go_version, os_name, arch):
    ext = "zip" if os_name == "windows" else "tar.gz"
    filename = f"go{go_version}.{os_name}-{arch}.{ext}"
    url = f"https://go.dev/dl/{filename}"

    print(f"Downloading {url} to {filename}")
    start = time.time()
    urlretrieve(url, filename)
    took = time.time() - start
    size = os.stat(filename).st_size
    print(f"Downloaded {size / 1024.0 / 1024.0:.1f}MB in {took:.1f}s")

    try:
        os.mkdir("_go_binary")
    except FileExistsError:
        pass

    if ext == "zip":
        with zipfile.ZipFile(filename, "r") as zip_ref:
            zip_ref.extractall("_go_binary")
    else:
        with tarfile.open(filename, "r:gz") as tar_ref:
            tar_ref.extractall("_go_binary")


def main():
    go_version = get_go_version_from_mod("go.mod")
    assert go_version
    os_name, arch = detect_os_arch()
    download_and_extract(go_version, os_name, arch)


if __name__ == "__main__":
    main()
