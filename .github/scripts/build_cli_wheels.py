#!/usr/bin/env python3
"""Package the released Databricks CLI binaries as PyPI wheels.

Input is the directory of the `cli` release-build artifact: the goreleaser
archives (databricks_cli_*_<os>_<arch>.{tar.gz,zip}) and the SHA256SUMS file.
The binaries are taken from the archives as-is, so the signed Windows binaries
are what ends up in the wheels.

Each wheel is py3-none-<platform>: platform-specific but independent of the
Python version. It contains only the binary, installed to the environment's
scripts directory (bin/ or Scripts/) via the wheel `.data/scripts` mechanism,
and no Python package, so `databricks` and `uvx databricks` run the binary
directly without a Python bootstrapper.

Usage: build_cli_wheels.py <cli-artifact-dir> <output-dir> [license-file]
"""

import base64
import hashlib
import io
import re
import sys
import tarfile
import zipfile
from pathlib import Path

DIST_NAME = "databricks"
SUMMARY = "Databricks CLI"
DESCRIPTION = """# Databricks CLI

Command-line interface for Databricks. This package installs the prebuilt `databricks` binary.

Run it with `uvx databricks` or `pipx run databricks`, or install it with `uv tool install databricks`.

Documentation: https://docs.databricks.com/dev-tools/cli/
Source: https://github.com/databricks/cli
"""

# (goreleaser os, goreleaser arch) -> wheel platform tag. Compressed tag sets
# (dot-separated) make one wheel serve glibc and musl Linux: the binary is
# statically linked (CGO_ENABLED=0).
PLATFORMS = {
    ("linux", "amd64"): "manylinux_2_17_x86_64.manylinux2014_x86_64.musllinux_1_1_x86_64",
    ("linux", "arm64"): "manylinux_2_17_aarch64.manylinux2014_aarch64.musllinux_1_1_aarch64",
    ("darwin", "amd64"): "macosx_11_0_x86_64",
    ("darwin", "arm64"): "macosx_11_0_arm64",
    ("windows", "amd64"): "win_amd64",
    ("windows", "arm64"): "win_arm64",
}

# Fixed timestamp so rebuilding from the same artifact yields identical wheels.
ZIP_DATE_TIME = (1980, 1, 1, 0, 0, 0)


def read_checksums(artifact_dir: Path) -> tuple[str, dict[str, str]]:
    """Return (cli_version, {file name: sha256}) from the SHA256SUMS file."""
    sums_files = list(artifact_dir.glob("databricks_cli_*_SHA256SUMS"))
    if len(sums_files) != 1:
        raise SystemExit(f"expected exactly one databricks_cli_*_SHA256SUMS in {artifact_dir}, found {sums_files}")
    version = sums_files[0].name[len("databricks_cli_") : -len("_SHA256SUMS")]
    sums = {}
    for line in sums_files[0].read_text().splitlines():
        digest, _, name = line.partition("  ")
        sums[name.strip()] = digest
    return version, sums


def pep440(cli_version: str) -> str:
    """Convert a goreleaser version to PEP 440.

    Releases (1.23.0) are unchanged. Snapshots (1.23.1-dev+abc123) become
    1.23.1.dev0+abc123.
    """
    m = re.fullmatch(r"(\d+\.\d+\.\d+)(?:-dev(?:\+([0-9A-Za-z.]+))?)?", cli_version)
    if not m:
        raise SystemExit(f"unsupported CLI version {cli_version!r}")
    base, local = m.groups()
    if cli_version == base:
        return base
    return f"{base}.dev0" + (f"+{local}" if local else "")


def find_archive(artifact_dir: Path, goos: str, goarch: str) -> Path:
    ext = "zip" if goos == "windows" else "tar.gz"
    matches = list(artifact_dir.glob(f"databricks_cli_*{goos}_{goarch}.{ext}"))
    if len(matches) != 1:
        raise SystemExit(f"expected exactly one {goos}/{goarch} .{ext} archive in {artifact_dir}, found {matches}")
    return matches[0]


def extract_binary(archive: Path, binary_name: str) -> bytes:
    if archive.name.endswith(".zip"):
        with zipfile.ZipFile(archive) as z:
            return z.read(binary_name)
    with tarfile.open(archive) as t:
        f = t.extractfile(binary_name)
        if f is None:
            raise SystemExit(f"{binary_name} is not a regular file in {archive}")
        return f.read()


def record_hash(data: bytes) -> str:
    digest = base64.urlsafe_b64encode(hashlib.sha256(data).digest()).rstrip(b"=").decode()
    return f"sha256={digest},{len(data)}"


def build_wheel(
    out_dir: Path,
    version: str,
    platform_tag: str,
    binary_name: str,
    binary: bytes,
    license_text: bytes,
) -> Path:
    dist_info = f"{DIST_NAME}-{version}.dist-info"
    data_dir = f"{DIST_NAME}-{version}.data"

    metadata = (
        "Metadata-Version: 2.1\n"
        f"Name: {DIST_NAME}\n"
        f"Version: {version}\n"
        f"Summary: {SUMMARY}\n"
        "Home-page: https://github.com/databricks/cli\n"
        "License: Databricks License\n"
        "License-File: LICENSE\n"
        "Description-Content-Type: text/markdown\n"
        f"\n{DESCRIPTION}"
    ).encode()
    wheel_meta = (
        "Wheel-Version: 1.0\n"
        "Generator: databricks-cli build_cli_wheels.py\n"
        "Root-Is-Purelib: false\n" + "".join(f"Tag: py3-none-{t}\n" for t in platform_tag.split("."))
    ).encode()

    # name -> (content, unix mode)
    files = {
        f"{data_dir}/scripts/{binary_name}": (binary, 0o755),
        f"{dist_info}/METADATA": (metadata, 0o644),
        f"{dist_info}/WHEEL": (wheel_meta, 0o644),
        f"{dist_info}/licenses/LICENSE": (license_text, 0o644),
    }
    record = "".join(f"{name},{record_hash(data)}\n" for name, (data, _) in files.items())
    record += f"{dist_info}/RECORD,,\n"
    files[f"{dist_info}/RECORD"] = (record.encode(), 0o644)

    out = out_dir / f"{DIST_NAME}-{version}-py3-none-{platform_tag}.whl"
    buf = io.BytesIO()
    with zipfile.ZipFile(buf, "w", zipfile.ZIP_DEFLATED) as z:
        for name, (data, mode) in files.items():
            info = zipfile.ZipInfo(name, ZIP_DATE_TIME)
            info.external_attr = (0o100000 | mode) << 16
            info.compress_type = zipfile.ZIP_DEFLATED
            z.writestr(info, data)
    out.write_bytes(buf.getvalue())
    return out


def main() -> None:
    if len(sys.argv) not in (3, 4):
        raise SystemExit(__doc__)
    artifact_dir = Path(sys.argv[1])
    out_dir = Path(sys.argv[2])
    license_file = Path(sys.argv[3]) if len(sys.argv) == 4 else Path(__file__).resolve().parents[2] / "LICENSE"

    cli_version, sums = read_checksums(artifact_dir)
    version = pep440(cli_version)
    license_text = license_file.read_bytes()
    out_dir.mkdir(parents=True, exist_ok=True)

    for (goos, goarch), platform_tag in PLATFORMS.items():
        archive = find_archive(artifact_dir, goos, goarch)
        actual = hashlib.sha256(archive.read_bytes()).hexdigest()
        if sums.get(archive.name) != actual:
            raise SystemExit(f"checksum mismatch for {archive.name}: expected {sums.get(archive.name)}, got {actual}")
        binary_name = "databricks.exe" if goos == "windows" else "databricks"
        binary = extract_binary(archive, binary_name)
        print(build_wheel(out_dir, version, platform_tag, binary_name, binary, license_text))


if __name__ == "__main__":
    main()
