#!/usr/bin/env python3
"""
List files in zip archive
"""

import sys
import zipfile
from pathlib import Path


for zip_path in sys.argv[1:]:
    with zipfile.ZipFile(zip_path.strip()) as z:
        for info in z.infolist():
            print(info.filename)
