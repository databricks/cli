#!/usr/bin/env python3
"""
Contract check for gen_fuzz_config.to_yaml: every scalar is on its own line as
`key: <json>`. edit_fuzz_config.py relies on this to edit a field by regex, not a YAML
parser. Prints each case's YAML (diffed by the harness) and exits non-zero on a violation.
"""

import json
import os
import re
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from edit_fuzz_config import FIELD_RE
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

    # edit_fuzz_config's FIELD_RE must still match when the value contains a colon (CASES[0]).
    if not any(FIELD_RE.match(line) for line in to_yaml(CASES[0]).splitlines()):
        sys.stderr.write("FIELD_RE did not match a comment/description line\n")
        failed = True

    # Generate mode now emits DANGEROUS_STRINGS into free-form fields; each must still
    # serialize to a single `key: <json>` line so edit_fuzz_config can rewrite it in place.
    for i, val in enumerate(DANGEROUS_STRINGS):
        line = to_yaml({"description": val}).rstrip("\n")
        if "\n" in line or not FIELD_RE.match(line):
            sys.stderr.write(f"DANGEROUS_STRINGS[{i}] broke the one-line comment/description contract: {line!r}\n")
            failed = True

    # Multi-resource configs merge types under resources.<type>, and one resource
    # references another's name so the interpolation/ordering path is exercised.
    def resource_type(field):
        return {
            "oneOf": [
                {
                    "type": "object",
                    "additionalProperties": {
                        "type": "object",
                        "properties": {"name": {"type": "string"}, field: {"type": "string"}},
                        "required": ["name"],
                    },
                }
            ]
        }

    multi = gen_config(
        {
            "$defs": {},
            "properties": {
                "resources": {
                    "oneOf": [
                        {
                            "type": "object",
                            "properties": {
                                "jobs": resource_type("description"),
                                "volumes": resource_type("comment"),
                            },
                        }
                    ]
                }
            },
        },
        seed=42,
        unique="check",
        allowed=set(),
        resource_count=2,
    )
    if len(multi["resources"]) < 1 or sum(len(v) for v in multi["resources"].values()) != 2:
        sys.stderr.write("gen_config did not emit two resources\n")
        failed = True

    values = [v for insts in multi["resources"].values() for inst in insts.values() for v in inst.values()]
    if not any(isinstance(v, str) and v.startswith("${resources.") for v in values):
        sys.stderr.write("gen_config did not inject a cross-resource reference\n")
        failed = True

    schema_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "../../bundle/schema/jsonschema.json")
    with open(schema_path) as f:
        schema = json.load(f)
    seed24 = gen_config(schema, seed=24, unique="check", allowed={"registered_models"}, resource_count=1)
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
