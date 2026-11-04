"""Decode a bundle bitmap read from stdin, like ``databricks bundle bitmap decode``.

Usage: ``python -m databricks.bundles._bitmap < bitmap.b64``
"""

import sys
import zlib

from databricks.bundles._bitmap import decode


def main() -> int:
    try:
        lines = decode(sys.stdin.read())
    except (ValueError, zlib.error) as e:
        print(f"Error: {e}", file=sys.stderr)
        return 1
    # Write bytes so Windows does not translate "\n" to "\r\n": the output must
    # match the CLI's byte for byte.
    sys.stdout.buffer.write("".join(line + "\n" for line in lines).encode())
    return 0


if __name__ == "__main__":
    sys.exit(main())
