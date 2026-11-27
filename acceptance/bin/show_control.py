#!/usr/bin/env python3
"""
Passthrough filter that prints control bytes other than newline and tab as
<0xNN>, so that an output file never holds a raw terminal escape sequence.
"""

import sys

out = bytearray()
for byte in sys.stdin.buffer.read():
    if (byte < 0x20 and byte not in (0x09, 0x0A)) or byte == 0x7F:
        out += b"<0x%02x>" % byte
    else:
        out.append(byte)

sys.stdout.buffer.write(out)
