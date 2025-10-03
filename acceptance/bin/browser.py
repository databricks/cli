#!/usr/bin/env python3
"""
This script fetches a URL.
It follows redirects if applicable.

Usage: browser.py <url>
"""

import urllib.request
import sys

if len(sys.argv) < 2:
    sys.stderr.write("Usage: browser.py <url>\n")
    sys.exit(1)

url = sys.argv[1]
try:
    response = urllib.request.urlopen(url)
    if response.status != 200:
        sys.stderr.write(f"Failed to fetch URL: {url} (status {response.status})\n")
        sys.exit(1)
except Exception as e:
    sys.stderr.write(f"Failed to fetch URL: {url} ({e})\n")
    sys.exit(1)

sys.exit(0)
