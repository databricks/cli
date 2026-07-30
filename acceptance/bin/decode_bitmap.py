#!/usr/bin/env python3
"""Decode a base64 bundle bitmap and print '0/1 field_path' per schema line.

Usage: decode_bitmap.py SCHEMA_FILE < ENCODED_BITMAP

The output matches `databricks bundle bitmap bitmap-text`, so a test can assert
the encoded bitmap round-trips to the same per-field view.

Wire format (see bundle/bitmap/bitmap.go): raw-DEFLATE of
  magic "DBTB" (4) | size uint32 BE | context uint16 BE | bitmap ceil(size/8) bytes
with bit i (MSB-first within each byte) corresponding to schema line i.
"""

import base64
import struct
import sys
import zlib

MAGIC = b"DBTB"
HEADER_SIZE = 10


def main():
    schema_file = sys.argv[1]
    with open(schema_file) as f:
        schema = f.read().splitlines()

    encoded = sys.stdin.read().strip()
    # -15 selects raw DEFLATE (no zlib/gzip wrapper), matching compress/flate.
    raw = zlib.decompress(base64.b64decode(encoded), -15)

    if raw[:4] != MAGIC:
        sys.exit(f"bad magic: {raw[:4]!r}")
    (size,) = struct.unpack(">I", raw[4:8])
    if len(raw) != HEADER_SIZE + (size + 7) // 8:
        sys.exit(f"payload size mismatch: {size} bits, {len(raw)} bytes")
    if size != len(schema):
        sys.exit(f"schema has {len(schema)} lines but bitmap has {size} bits")

    bitmap = raw[HEADER_SIZE:]
    for i, field in enumerate(schema):
        bit = (bitmap[i // 8] >> (7 - i % 8)) & 1
        print(f"{bit} {field}")


if __name__ == "__main__":
    main()
