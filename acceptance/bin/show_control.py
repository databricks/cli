#!/usr/bin/env python3
"""
Passthrough filter that prints control characters other than newline and tab
as <0xNN>, and a byte that is not valid UTF-8 as <0xNN> too, so that an output
file never holds a raw terminal escape sequence. The carriage return of a CRLF
line ending is kept: a Windows program passed through as is ends its lines so.
"""

import sys
import unicodedata

text = sys.stdin.buffer.read().decode("utf-8", errors="surrogateescape")
out = []
for i, char in enumerate(text):
    code = ord(char)
    if char == "\r" and text[i + 1 : i + 2] == "\n":
        out.append(char)
    elif 0xDC80 <= code <= 0xDCFF:
        # A byte that did not decode.
        out.append("<0x%02x>" % (code - 0xDC00))
    elif unicodedata.category(char) == "Cc" and char not in "\n\t":
        out.append("<0x%02x>" % code)
    else:
        out.append(char)

sys.stdout.buffer.write("".join(out).encode("utf-8"))
