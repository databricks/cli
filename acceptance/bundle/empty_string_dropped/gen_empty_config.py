#!/usr/bin/env python3
"""Generate databricks.yml + empty_fields.txt covering every settable string field.

For each resource we have a deployable base config (the invariant fixtures), we
overlay every top-level optional string field set to "" and record it in
empty_fields.txt as "<create-path> <field>". The comparison test then deploys
through both engines and diffs which empties each drops.

Sources:
  - out.fields.txt: field inventory (path, Go type, flags). type=="string"
    distinguishes plain strings from enums (e.g. compute.RuntimeEngine), so
    filtering to "string" automatically skips enums, which terraform rejects
    when empty.
  - resources.generated.yml: output_only classification (fields the user cannot
    set), excluded from the overlay.

Run from the repo root; writes databricks.yml and empty_fields.txt in the test
directory:
  acceptance/bundle/empty_string_dropped/gen_empty_config.py
"""

import re
import sys
from pathlib import Path

import yaml

FIELDS = Path("acceptance/bundle/refschema/out.fields.txt")
GENERATED = Path("bundle/direct/dresources/resources.generated.yml")
BASE = Path("acceptance/bundle/empty_string_dropped/base.yml")

# Bundle-framework fields present on every resource; not real API inputs.
FRAMEWORK_FIELDS = {"id", "url", "modified_status"}

# Fields that make terraform error at plan time, so no request is recorded and
# there is nothing to compare. Keyed by resource type. Direct-engine behavior for
# these is covered elsewhere.
TERRAFORM_ERRORS = {
    # ConflictsWith: a cluster may set node type OR instance pool, not both, and
    # terraform rejects even empty strings for the losing side.
    "clusters": {"instance_pool_id", "driver_instance_pool_id", "driver_node_type_id"},
    "pipelines": {
        # Enum validation ("" not accepted) — these are typed `string` in
        # out.fields.txt but the TF provider validates them as enums.
        "channel",
        "edition",
        # ConflictsWith pairs: catalog<->storage, schema<->target.
        "catalog",
        "storage",
        "schema",
        "target",
    },
    "sql_warehouses": {
        # Computed/unconfigurable in the TF provider; setting any value errors.
        "creator_name",
    },
}

# Map resource type (plural, as in databricks.yml) to its create-request path.
CREATE_PATHS = {
    "clusters": "clusters/create",
    "jobs": "jobs/create",
    "pipelines": "pipelines",
    "apps": "apps",
    "instance_pools": "instance-pools/create",
    "sql_warehouses": "warehouses",
}


def settable_string_fields():
    """Return {resource_type: [field, ...]} of top-level settable string fields."""
    pat = re.compile(r"^resources\.([a-z_]+)\.\*\.([a-z_]+)$")
    result = {}
    for line in FIELDS.read_text().splitlines():
        parts = line.split("\t")
        if len(parts) < 3:
            continue
        path, typ, flag = parts[0], parts[1], parts[2]
        if typ != "string" or flag not in ("ALL", "INPUT"):
            continue
        m = pat.match(path)
        if not m:
            continue
        result.setdefault(m.group(1), []).append(m.group(2))
    return result


def output_only_fields():
    """Return {resource_type: {field, ...}} the backend computes (user can't set)."""
    gen = yaml.safe_load(GENERATED.read_text()) or {}
    result = {}
    for rtype, spec in (gen.get("resources") or {}).items():
        fields = set()
        for entry in (spec or {}).get("ignore_remote_changes") or []:
            if str(entry.get("reason", "")).startswith("spec:output_only"):
                fields.add(entry["field"])
        result[rtype] = fields
    return result


def main():
    settable = settable_string_fields()
    output_only = output_only_fields()
    base = yaml.safe_load(BASE.read_text())

    empty_fields = []
    resources = base.setdefault("resources", {})

    for rtype, entries in sorted(resources.items()):
        if rtype not in CREATE_PATHS:
            sys.exit(f"no create path mapped for resource type {rtype!r}")
        create_path = CREATE_PATHS[rtype]
        excluded = output_only.get(rtype, set()) | FRAMEWORK_FIELDS | TERRAFORM_ERRORS.get(rtype, set())
        overlay = sorted(f for f in settable.get(rtype, []) if f not in excluded)

        for cfg in dict(sorted(entries.items())).values():
            for field in overlay:
                # Don't overwrite fields the base config sets: those are the
                # required / meaningful values that make the resource deployable
                # (e.g. warehouse name, cluster node_type_id). Emptying them
                # would trip client-side "required" validation and abort the
                # whole deploy before any request is recorded.
                if field in cfg:
                    continue
                cfg[field] = ""
                empty_fields.append(f"{create_path} {field}")

    testdir = Path("acceptance/bundle/empty_string_dropped")
    lines = sorted({f + "\n" for f in empty_fields})
    (testdir / "empty_fields.txt").write_text(
        '# Generated by gen_empty_config.py. Fields set to "" in databricks.yml,\n'
        '# as "<create-path> <body-field>".\n' + "".join(lines)
    )
    (testdir / "databricks.yml").write_text(
        "# Generated by gen_empty_config.py. Do not edit; edit base.yml and regenerate.\n"
        + yaml.dump(base, sort_keys=True, default_flow_style=False)
    )


if __name__ == "__main__":
    main()
