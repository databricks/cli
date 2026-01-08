#!/usr/bin/env python3
"""
Sort ACLs in JSON files recursively to ensure consistent ordering.

This script reads JSON from stdin, recursively finds all "acls" arrays,
sorts them by principal, and outputs the normalized JSON pretty-printed.

Usage:
    cat file.json | sort_acls_json.py
    sort_acls_json.py < file.json
"""

import json
import sys


def sort_acls_recursive(obj):
    """Recursively traverse the object and sort any 'acls' arrays by principal."""
    if isinstance(obj, dict):
        result = {}
        for key, value in obj.items():
            if key == "acls" and isinstance(value, list):
                # Sort ACLs by principal
                result[key] = sorted(value, key=repr)
            else:
                result[key] = sort_acls_recursive(value)
        return result
    elif isinstance(obj, list):
        return [sort_acls_recursive(item) for item in obj]
    else:
        return obj


def main():
    raw = sys.stdin.read()
    try:
        data = json.loads(raw)
    except Exception:
        print("Not json:\n" + raw, flush=True)
        raise
    normalized = sort_acls_recursive(data)
    print(json.dumps(normalized, indent=2))


if __name__ == "__main__":
    main()
