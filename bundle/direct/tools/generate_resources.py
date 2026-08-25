#!/usr/bin/env python3
# /// script
# dependencies = [
#   "pyyaml",
# ]
# ///
"""
Generate resources.generated.yml from cli.json field behaviors.
"""

import argparse
import json
import re
import sys
from pathlib import Path

import yaml


def parse_apitypes(generated_path, override_path):
    """Parse apitypes.generated.yml and override with apitypes.yml."""
    result = yaml.safe_load(generated_path.read_text()) or {}

    # Override with non-generated apitypes.yml (null values remove entries)
    override_data = yaml.safe_load(override_path.read_text()) or {}
    for resource, type_name in override_data.items():
        if type_name:
            result[resource] = type_name
        else:
            result.pop(resource, None)

    return result


def parse_out_fields(path):
    """Parse out.fields.txt to extract STATE field names per resource and array element types."""
    state_fields = {}
    array_element_types = {}  # (resource, base_path) -> element_type

    for line in path.read_text().splitlines():
        parts = line.split("\t")
        if len(parts) < 3 or not parts[0].startswith("resources."):
            continue

        field_path, field_type, flags = parts[0], parts[1], parts[2:]
        if "STATE" not in flags and "ALL" not in flags:
            continue

        # Field line: resources.<name>.*.<field>
        match = re.match(r"resources\.([a-z_]+)\.\*\.(.+)", field_path)
        if not match:
            continue
        resource, rest = match.group(1), match.group(2)

        # Array element type declarations (path ends with [*]): record the element type
        # so the generator can traverse into array element schemas that lack a ref in the
        # OpenAPI schema (e.g. App.resources[] -> AppResource).
        if rest.endswith("[*]"):
            base_path = rest[:-3]
            elem_type = field_type.lstrip("*")
            array_element_types[(resource, base_path)] = elem_type

        state_fields.setdefault(resource, set()).add(rest)

    return state_fields, array_element_types


def get_field_behaviors(schemas, type_name, resource_name=None, array_element_types=None):
    """Extract field behaviors from a schema, propagating INPUT_ONLY/OUTPUT_ONLY from containers."""
    if type_name not in schemas:
        return {}
    if array_element_types is None:
        array_element_types = {}

    def extract(schema, prefix, visited, depth, inherited):
        # Bound recursion as a runaway guard only; `visited` already guarantees
        # termination (each type is expanded at most once). Real fields reach depth 5
        # (e.g. external_model.custom_provider_config.bearer_token_auth.token_plaintext),
        # so keep ample headroom above that.
        if depth > 10:
            return {}
        results = {}
        for name, prop in schema.get("fields", {}).items():
            path = f"{prefix}.{name}" if prefix else name
            behaviors = list(prop.get("behaviors", []))
            for b in inherited:
                if b not in behaviors:
                    behaviors.append(b)
            if behaviors:
                results[path] = behaviors
            if "ref" in prop:
                ref = prop["ref"]
                if ref in schemas and ref not in visited:
                    visited.add(ref)
                    propagate = [b for b in behaviors if b in ("INPUT_ONLY", "OUTPUT_ONLY")]
                    results.update(extract(schemas[ref], path, visited, depth + 1, propagate))
            elif resource_name is not None:
                # For array fields with no ref in the schema, use the element type from
                # out.fields.txt (e.g. App.resources[] -> AppResource).
                elem_type = array_element_types.get((resource_name, path))
                if elem_type and elem_type in schemas and elem_type not in visited:
                    visited.add(elem_type)
                    propagate = [b for b in behaviors if b in ("INPUT_ONLY", "OUTPUT_ONLY")]
                    results.update(extract(schemas[elem_type], f"{path}[*]", visited, depth + 1, propagate))
        return results

    # Find INPUT_ONLY/OUTPUT_ONLY from container types that reference this type
    inherited = find_inherited_behaviors(schemas, type_name)
    return extract(schemas[type_name], "", set(), 0, inherited)


def find_inherited_behaviors(schemas, type_name):
    """Find INPUT_ONLY/OUTPUT_ONLY behaviors from containers that reference type_name."""
    inherited = []
    for container_schema in schemas.values():
        for field_prop in container_schema.get("fields", {}).values():
            if field_prop.get("ref", "") != type_name:
                continue
            behaviors = field_prop.get("behaviors", [])
            if "INPUT_ONLY" in behaviors and "INPUT_ONLY" not in inherited:
                inherited.append("INPUT_ONLY")
            if "OUTPUT_ONLY" in behaviors and "OUTPUT_ONLY" not in inherited:
                inherited.append("OUTPUT_ONLY")
    return inherited


def filter_prefixes(fields):
    """Remove fields that are children of other fields in the list."""
    result = []
    for field, behavior in sorted(fields):
        if not any(field.startswith(f + ".") for f, _ in result):
            result.append((field, behavior))
    return result


def write_field_group(lines, header, fields):
    """Write a group of fields with field and reason, grouped by behavior."""
    lines.append(f"\n    {header}:")
    # Group by behavior
    by_behavior = {}
    for field, behavior in fields:
        by_behavior.setdefault(behavior, []).append(field)
    first = True
    for behavior in sorted(by_behavior):
        if not first:
            lines.append("")
        first = False
        reason = f"spec:{behavior.lower()}"
        for field in by_behavior[behavior]:
            lines.append(f"      - field: {field}")
            lines.append(f"        reason: {reason}")


def generate(resource_behaviors):
    """Generate resources.yml."""
    lines = [
        """# Generated, do not edit. API field behaviors from OpenAPI schema.
#
# For manual edits and schema description, see resources.yml.

resources:"""
    ]

    for resource in sorted(resource_behaviors):
        behaviors = resource_behaviors[resource]

        ignore_remote, recreate = [], []
        for field, fb in sorted(behaviors.items()):
            if "OUTPUT_ONLY" in fb:
                ignore_remote.append((field, "OUTPUT_ONLY"))
            elif "INPUT_ONLY" in fb:
                ignore_remote.append((field, "INPUT_ONLY"))
            if "IMMUTABLE" in fb:
                recreate.append((field, "IMMUTABLE"))

        ignore_remote = filter_prefixes(ignore_remote)
        recreate = filter_prefixes(recreate)

        if not ignore_remote and not recreate:
            lines.append(f"\n  # {resource}: no api field behaviors")
            continue

        lines.append(f"\n  {resource}:")

        if recreate:
            write_field_group(lines, "recreate_on_changes", recreate)

        if ignore_remote:
            write_field_group(lines, "ignore_remote_changes", ignore_remote)

    while lines and lines[-1] == "":
        lines.pop()

    return "\n".join(lines)


def main():
    parser = argparse.ArgumentParser(description="Generate resources YAML from the cli.json schema block")
    parser.add_argument("apischema", type=Path, help="Path to cli.json schema block")
    parser.add_argument("apitypes", type=Path, help="Path to apitypes.generated.yml file")
    parser.add_argument("apitypes_override", type=Path, help="Path to apitypes.yml override file")
    parser.add_argument("out_fields", type=Path, help="Path to out.fields.txt file")
    args = parser.parse_args()

    resource_types = parse_apitypes(args.apitypes, args.apitypes_override)
    state_fields, array_element_types = parse_out_fields(args.out_fields)
    schemas = json.loads(args.apischema.read_text())["schemas"]

    resource_behaviors = {}
    for resource, type_name in sorted(resource_types.items()):
        fields = state_fields.get(resource, set())
        print(f"\n{resource}: type={type_name}", file=sys.stderr)
        all_behaviors = get_field_behaviors(schemas, type_name, resource, array_element_types)
        if all_behaviors:
            print(f"  field behaviors from {type_name}:", file=sys.stderr)
            for field in sorted(all_behaviors):
                print(f"    {field}: {all_behaviors[field]}", file=sys.stderr)
        resource_behaviors[resource] = {f: b for f, b in all_behaviors.items() if f in fields}

    print(generate(resource_behaviors))


if __name__ == "__main__":
    main()
