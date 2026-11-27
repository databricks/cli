#!/usr/bin/env python3
import tempfile
import unittest
import zipfile
from pathlib import Path

import build_cli_wheels as b
import validate_cli_wheels as v

VERSION = "1.23.0"
LINUX_WHEEL = f"databricks-{VERSION}-py3-none-{b.PLATFORMS[('linux', 'amd64')]}.whl"
DIST_INFO = f"databricks-{VERSION}.dist-info"
SCRIPT = f"databricks-{VERSION}.data/scripts/databricks"


def build_wheels(out: Path) -> None:
    out.mkdir()
    for (goos, _), tag in b.PLATFORMS.items():
        binary_name = "databricks.exe" if goos == "windows" else "databricks"
        b.build_wheel(out, VERSION, tag, binary_name, b"binary", b"license")


def rewrite(whl: Path, changes: dict, fix_record: bool = True, new_name: str = "") -> None:
    """Rewrite a wheel with files replaced (bytes) or removed (None), optionally keeping RECORD consistent."""
    with zipfile.ZipFile(whl) as z:
        files = {i.filename: (i, z.read(i)) for i in z.infolist()}
    for name, data in changes.items():
        if data is None:
            del files[name]
        else:
            info = files[name][0] if name in files else zipfile.ZipInfo(name, b.ZIP_DATE_TIME)
            files[name] = (info, data)
    if fix_record:
        record_name = f"{DIST_INFO}/RECORD"
        record = "".join(f"{n},{b.record_hash(d)}\n" for n, (_, d) in files.items() if n != record_name)
        files[record_name] = (files[record_name][0], (record + f"{record_name},,\n").encode())
    whl.unlink()
    with zipfile.ZipFile(whl.with_name(new_name or whl.name), "w", zipfile.ZIP_DEFLATED) as z:
        for info, data in files.values():
            z.writestr(info, data)


class TestValidateCliWheels(unittest.TestCase):
    def setUp(self):
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.out = Path(tmp.name, "wheels")
        build_wheels(self.out)
        self.linux = self.out / LINUX_WHEEL

    def assert_invalid(self, message: str):
        with self.assertRaises(SystemExit) as cm:
            v.validate_dir(self.out)
        self.assertIn(message, str(cm.exception))

    def test_valid(self):
        self.assertEqual(len(v.validate_dir(self.out)), 6)

    def test_metadata_version_too_old_for_license_file(self):
        with zipfile.ZipFile(self.linux) as z:
            metadata = z.read(f"{DIST_INFO}/METADATA").replace(b"Metadata-Version: 2.4", b"Metadata-Version: 2.1")
        rewrite(self.linux, {f"{DIST_INFO}/METADATA": metadata})
        self.assert_invalid("license-file introduced in metadata version 2.4")

    def test_metadata_version_mismatch(self):
        with zipfile.ZipFile(self.linux) as z:
            metadata = z.read(f"{DIST_INFO}/METADATA").replace(b"Version: 1.23.0", b"Version: 1.23.1")
        rewrite(self.linux, {f"{DIST_INFO}/METADATA": metadata})
        self.assert_invalid("do not match the filename")

    def test_record_hash_mismatch(self):
        rewrite(self.linux, {SCRIPT: b"tampered"}, fix_record=False)
        self.assert_invalid(f"RECORD: wrong hash or size for {SCRIPT}")

    def test_extra_file(self):
        rewrite(self.linux, {"evil.pth": b"import os"})
        self.assert_invalid("unexpected files: ['evil.pth']")

    def test_missing_license(self):
        rewrite(self.linux, {f"{DIST_INFO}/licenses/LICENSE": None})
        self.assert_invalid("unexpected files")

    def test_wheel_tags_differ_from_filename(self):
        rewrite(self.linux, {}, new_name=f"databricks-{VERSION}-py3-none-manylinux_2_28_x86_64.whl")
        self.assert_invalid("do not match the filename")

    def test_missing_platform(self):
        next(self.out.glob("*win_arm64.whl")).unlink()
        self.assert_invalid("missing wheels for [('windows', 'arm64')]")

    def test_error_paths_use_forward_slashes(self):
        rewrite(self.linux, {"evil.pth": b"import os"})
        self.assert_invalid(f"{self.linux.as_posix()}: ")


if __name__ == "__main__":
    unittest.main()
