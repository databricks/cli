#!/usr/bin/env python3
"""
Generate resources.generated.yml from OpenAPI schema field behaviors.
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
    """Parse out.fields.txt to extract STATE field names per resource."""
    state_fields = {}

    for line in path.read_text().splitlines():
        parts = line.split("\t")
        if len(parts) < 3 or not parts[0].startswith("resources."):
            continue

        field_path, flags = parts[0], parts[2:]
        if "STATE" not in flags and "ALL" not in flags:
            continue

        # Field line: resources.<name>.*.<field>
        match = re.match(r"resources\.([a-z_]+)\.\*\.(.+)", field_path)
        if match and "[*]" not in match.group(2):
            state_fields.setdefault(match.group(1), set()).add(match.group(2))

    return state_fields


def get_field_behaviors(schemas, type_name):
    """Extract field behaviors from a schema, propagating INPUT_ONLY/OUTPUT_ONLY from containers."""
    if type_name not in schemas:
        return {}

    def extract(schema, prefix, visited, depth, inherited):
        if depth > 4:
            return {}
        results = {}
        for name, prop in schema.get("properties", {}).items():
            path = f"{prefix}.{name}" if prefix else name
            behaviors = list(prop.get("x-databricks-field-behaviors", []))
            if prop.get("x-databricks-immutable") and "IMMUTABLE" not in behaviors:
                behaviors.append("IMMUTABLE")
            for b in inherited:
                if b not in behaviors:
                    behaviors.append(b)
            if behaviors:
                results[path] = behaviors
            if "$ref" in prop:
                ref = prop["$ref"].split("/")[-1]
                if ref in schemas and ref not in visited:
                    visited.add(ref)
                    propagate = [b for b in behaviors if b in ("INPUT_ONLY", "OUTPUT_ONLY")]
                    results.update(extract(schemas[ref], path, visited, depth + 1, propagate))
        return results

    # Find INPUT_ONLY/OUTPUT_ONLY from container types that reference this type
    inherited = find_inherited_behaviors(schemas, type_name)
    return extract(schemas[type_name], "", set(), 0, inherited)


def find_inherited_behaviors(schemas, type_name):
    """Find INPUT_ONLY/OUTPUT_ONLY behaviors from containers that reference type_name."""
    inherited = []
    for container_schema in schemas.values():
        for field_prop in container_schema.get("properties", {}).values():
            if field_prop.get("$ref", "").split("/")[-1] != type_name:
                continue
            behaviors = field_prop.get("x-databricks-field-behaviors", [])
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
    """Write a group of fields with behavior type comments."""
    lines.append(f"\n    {header}:")
    by_behavior = {}
    for field, behavior in fields:
        by_behavior.setdefault(behavior, []).append(field)
    first = True
    for behavior in sorted(by_behavior):
        if not first:
            lines.append("")
        first = False
        lines.append(f"      # {behavior}:")
        for field in by_behavior[behavior]:
            lines.append(f"      - {field}")


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
    parser = argparse.ArgumentParser(description="Generate resources YAML from OpenAPI schema")
    parser.add_argument("apischema", type=Path, help="Path to OpenAPI schema JSON file")
    parser.add_argument("apitypes", type=Path, help="Path to apitypes.generated.yml file")
    parser.add_argument("apitypes_override", type=Path, help="Path to apitypes.yml override file")
    parser.add_argument("out_fields", type=Path, help="Path to out.fields.txt file")
    args = parser.parse_args()

    resource_types = parse_apitypes(args.apitypes, args.apitypes_override)
    state_fields = parse_out_fields(args.out_fields)
    schemas = json.loads(args.apischema.read_text()).get("components", {}).get("schemas", {})

    resource_behaviors = {}
    for resource, type_name in sorted(resource_types.items()):
        fields = state_fields.get(resource, set())
        print(f"\n{resource}: type={type_name}", file=sys.stderr)
        all_behaviors = get_field_behaviors(schemas, type_name)
        if all_behaviors:
            print(f"  field behaviors from {type_name}:", file=sys.stderr)
            for field in sorted(all_behaviors):
                print(f"    {field}: {all_behaviors[field]}", file=sys.stderr)
        resource_behaviors[resource] = {f: b for f, b in all_behaviors.items() if f in fields}

    print(generate(resource_behaviors))


if __name__ == "__main__":
    main()
