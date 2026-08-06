#!/usr/bin/env python3
"""
Contract checks for gen_fuzz_config:

- to_yaml puts every scalar on its own line as `key: <json>`, which mutate_fuzz_config's
  line-based loader relies on. Each case's YAML is printed, and the harness diffs it.
- The curated tables still agree with the schema they annotate (check_tables).

Exits non-zero on a violation, reported on stderr.
"""

import json
import os
import random
import re
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from gen_fuzz_config import (
    DANGEROUS_STRINGS,
    GRANT_PRIVILEGE,
    PERMISSION_LEVEL,
    SKIP_PROPERTY_NAMES,
    Generator,
    gen_config,
    resource_element,
    resource_types,
    to_yaml,
)

# Tricky shapes: strings with ':' and '"', nested maps, lists of dicts, lists in lists,
# empty containers.
CASES = [
    {"comment": "value: with a colon", "description": 'quote " and : colon'},
    {"resources": {"jobs": {"j": {"name": "n", "tags": {"team": "jobs"}}}}},
    {"tasks": [{"description": "d", "timeout_seconds": 3600}, {"comment": "c"}]},
    {"nums": [0, 1, 2], "flag": True, "ratio": 1.5, "empty_map": {}, "empty_list": []},
    {"matrix": [[1, 2], [], {}]},
]

HEADER = re.compile(r"[\w.\-]+:$")  # non-empty container: `key:`
SCALAR = re.compile(r"[\w.\-]+: (.+)$")  # `key: <json>`

# Resources without a permission enum of their own point at this union of every level of every
# resource type, so it does not say which ones this resource accepts. Those entries in
# PERMISSION_LEVEL are unverifiable here and are checked only for naming a real resource type.
GENERIC_LEVEL_REF = "iam.PermissionLevel"


def check_line(line):
    rest = line.lstrip(" ")
    if rest == "-":
        return  # nested container marker; value is on following lines
    rest = rest.removeprefix("- ")
    if HEADER.fullmatch(rest):
        return  # container header; value is on following lines
    m = SCALAR.fullmatch(rest)
    if m:
        json.loads(m.group(1))  # value must be single-line JSON
        return
    json.loads(rest)  # bare list scalar: `- <json>`


def branches(gen, node):
    node = gen.resolve(node)
    return [gen.resolve(b) for b in node.get("oneOf", node.get("anyOf", [node]))]


def nested_enum(gen, element, field, item_field):
    """The enum behind <resource>.<field>[].<item_field>, or None if absent or generic."""
    for el in branches(gen, element):
        prop = el.get("properties", {}).get(field)
        if prop is None:
            continue
        for array in branches(gen, prop):
            if array.get("type") != "array":
                continue
            for item in branches(gen, array["items"]):
                inner = item.get("properties", {}).get(item_field)
                if inner is None:
                    continue
                if GENERIC_LEVEL_REF in inner.get("$ref", ""):
                    return None
                for branch in branches(gen, inner):
                    if branch.get("enum"):
                        return branch["enum"]
                    # grants[].privileges holds a list of enum values rather than one.
                    if branch.get("type") == "array":
                        for value in branches(gen, branch["items"]):
                            if value.get("enum"):
                                return value["enum"]
    return None


def property_names(node, out):
    if isinstance(node, dict):
        for key, value in node.items():
            if key == "properties" and isinstance(value, dict):
                out.update(value)
            property_names(value, out)
    elif isinstance(node, list):
        for value in node:
            property_names(value, out)


def check_tables(schema):
    """The curated tables are pinned to a schema that moves under them; report what no longer fits."""
    gen = Generator(schema, random.Random(0), "check")
    types = resource_types(gen)
    errors = []

    for rtype in sorted(set(PERMISSION_LEVEL) | set(GRANT_PRIVILEGE)):
        if rtype not in types:
            errors.append(f"{rtype}: not a resource type in the schema")

    for rtype in sorted(types):
        element = resource_element(gen, types[rtype])

        levels = nested_enum(gen, element, "permissions", "level")
        level = PERMISSION_LEVEL.get(rtype)
        if levels and level is None:
            # gen_permissions emits nothing without an entry, so the resource silently loses its
            # permissions coverage even though the schema says exactly what it accepts.
            errors.append(f"{rtype}: schema names its levels ({', '.join(levels)}) but PERMISSION_LEVEL has none")
        elif levels and level not in levels:
            errors.append(f"{rtype}: PERMISSION_LEVEL {level!r} is not one of {levels}")

        privileges = nested_enum(gen, element, "grants", "privileges")
        privilege = GRANT_PRIVILEGE.get(rtype)
        if privileges and privilege is not None and privilege not in privileges:
            errors.append(f"{rtype}: GRANT_PRIVILEGE {privilege!r} is not a catalog privilege")

    declared = set()
    property_names(schema, declared)
    for name in sorted(SKIP_PROPERTY_NAMES - declared):
        errors.append(f"SKIP_PROPERTY_NAMES has {name!r}, which no resource declares")

    return errors


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

    # Each DANGEROUS_STRINGS probe must still serialize to a single `key: <json>` line.
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

    for error in check_tables(schema):
        sys.stderr.write(error + "\n")
        failed = True

    if failed:
        sys.exit(1)


if __name__ == "__main__":
    main()
