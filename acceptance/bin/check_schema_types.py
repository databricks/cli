#!/usr/bin/env python3
"""
Assert every `type` in the bundle schema is one gen_fuzz_config.py can generate, so a new
libs/jsonschema.Type fails loudly here instead of being silently skipped by the fuzz loop.
"""

import argparse
import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from gen_fuzz_config import HANDLED_TYPES


def collect_types(node, found):
    if isinstance(node, dict):
        t = node.get("type")
        if isinstance(t, str):
            found.add(t)
        elif isinstance(t, list):
            found.update(x for x in t if isinstance(x, str))
        for v in node.values():
            collect_types(v, found)
    elif isinstance(node, list):
        for v in node:
            collect_types(v, found)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--schema", required=True)
    args = parser.parse_args()

    with open(args.schema) as f:
        schema = json.load(f)

    found = set()
    collect_types(schema, found)

    unhandled = sorted(found - HANDLED_TYPES)
    if unhandled:
        sys.exit(f"check_schema_types: gen_fuzz_config.py cannot generate schema types {unhandled}")


if __name__ == "__main__":
    main()
