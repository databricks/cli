#!/usr/bin/env python3
"""
Simulates the login.databricks.com discovery flow for acceptance tests.

When the CLI opens this "browser" with the login.databricks.com URL,
the script extracts the OAuth parameters from the destination_url,
constructs a callback to localhost with an iss parameter pointing
at the testserver, and fetches it.

Usage: discovery_browser.py <url>
"""

import os
import sys
import urllib.parse
import urllib.request

if len(sys.argv) < 2:
    sys.stderr.write("Usage: discovery_browser.py <url>\n")
    sys.exit(1)

url = sys.argv[1]
parsed = urllib.parse.urlparse(url)
top_params = urllib.parse.parse_qs(parsed.query)

destination_url = top_params.get("destination_url", [None])[0]
if not destination_url:
    sys.stderr.write(f"No destination_url found in: {url}\n")
    sys.exit(1)

dest_parsed = urllib.parse.urlparse(destination_url)
dest_params = urllib.parse.parse_qs(dest_parsed.query)

expected_group_id = os.environ.get("DATABRICKS_TEST_GROUP_ID")
if expected_group_id is not None:
    top_level_group_ids = top_params.get("assume_group", [])
    if top_level_group_ids:
        sys.stderr.write(
            f"Expected no top-level assume_group, got {top_level_group_ids!r}\n"
        )
        sys.exit(1)
    group_ids = dest_params.get("assume_group", [])
    expected_group_ids = [] if expected_group_id == "" else [expected_group_id]
    if group_ids != expected_group_ids:
        sys.stderr.write(
            f"Expected nested assume_group values {expected_group_ids!r}, got {group_ids!r}\n"
        )
        sys.exit(1)

redirect_uri = dest_params.get("redirect_uri", [None])[0]
state = dest_params.get("state", [None])[0]

if not redirect_uri or not state:
    sys.stderr.write(f"Missing redirect_uri or state in destination_url: {destination_url}\n")
    sys.exit(1)

# The testserver's host acts as the workspace issuer.
testserver_host = os.environ.get("DATABRICKS_HOST", "")
if not testserver_host:
    sys.stderr.write("DATABRICKS_HOST not set\n")
    sys.exit(1)

issuer = testserver_host.rstrip("/") + "/oidc"

# Build the callback URL with code, state, and iss (the workspace issuer).
callback_params = urllib.parse.urlencode(
    {
        "code": "oauth-code",
        "state": state,
        "iss": issuer,
    }
)
callback_url = f"{redirect_uri}?{callback_params}"

try:
    response = urllib.request.urlopen(callback_url)
    if response.status != 200:
        sys.stderr.write(f"Callback failed: {callback_url} (status {response.status})\n")
        sys.exit(1)
except Exception as e:
    sys.stderr.write(f"Callback failed: {callback_url} ({e})\n")
    sys.exit(1)

sys.exit(0)
