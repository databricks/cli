#!/usr/bin/env python3
"""
Contract check for gen_fuzz_config.to_yaml: every scalar is on its own line as
`key: <json>`. mutate_fuzz_config.py's line-based loader relies on this. Prints each
case's YAML (diffed by the harness) and exits non-zero on a violation.
"""

import json
import os
import re
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from gen_fuzz_config import DANGEROUS_STRINGS, SKIP_PROPERTY_NAMES, gen_config, to_yaml

# Tricky shapes: strings with ':' and '"', nested maps, lists of dicts, empty containers.
CASES = [
    {"comment": "value: with a colon", "description": 'quote " and : colon'},
    {"resources": {"jobs": {"j": {"name": "n", "tags": {"team": "jobs"}}}}},
    {"tasks": [{"description": "d", "timeout_seconds": 3600}, {"comment": "c"}]},
    {"nums": [0, 1, 2], "flag": True, "ratio": 1.5, "empty_map": {}, "empty_list": []},
]

HEADER = re.compile(r"[\w.\-]+:$")  # non-empty container: `key:`
SCALAR = re.compile(r"[\w.\-]+: (.+)$")  # `key: <json>`


def check_line(line):
    rest = line.lstrip(" ")
    rest = rest.removeprefix("- ")
    if HEADER.fullmatch(rest):
        return  # container header; value is on following lines
    m = SCALAR.fullmatch(rest)
    if m:
        json.loads(m.group(1))  # value must be single-line JSON
        return
    json.loads(rest)  # bare list scalar: `- <json>`


def main():
    failed = False
    for case in CASES:
        text = to_yaml(case)
        sys.stdout.write(text)
        for line in text.splitlines():
            if not line.strip():
                continue
            try:
                check_line(line)
            except ValueError:
                sys.stderr.write(f"contract violation: not `key: <json>`: {line!r}\n")
                failed = True

    # Generate mode emits DANGEROUS_STRINGS into free-form fields; each must still
    # serialize to a single `key: <json>` line so the line-based loader can parse it.
    for i, val in enumerate(DANGEROUS_STRINGS):
        line = to_yaml({"description": val}).rstrip("\n")
        if "\n" in line or not SCALAR.fullmatch(line):
            sys.stderr.write(f"DANGEROUS_STRINGS[{i}] broke the one-line scalar contract: {line!r}\n")
            failed = True

    schema_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "../../bundle/schema/jsonschema.json")
    with open(schema_path) as f:
        schema = json.load(f)
    seed24 = gen_config(schema, seed=24, unique="check", allowed={"registered_models"})
    rm = seed24["resources"]["registered_models"]["fuzz_registered_models_24"]
    if SKIP_PROPERTY_NAMES & set(rm):
        sys.stderr.write(
            f"seed 24 registered_models emitted output-only fields: {sorted(SKIP_PROPERTY_NAMES & set(rm))}\n"
        )
        failed = True
    for field in ("name", "catalog_name", "schema_name"):
        if field not in rm:
            sys.stderr.write(f"seed 24 registered_models missing {field}\n")
            failed = True

    if failed:
        sys.exit(1)


if __name__ == "__main__":
    main()
