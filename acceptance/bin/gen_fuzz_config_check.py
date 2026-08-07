#!/usr/bin/env python3
"""
Contract checks for gen_fuzz_config: the curated grant/permission/skip tables still agree with
the schema they annotate. Exits non-zero on a violation, reported on stderr.
"""

import json
import os
import random
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from gen_fuzz_config import (
    GRANT_PRIVILEGE,
    PERMISSION_LEVEL,
    SKIP_PROPERTY_NAMES,
    Generator,
    resource_element,
    resource_types,
)

# The union of every level of every resource type, which resources without an enum of their own
# point at. It says nothing about what one resource accepts, so those entries go unverified.
GENERIC_LEVEL_REF = "iam.PermissionLevel"


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
            continue

        element = resource_element(gen, types[rtype])

        levels = nested_enum(gen, element, "permissions", "level")
        level = PERMISSION_LEVEL.get(rtype)
        if level is not None:
            if not levels:
                errors.append(f"{rtype}: PERMISSION_LEVEL has {level!r} but schema has no levels")
            elif level not in levels:
                errors.append(f"{rtype}: PERMISSION_LEVEL {level!r} is not one of {levels}")

        privileges = nested_enum(gen, element, "grants", "privileges")
        privilege = GRANT_PRIVILEGE.get(rtype)
        if privilege is not None:
            if not privileges:
                errors.append(f"{rtype}: GRANT_PRIVILEGE has {privilege!r} but schema has no privileges")
            elif privilege not in privileges:
                errors.append(f"{rtype}: GRANT_PRIVILEGE {privilege!r} is not a catalog privilege")

    declared = set()
    property_names(schema, declared)
    for name in sorted(SKIP_PROPERTY_NAMES - declared):
        errors.append(f"SKIP_PROPERTY_NAMES has {name!r}, which no resource declares")

    return errors


def main():
    schema_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "../../bundle/schema/jsonschema.json")
    with open(schema_path) as f:
        schema = json.load(f)

    failed = False
    for error in check_tables(schema):
        sys.stderr.write(error + "\n")
        failed = True

    if failed:
        sys.exit(1)


if __name__ == "__main__":
    main()
