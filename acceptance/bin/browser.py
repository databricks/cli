#!/usr/bin/env python3
"""
This script fetches a URL.
It follows redirects if applicable.

Usage: get_url.py <url>
"""

import urllib.request
import sys

url = sys.argv[1]
response = urllib.request.urlopen(url)
if response.status != 200:
    sys.stderr.write(f"Failed to fetch URL: {url} (status {response.status})\n")
    sys.exit(1)

sys.exit(0)
