#!/usr/bin/env python3
"""
Fake browser that prints the URL it was asked to open and exits.

Used by acceptance tests that exercise commands which call libs/browser.Open
but don't need to follow the URL (unlike auth tests, which use browser.py to
close the OAuth callback loop). Setting BROWSER=echo_browser.py is portable
across darwin/linux/windows because libs/browser routes through libs/exec.

Usage: echo_browser.py <url>
"""

import sys

if len(sys.argv) < 2:
    sys.stderr.write("Usage: echo_browser.py <url>\n")
    sys.exit(1)

print(sys.argv[1])
