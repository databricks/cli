#!/usr/bin/env python3
"""
This script fetches a URL.
It follows redirects if applicable.

Usage: browser.py <url>
"""

import os
import sys
import urllib.parse
import urllib.request

if len(sys.argv) < 2:
    sys.stderr.write("Usage: browser.py <url>\n")
    sys.exit(1)

url = sys.argv[1]
expected_group_id = os.environ.get("DATABRICKS_TEST_GROUP_ID")
if expected_group_id is not None:
    group_ids = urllib.parse.parse_qs(urllib.parse.urlparse(url).query).get(
        "assume_group", []
    )
    expected_group_ids = [] if expected_group_id == "" else [expected_group_id]
    if group_ids != expected_group_ids:
        sys.stderr.write(
            f"Expected assume_group values {expected_group_ids!r}, got {group_ids!r}\n"
        )
        sys.exit(1)

try:
    response = urllib.request.urlopen(url)
    if response.status != 200:
        sys.stderr.write(f"Failed to fetch URL: {url} (status {response.status})\n")
        sys.exit(1)
except Exception as e:
    sys.stderr.write(f"Failed to fetch URL: {url} ({e})\n")
    sys.exit(1)

sys.exit(0)
