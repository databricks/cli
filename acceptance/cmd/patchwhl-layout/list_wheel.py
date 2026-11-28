#!/usr/bin/env python3
"""
List entries of a wheel with their unix file mode.
"""

import sys
import zipfile

for path in sys.argv[1:]:
    with zipfile.ZipFile(path) as z:
        for info in z.infolist():
            print(f"{info.external_attr >> 16:o} {info.filename}")
