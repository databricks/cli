#!/usr/bin/env python3
"""
Cross-platform set mtime with nanosecond precision.
Usage: setmtime.py <timestamp> <filenames>
"""

import sys
import os
import datetime

timestamp = sys.argv[1]
ts, ns = timestamp.split(".")
dt = datetime.datetime.strptime(ts, "%Y-%m-%d %H:%M:%S").replace(tzinfo=datetime.timezone.utc)
ns = int(ns.ljust(9, "0"))
ts = int(dt.timestamp()) * 10**9 + ns
for filename in sys.argv[2:]:
    os.utime(filename, ns=(ts, ts))
