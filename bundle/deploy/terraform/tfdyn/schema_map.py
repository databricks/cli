#!/usr/bin/env python3
"""Build a bidirectional map between DABs fields and Terraform resource fields.

Usage:
    python3 schema_map.py <dabs_fields_file> <terraform_schema_json>

Inputs:
    dabs_fields_file:     acceptance/bundle/refschema/out.fields.txt
    terraform_schema_json: full_schema.json from terraform-provider-databricks

Output:
    Tab-separated lines: dabs_path \t tf_resource_type \t tf_path \t status
    status is one of: match, dabs_only, tf_only, renamed
"""

import json
import sys
from pathlib import Path


# DABs resource type -> terraform resource type
RESOURCE_TYPE_MAP = {
    "alerts": "databricks_alert_v2",
    "apps": "databricks_app",
    "clusters": "databricks_cluster",
    "dashboards": "databricks_dashboard",
    "database_catalogs": "databricks_database_database_catalog",
    "database_instances": "databricks_database_instance",
    "experiments": "databricks_mlflow_experiment",
    "jobs": "databricks_job",
    "model_serving_endpoints": "databricks_model_serving",
    "models": "databricks_mlflow_model",
    "pipelines": "databricks_pipeline",
    "postgres_branches": "databricks_postgres_branch",
    "postgres_endpoints": "databricks_postgres_endpoint",
    "postgres_projects": "databricks_postgres_project",
    "quality_monitors": "databricks_quality_monitor",
    "registered_models": "databricks_registered_model",
    "schemas": "databricks_schema",
    "secret_scopes": "databricks_secret_scope",
    "sql_warehouses": "databricks_sql_endpoint",
    "synced_database_tables": "databricks_database_synced_database_table",
    "volumes": "databricks_volume",
}

# Rename rules per resource type.
# Each rule: (dabs_parent_glob, {old_key: new_key})
# dabs_parent_glob uses the DABs path (before renames). Empty string = top-level.
# Globs support "*" to match any single path segment.
RENAME_RULES = {
    "jobs": [
        (
            "",
            {"tasks": "task", "job_clusters": "job_cluster", "parameters": "parameter", "environments": "environment"},
        ),
        (
            "git_source",
            {
                "git_branch": "branch",
                "git_commit": "commit",
                "git_provider": "provider",
                "git_tag": "tag",
                "git_url": "url",
            },
        ),
        ("tasks", {"libraries": "library"}),
        ("tasks.for_each_task.task", {"libraries": "library"}),
    ],
    "pipelines": [
        ("", {"libraries": "library", "clusters": "cluster", "notifications": "notification"}),
    ],
}


def parse_dabs_fields(path):
    """Parse DABs fields file into a dict of {path: (go_type, tags)}."""
    fields = {}
    for line in Path(path).read_text().splitlines():
        line = line.strip()
        if not line:
            continue
        parts = line.split("\t")
        if len(parts) >= 2:
            fields[parts[0]] = (parts[1], parts[2] if len(parts) > 2 else "")
    return fields


def parse_tf_schema(path):
    """Parse Terraform schema JSON, return dict of {resource_type: set of field paths}."""
    data = json.loads(Path(path).read_text())
    provider = data["provider_schemas"]["registry.terraform.io/databricks/databricks"]
    resources = provider["resource_schemas"]

    result = {}
    for res_type, res_schema in resources.items():
        fields = set()
        _flatten_tf_block(res_schema["block"], "", fields)
        result[res_type] = fields
    return result


def _flatten_tf_block(block, prefix, fields):
    """Recursively flatten a Terraform block into dot-separated field paths."""
    if "attributes" in block:
        for name, attr in block["attributes"].items():
            fields.add(f"{prefix}{name}")
            if "nested_type" in attr:
                _flatten_tf_nested(attr["nested_type"], f"{prefix}{name}.", fields)
    if "block_types" in block:
        for name, bt in block["block_types"].items():
            fields.add(f"{prefix}{name}")
            if "block" in bt:
                _flatten_tf_block(bt["block"], f"{prefix}{name}.", fields)


def _flatten_tf_nested(nested_type, prefix, fields):
    """Recursively flatten nested_type attributes."""
    if "attributes" in nested_type:
        for name, attr in nested_type["attributes"].items():
            fields.add(f"{prefix}{name}")
            if "nested_type" in attr:
                _flatten_tf_nested(attr["nested_type"], f"{prefix}{name}.", fields)


def apply_renames(dabs_type, field_path):
    """Apply key renames for a given DABs resource type and field path.

    The field_path uses dots as separators, with array notation stripped.
    Renames are applied segment-by-segment, checking if the parent of the
    current segment matches a rename rule's parent glob pattern.
    """
    rules = RENAME_RULES.get(dabs_type, [])
    if not rules:
        return field_path

    parts = field_path.split(".")
    result = []

    for i, part in enumerate(parts):
        # The parent path (in original DABs terms) is parts[0:i] joined
        parent = ".".join(parts[:i])

        renamed = False
        for rule_parent, renames in rules:
            if part not in renames:
                continue
            if _match_parent(parent, rule_parent):
                result.append(renames[part])
                renamed = True
                break

        if not renamed:
            result.append(part)

    return ".".join(result)


def _match_parent(actual_parent, rule_parent):
    """Check if actual_parent matches the rule_parent pattern.

    Both are dot-separated paths. Empty rule_parent matches empty actual_parent (top-level).
    """
    if rule_parent == "":
        return actual_parent == ""

    rule_parts = rule_parent.split(".")
    actual_parts = actual_parent.split(".") if actual_parent else []

    # The actual parent must end with the rule pattern
    if len(actual_parts) < len(rule_parts):
        return False

    # Match from the end
    offset = len(actual_parts) - len(rule_parts)
    for i, rp in enumerate(rule_parts):
        ap = actual_parts[offset + i]
        if rp != "*" and rp != ap:
            return False
    return True


def normalize_dabs_path(path):
    """Convert DABs path like resources.jobs.*.foo.bar[*].baz to (resource_type, field_path).

    Returns (resource_type, field_path) where field_path uses dots only (no array notation).
    """
    parts = path.split(".")
    if len(parts) < 3 or parts[0] != "resources":
        return None, None
    resource_type = parts[1]
    if len(parts) < 4 or parts[2] != "*":
        return resource_type, None
    # Join remaining, strip array markers
    field_parts = []
    for p in parts[3:]:
        if p == "[*]" or p == "*":
            continue
        clean = p.replace("[*]", "")
        if clean:
            field_parts.append(clean)
    return resource_type, ".".join(field_parts) if field_parts else None


def main():
    if len(sys.argv) != 3:
        print(__doc__)
        sys.exit(1)

    dabs_fields_file = sys.argv[1]
    tf_schema_file = sys.argv[2]

    dabs_fields = parse_dabs_fields(dabs_fields_file)
    tf_schemas = parse_tf_schema(tf_schema_file)

    # Group DABs fields by resource type
    by_resource = {}
    for path in dabs_fields:
        res_type, field_path = normalize_dabs_path(path)
        if res_type is None:
            continue
        if res_type not in by_resource:
            by_resource[res_type] = {}
        if field_path:
            by_resource[res_type][field_path] = path

    # For each resource type, compare fields
    print("dabs_path\ttf_resource\ttf_path\tstatus")
    for dabs_type in sorted(by_resource):
        tf_type = RESOURCE_TYPE_MAP.get(dabs_type)
        if not tf_type:
            for field_path, dabs_path in sorted(by_resource[dabs_type].items()):
                print(f"{dabs_path}\t?\t?\tno_tf_mapping")
            continue

        tf_fields = tf_schemas.get(tf_type, set())
        dabs_type_fields = by_resource[dabs_type]

        matched_tf = set()
        for field_path, dabs_path in sorted(dabs_type_fields.items()):
            tf_path = apply_renames(dabs_type, field_path)

            if tf_path in tf_fields:
                status = "renamed" if tf_path != field_path else "match"
                matched_tf.add(tf_path)
                print(f"{dabs_path}\t{tf_type}\t{tf_path}\t{status}")
            elif field_path in tf_fields:
                matched_tf.add(field_path)
                print(f"{dabs_path}\t{tf_type}\t{field_path}\tmatch")
            else:
                print(f"{dabs_path}\t{tf_type}\t{tf_path}\tdabs_only")

        # TF-only fields
        for tf_path in sorted(tf_fields - matched_tf):
            print(f"?\t{tf_type}\t{tf_path}\ttf_only")


if __name__ == "__main__":
    main()
