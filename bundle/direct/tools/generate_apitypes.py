#!/usr/bin/env python3
"""
Find candidate types from apischema.json that match resource shapes in out.fields.txt and save them (apitypes.generated.yml).

The types are found based on top level field overlap between bundle schema and API type.
"""

import argparse
import json
import re
import sys
from collections import Counter
from pathlib import Path


def parse_out_fields(path):
    """Parse out.fields.txt to extract top-level STATE field names per resource."""
    resource_fields = {}

    for line in path.read_text().splitlines():
        parts = line.split("\t")
        if len(parts) < 3 or not parts[0].startswith("resources."):
            continue

        field_path, flags = parts[0], parts[2:]
        if "STATE" not in flags and "ALL" not in flags:
            continue

        # Field line: resources.<name>.*.<field>
        match = re.match(r"resources\.([a-z_]+)\.\*\.([a-z_]+)$", field_path)
        if match:
            resource_fields.setdefault(match.group(1), set()).add(match.group(2))

    return resource_fields


def get_schema_fields(schemas):
    """Get top-level field names for each schema type."""
    schema_fields = {}
    for name, schema in schemas.items():
        props = schema.get("properties", {})
        if props:
            schema_fields[name] = set(props.keys())
    return schema_fields


def main():
    parser = argparse.ArgumentParser(description="Generate apitypes.yml from OpenAPI schema")
    parser.add_argument("apischema", type=Path, help="Path to OpenAPI schema JSON file")
    parser.add_argument("out_fields", type=Path, help="Path to out.fields.txt file")
    args = parser.parse_args()

    resource_fields = parse_out_fields(args.out_fields)
    schemas = json.loads(args.apischema.read_text()).get("components", {}).get("schemas", {})
    schema_fields = get_schema_fields(schemas)

    field_counts = Counter()
    for fields in schema_fields.values():
        field_counts.update(fields)

    field_weights = {f: 1.0 / c for f, c in field_counts.items()}

    # Fields to ignore in resource definitions (handled separately)
    ignore_resource_fields = {"permissions", "grants"}

    top_matches = {}

    for resource in sorted(resource_fields):
        res_fields = resource_fields[resource] - ignore_resource_fields
        max_score = sum(field_weights.get(f, 1.0) for f in res_fields)

        # Find matching schema types
        candidates = []
        for schema_name, s_fields in schema_fields.items():
            overlap = res_fields & s_fields
            if not overlap:
                continue
            match_score = sum(field_weights.get(f, 1.0) for f in overlap)
            extra_resource = len(res_fields - s_fields)
            extra_schema = len(s_fields - res_fields)
            score = match_score - 0.0001 * (extra_resource + extra_schema)
            pct = score / max_score * 100 if max_score > 0 else 0
            candidates.append((pct, score, schema_name, overlap, s_fields))

        candidates.sort(reverse=True)
        if candidates:
            top_matches[resource] = candidates[0][2]

        print(f"\n{resource}: {len(res_fields)} fields, max_score={max_score:.2f}", file=sys.stderr)
        top_pct = candidates[0][0] if candidates else 0
        for pct, score, schema_name, overlap, s_fields in candidates[:5]:
            # Only show if >= 80% or within 20% of top entry
            if pct < 80 and (top_pct - pct) >= 20:
                continue
            missing = res_fields - s_fields
            extra = s_fields - res_fields
            print(f"  {pct:5.1f}% {schema_name}", file=sys.stderr)
            print(f"         # matching: {sorted(overlap)}", file=sys.stderr)
            print(f"         # in_resource_only: {sorted(missing)}", file=sys.stderr)
            print(f"         # in_schema_only: {sorted(extra)}", file=sys.stderr)

    print("# Generated, do not edit. Override via apitypes.yml")
    for resource in sorted(top_matches):
        print("")
        print(f"{resource}:")
        print(f"  - {top_matches[resource]}")


if __name__ == "__main__":
    main()
