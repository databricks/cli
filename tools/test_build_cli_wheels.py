#!/usr/bin/env python3
import io
import tarfile
import tempfile
import unittest
import zipfile
from pathlib import Path

import build_cli_wheels as b


class TestBuildCliWheels(unittest.TestCase):
    def test_pep440(self):
        self.assertEqual(b.pep440("1.23.0"), "1.23.0")
        self.assertEqual(b.pep440("1.23.1-dev+abc123"), "1.23.1.dev0+abc123")
        with self.assertRaises(SystemExit):
            b.pep440("v1.23.0")

    def test_end_to_end(self):
        with tempfile.TemporaryDirectory() as tmp:
            art, out = Path(tmp, "art"), Path(tmp, "out")
            art.mkdir()
            for (goos, goarch), _ in b.PLATFORMS.items():
                if goos == "windows":
                    path = art / f"databricks_cli_1.23.0_{goos}_{goarch}.zip"
                    with zipfile.ZipFile(path, "w") as z:
                        z.writestr("databricks.exe", b"MZ")
                else:
                    path = art / f"databricks_cli_1.23.0_{goos}_{goarch}.tar.gz"
                    with tarfile.open(path, "w:gz") as t:
                        info = tarfile.TarInfo("databricks")
                        info.size = 3
                        t.addfile(info, io.BytesIO(b"ELF"))
            (art / "databricks_cli_1.23.0_SHA256SUMS").touch()
            lic = Path(tmp, "LICENSE")
            lic.write_text("license")

            b.sys.argv = ["x", str(art), str(out), str(lic)]
            b.main()

            wheels = sorted(p.name for p in out.glob("*.whl"))
            self.assertEqual(len(wheels), len(b.PLATFORMS))
            self.assertIn("databricks-1.23.0-py3-none-win_amd64.whl", wheels)
            with zipfile.ZipFile(out / "databricks-1.23.0-py3-none-macosx_11_0_arm64.whl") as z:
                script = z.getinfo("databricks-1.23.0.data/scripts/databricks")
                self.assertEqual(script.external_attr >> 16, 0o100755)
                self.assertIn("Tag: py3-none-macosx_11_0_arm64", z.read("databricks-1.23.0.dist-info/WHEEL").decode())


if __name__ == "__main__":
    unittest.main()
