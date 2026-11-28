#!/usr/bin/env python3
"""
Write small wheels that exercise parts of the wheel format the prebuilt test wheel lacks:
- mypkg: an executable file inside the package and a <dist>-<version>.data/scripts/ entry;
- lib2: a distribution name that contains its own version ("lib2", version "2");
- readme: a description body (after the blank line ending the headers) with a "Version:" line;
- samever: a description body with a "Version:" line equal to the real version.
"""

import base64
import hashlib
import zipfile


def record_hash(data):
    return "sha256=" + base64.urlsafe_b64encode(hashlib.sha256(data).digest()).rstrip(b"=").decode()


def write_wheel(dist, version, files, description=""):
    dist_info = f"{dist}-{version}.dist-info/"
    files = dict(files)
    files[dist_info + "METADATA"] = (
        f"Metadata-Version: 2.1\nName: {dist}\nVersion: {version}\n{description}".encode(),
        0o644,
    )
    files[dist_info + "WHEEL"] = (
        b"Wheel-Version: 1.0\nGenerator: test\nRoot-Is-Purelib: true\nTag: py3-none-any\n",
        0o644,
    )
    record = "".join(f"{name},{record_hash(data)},{len(data)}\n" for name, (data, _) in files.items())
    record += dist_info + "RECORD,,\n"
    files[dist_info + "RECORD"] = (record.encode(), 0o644)

    with zipfile.ZipFile(f"{dist}-{version}-py3-none-any.whl", "w") as z:
        for name, (data, mode) in files.items():
            info = zipfile.ZipInfo(name, (2025, 3, 5, 15, 7, 32))
            info.create_system = 3  # unix, so that external_attr carries the file mode
            info.external_attr = (0o100000 | mode) << 16
            info.compress_type = zipfile.ZIP_DEFLATED
            z.writestr(info, data)


write_wheel(
    "mypkg",
    "1.0",
    {
        "mypkg/__init__.py": (b"", 0o644),
        "mypkg/bin/tool": (b"#!/bin/sh\necho tool\n", 0o755),
        "mypkg-1.0.data/scripts/myscript": (b"#!/bin/sh\necho myscript\n", 0o755),
    },
)

write_wheel("lib2", "2", {"lib2/__init__.py": (b"", 0o644)})

write_wheel(
    "readme", "1.0", {"readme/__init__.py": (b"", 0o644)}, description="\nChangelog\n\nVersion: 0.9\n- fixed bugs\n"
)

write_wheel(
    "samever",
    "1.0",
    {"samever/__init__.py": (b"", 0o644)},
    description="\nChangelog\n\nVersion: 1.0\n- first release\n",
)
