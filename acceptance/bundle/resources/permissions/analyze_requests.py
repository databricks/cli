#!/usr/bin/env python3
"""
Analyze all requests recorded in subtests to highlight differences between direct and terraform.
"""

import json
import re
import sys
import tomllib
from pathlib import Path
from difflib import unified_diff


def read_json_many(file):
    with open(file) as fobj:
        s = fobj.read()

    # fix invalid json due to replacement: x: [NUMID]  -->  x: "[NUMID]"
    s = re.sub(r": \[(.*?)\]", r': "[\1]"', s)

    result = []

    try:
        dec = json.JSONDecoder()
        pos = 0
        n = len(s)
        while True:
            # skip whitespace between objects
            while pos < n and s[pos].isspace():
                pos += 1
            if pos >= n:
                break
            obj, idx = dec.raw_decode(s, pos)
            result.append(obj)
            pos = idx

    except Exception as ex:
        sys.exit(f"Failed to parse {file}: {ex}")

    return result


def normalize_acls(data):
    """Recursively normalize ACLs in the data structure by sorting them."""
    if isinstance(data, dict):
        result = {}
        for key, value in data.items():
            if key == "access_control_list" and isinstance(value, list):
                # Sort ACLs by all fields to normalize order
                result[key] = sorted(value, key=lambda x: json.dumps(x, sort_keys=True))
            else:
                result[key] = normalize_acls(value)
        return result
    elif isinstance(data, list):
        return [normalize_acls(item) for item in data]
    else:
        return data


def compare_files(file1, file2):
    """Compare two JSON files and return comparison result."""
    data1 = read_json_many(file1)
    data2 = read_json_many(file2)

    if data1 == data2:
        return "EXACT", ""

    normalized1 = normalize_acls(data1)
    normalized2 = normalize_acls(data2)

    if normalized1 == normalized2:
        return "MATCH", ""

    json1_str = json.dumps(normalized1, indent=2, sort_keys=True)
    json2_str = json.dumps(normalized2, indent=2, sort_keys=True)

    diff_lines = list(
        unified_diff(
            json1_str.splitlines(keepends=True),
            json2_str.splitlines(keepends=True),
            fromfile=str(file1),
            tofile=str(file2),
            n=3,
        )
    )

    return "DIFF ", "\n" + to_slash("".join(diff_lines).rstrip())


def to_slash(x):
    return str(x).replace("\\", "/")


def load_supported_engines(path):
    current = path
    while True:
        for name in ("out.test.toml", "test.toml"):
            config_file = current / name
            if config_file.exists():
                with config_file.open("rb") as fobj:
                    config = tomllib.load(fobj)
                return set(config.get("EnvMatrix", {}).get("DATABRICKS_BUNDLE_ENGINE", []))
        if current == Path("."):
            return set()
        current = current.parent


def main():
    current_dir = Path(".")

    direct_files = list(current_dir.glob("**/*.direct.json"))

    for direct_file in sorted(direct_files):
        if direct_file.name.startswith("out.plan"):
            # expected difference
            continue

        terraform_file = direct_file.parent / direct_file.name.replace(".direct.", ".terraform.")

        fname = to_slash(direct_file)
        supported_engines = load_supported_engines(direct_file.parent)

        if terraform_file.exists():
            result, diff = compare_files(direct_file, terraform_file)
            print(result + " " + fname + diff)
        elif "terraform" not in supported_engines:
            print(f"DIRECT_ONLY {fname}")
        else:
            print(f"ERROR {fname}: Missing terraform file {to_slash(terraform_file)}")


if __name__ == "__main__":
    main()
