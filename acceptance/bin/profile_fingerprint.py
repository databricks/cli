#!/usr/bin/env python3

import hashlib
import sys


def append_uvarint(data, value):
    while value >= 0x80:
        data.append((value & 0x7F) | 0x80)
        value >>= 7
    data.append(value)


values = {
    "auth_type": "databricks-cli",
    "host": sys.argv[1],
}
serialized = bytearray()
for key in sorted(values):
    value = values[key]
    append_uvarint(serialized, len(key.encode()))
    serialized.extend(key.encode())
    append_uvarint(serialized, len(value.encode()))
    serialized.extend(value.encode())

print(hashlib.sha256(serialized).hexdigest())
