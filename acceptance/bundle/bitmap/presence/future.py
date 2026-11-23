#!/usr/bin/env python3
"""Read an encoded bundle bitmap from stdin and print it as a newer CLI, whose schema has two more
fields, would encode it with the last of those set.

Wire format: "bm1.<size>." then base64 of raw DEFLATE of "DBTB" | size (uint32, big-endian) | context (uint16,
big-endian) | the bits, MSB first."""

import base64
import struct
import sys
import zlib

_, _, body = sys.stdin.read().strip().split(".", 2)  # "bm1.<size>.<base64>"
raw = zlib.decompress(base64.b64decode(body), -15)
size, context = struct.unpack(">IH", raw[4:10])
bits = [(raw[10 + i // 8] >> (7 - i % 8)) & 1 for i in range(size)] + [0, 1]
bitmap = bytearray((len(bits) + 7) // 8)
for i, bit in enumerate(bits):
    bitmap[i // 8] |= bit << (7 - i % 8)
payload = raw[:4] + struct.pack(">IH", len(bits), context) + bytes(bitmap)
compressor = zlib.compressobj(wbits=-15)
print(f"bm1.{len(bits)}." + base64.b64encode(compressor.compress(payload) + compressor.flush()).decode())
