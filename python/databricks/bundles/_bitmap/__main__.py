"""Decode a bundle bitmap read from stdin, like ``databricks bundle bitmap decode``.

Usage: ``python -m databricks.bundles._bitmap [-o text|json] < bitmap.b64``
"""

import argparse
import json
import sys
import zlib

from databricks.bundles._bitmap import decode, decode_json


def main() -> int:
    parser = argparse.ArgumentParser(
        prog="python -m databricks.bundles._bitmap",
        description="Decode a bundle bitmap read from stdin.",
    )
    parser.add_argument(
        "-o",
        "--output",
        choices=["text", "json"],
        default="text",
        help="text: one line per set field; json: trees of set fields",
    )
    args = parser.parse_args()
    try:
        encoded = sys.stdin.read()
        if args.output == "json":
            lines = [json.dumps(decode_json(encoded), indent=2)]
        else:
            lines = decode(encoded)
    except (ValueError, zlib.error) as e:
        print(f"Error: {e}", file=sys.stderr)
        return 1
    # Write bytes so Windows does not translate "\n" to "\r\n": the output must
    # match the CLI's byte for byte.
    sys.stdout.buffer.write("".join(line + "\n" for line in lines).encode())
    return 0


if __name__ == "__main__":
    sys.exit(main())
