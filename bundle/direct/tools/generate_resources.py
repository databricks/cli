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


def parse_apitypes(path):
    """Parse apitypes.generated.yml to get resource types."""
    data = yaml.safe_load(path.read_text())
    return {resource: types[0] for resource, types in data.items() if types}


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
    """Extract all field behaviors from a schema."""
    if type_name not in schemas:
        return {}

    def extract(schema, prefix, visited, depth):
        if depth > 4:
            return {}
        results = {}
        for name, prop in schema.get("properties", {}).items():
            path = f"{prefix}.{name}" if prefix else name
            behaviors = prop.get("x-databricks-field-behaviors", [])
            if prop.get("x-databricks-immutable") and "IMMUTABLE" not in behaviors:
                behaviors.append("IMMUTABLE")

            if behaviors:
                results[path] = behaviors

            if "$ref" in prop:
                ref = prop["$ref"].split("/")[-1]
                if ref in schemas and ref not in visited:
                    visited.add(ref)
                    results.update(extract(schemas[ref], path, visited, depth + 1))
        return results

    return extract(schemas[type_name], "", set(), 0)


def filter_prefixes(fields):
    """Remove fields that are children of other fields in the list."""
    result = []
    for field, behavior in sorted(fields):
        if not any(field.startswith(f + ".") for f, _ in result):
            result.append((field, behavior))
    return result


def write_field_group(lines, header, fields):
    """Write a group of fields with behavior type comments."""
    lines.append(f"    {header}:")
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
        """Generated, do not edit. API field behaviors from OpenAPI schema.

For manual edits and schema description, see resources.yml.

resources:
"""
    ]

    prev_had_content = False
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
            if prev_had_content:
                lines.append("")
                prev_had_content = False
            lines.append(f"  # {resource}: no api field behaviors")
            continue

        if prev_had_content or lines[-1].startswith("  #"):
            lines.append("")
        lines.append(f"  {resource}:")
        lines.append("")
        prev_had_content = True

        if recreate:
            write_field_group(lines, "recreate_on_changes", recreate)
            lines.append("")

        if ignore_remote:
            write_field_group(lines, "ignore_remote_changes", ignore_remote)

        lines.append("")

    while lines and lines[-1] == "":
        lines.pop()

    return "\n".join(lines)


def main():
    parser = argparse.ArgumentParser(description="Generate resources YAML from OpenAPI schema")
    parser.add_argument("apischema", type=Path, help="Path to OpenAPI schema JSON file")
    parser.add_argument("apitypes", type=Path, help="Path to apitypes.generated.yml file")
    # TODO: add non-generated apitypes.yml here once the need to override generated ones arises
    parser.add_argument("out_fields", type=Path, help="Path to out.fields.txt file")
    args = parser.parse_args()

    resource_types = parse_apitypes(args.apitypes)
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
