#!/usr/bin/env python3
"""Validate the Databricks CLI wheels built by build_cli_wheels.py.

Checks what a build backend would otherwise guarantee and what pip, uv and
PyPI rely on: the filename, METADATA (validated by `packaging`, which is
stricter than `twine check`), WHEEL tags, RECORD hashes and sizes, and that
each wheel installs only the CLI binary. Also requires exactly one wheel per
supported platform, all with the same version.

Requires `packaging` (see requirements.txt). Usage: validate_cli_wheels.py <wheels-dir>
"""

import base64
import csv
import email.parser
import hashlib
import io
import sys
import zipfile
from pathlib import Path

from packaging.metadata import Metadata
from packaging.utils import parse_wheel_filename

DIST_NAME = "databricks"
EXPECTED_PLATFORMS = {(goos, goarch) for goos in ("linux", "darwin", "windows") for goarch in ("amd64", "arm64")}
OS_PREFIXES = (("manylinux", "linux"), ("musllinux", "linux"), ("macosx", "darwin"), ("win", "windows"))
ARCH_SUFFIXES = (("x86_64", "amd64"), ("amd64", "amd64"), ("aarch64", "arm64"), ("arm64", "arm64"))


class InvalidWheel(Exception):
    pass


def platform_of(tag: str) -> tuple[str, str]:
    goos = next((o for prefix, o in OS_PREFIXES if tag.startswith(prefix)), None)
    goarch = next((a for suffix, a in ARCH_SUFFIXES if tag.endswith(suffix)), None)
    if goos is None or goarch is None:
        raise InvalidWheel(f"unsupported platform tag {tag}")
    return goos, goarch


def record_hash(data: bytes) -> str:
    return "sha256=" + base64.urlsafe_b64encode(hashlib.sha256(data).digest()).rstrip(b"=").decode()


def check_record(z: zipfile.ZipFile, record_name: str) -> None:
    rows = list(csv.reader(io.StringIO(z.read(record_name).decode())))
    listed = {}
    for row in rows:
        if len(row) != 3:
            raise InvalidWheel(f"RECORD: malformed row {row}")
        if row[0] in listed:
            raise InvalidWheel(f"RECORD: {row[0]} listed twice")
        listed[row[0]] = row
    names = z.namelist()
    if len(names) != len(set(names)):
        raise InvalidWheel("duplicate file in the zip")
    if set(listed) != set(names):
        raise InvalidWheel(f"RECORD does not match the zip: {sorted(set(listed) ^ set(names))}")
    for name, (_, digest, size) in listed.items():
        if name == record_name:
            if digest or size:
                raise InvalidWheel("RECORD must list itself without a hash or size")
            continue
        data = z.read(name)
        if digest != record_hash(data) or size != str(len(data)):
            raise InvalidWheel(f"RECORD: wrong hash or size for {name}")


def validate_wheel(path: Path) -> tuple[str, tuple[str, str]]:
    """Validate one wheel; return its version and (goos, goarch)."""
    name, version, build, tags = parse_wheel_filename(path.name)
    if name != DIST_NAME or build:
        raise InvalidWheel(f"unexpected name or build tag in {path.name}")
    if {(t.interpreter, t.abi) for t in tags} != {("py3", "none")}:
        raise InvalidWheel(f"expected py3-none tags, got {sorted(map(str, tags))}")
    platforms = {platform_of(t.platform) for t in tags}
    if len(platforms) != 1:
        raise InvalidWheel(f"tags cover more than one platform: {sorted(platforms)}")
    platform = platforms.pop()

    dist_info = f"{DIST_NAME}-{version}.dist-info"
    script = f"{DIST_NAME}-{version}.data/scripts/" + ("databricks.exe" if platform[0] == "windows" else "databricks")
    expected_files = {
        script,
        f"{dist_info}/METADATA",
        f"{dist_info}/WHEEL",
        f"{dist_info}/RECORD",
        f"{dist_info}/licenses/LICENSE",
    }

    with zipfile.ZipFile(path) as z:
        if set(z.namelist()) != expected_files:
            raise InvalidWheel(f"unexpected files: {sorted(set(z.namelist()) ^ expected_files)}")
        if z.getinfo(script).external_attr >> 16 != 0o100755:
            raise InvalidWheel(f"{script} is not a regular file with mode 755")
        check_record(z, f"{dist_info}/RECORD")

        try:
            metadata = Metadata.from_email(z.read(f"{dist_info}/METADATA"), validate=True)
        except Exception as e:
            details = "; ".join(str(x) for x in getattr(e, "exceptions", [e]))
            raise InvalidWheel(f"METADATA: {details}") from None
        if metadata.name != DIST_NAME or metadata.version != version:
            raise InvalidWheel(f"METADATA name/version {metadata.name} {metadata.version} do not match the filename")
        if metadata.license_files != ["LICENSE"]:
            raise InvalidWheel(f"METADATA: expected License-File: LICENSE, got {metadata.license_files}")

        wheel = email.parser.BytesParser().parsebytes(z.read(f"{dist_info}/WHEEL"))
        if wheel["Wheel-Version"] != "1.0" or wheel["Root-Is-Purelib"] != "false":
            raise InvalidWheel("WHEEL: expected Wheel-Version 1.0 and Root-Is-Purelib false")
        if set(wheel.get_all("Tag", [])) != {str(t) for t in tags}:
            raise InvalidWheel(f"WHEEL tags {wheel.get_all('Tag')} do not match the filename")

    return str(version), platform


def validate_dir(wheels_dir: Path) -> list[Path]:
    wheels = sorted(wheels_dir.glob("*.whl"))
    seen = {}
    versions = set()
    for whl in wheels:
        try:
            version, platform = validate_wheel(whl)
        except (InvalidWheel, ValueError) as e:
            raise SystemExit(f"{whl.as_posix()}: {e}") from None
        if platform in seen:
            raise SystemExit(f"{whl.as_posix()}: second wheel for {platform}, after {seen[platform].as_posix()}")
        seen[platform] = whl
        versions.add(version)
    if set(seen) != EXPECTED_PLATFORMS:
        raise SystemExit(f"{wheels_dir.as_posix()}: missing wheels for {sorted(EXPECTED_PLATFORMS - set(seen))}")
    if len(versions) != 1:
        raise SystemExit(f"{wheels_dir.as_posix()}: wheels have different versions {sorted(versions)}")
    return wheels


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit(__doc__)
    for whl in validate_dir(Path(sys.argv[1])):
        print(f"valid: {whl.as_posix()}")


if __name__ == "__main__":
    main()
