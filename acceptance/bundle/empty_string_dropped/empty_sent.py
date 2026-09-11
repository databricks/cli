#!/usr/bin/env python3
"""Classify empty-string handling by diffing terraform vs direct create requests.

Both engines deploy the same config, so their create requests have the same
shape. From the request logs we collect, per engine, the body fields sent with an
empty-string value; comparing the two sets splits three ways:

  out.empty_sent_by_terraform_only.txt  sent "" by terraform, not by direct
  out.empty_sent_by_direct_only.txt     sent "" by direct, not by terraform
  out.empty_sent_by_both.txt            sent "" by both

The fourth bucket -- fields set to "" in config that BOTH engines dropped -- is
not visible in the request logs (they're absent from both bodies), so it is
recovered from databricks.yml:

  out.empty_dropped_by_both.txt         set "" in config, sent by neither

Terraform drops empty optional strings via omitempty (SDKv2 resources), so today
direct_only lists the gap; both/terraform_only capture the exceptions (a required
field with no omitempty, or a plugin-framework resource that serializes "").
"""

import json
from pathlib import Path

# Substring uniquely identifying each resource type's create-request path, used
# to attribute a config field to the request it should appear in.
CREATE_PATHS = {
    "apps": "/apps",
    "clusters": "clusters/create",
    "jobs": "jobs/create",
    "pipelines": "/pipelines",
    "sql_warehouses": "/warehouses",
    "models": "/mlflow/registered-models",
    "experiments": "/experiments/create",
    "registered_models": "/unity-catalog/models",
    "schemas": "/unity-catalog/schemas",
    "volumes": "/unity-catalog/volumes",
    "database_instances": "/database/instances",
}


def read_json_many(text):
    """Decode concatenated JSON objects (the out.requests.<engine>.json format)."""
    decoder = json.JSONDecoder()
    objects = []
    pos = 0
    while pos < len(text):
        while pos < len(text) and text[pos].isspace():
            pos += 1
        if pos >= len(text):
            break
        obj, pos = decoder.raw_decode(text, pos)
        objects.append(obj)
    return objects


def empty_field_paths(obj, prefix):
    """Yield dotted field paths whose value is an empty string."""
    if isinstance(obj, dict):
        for key, value in obj.items():
            child = f"{prefix}.{key}" if prefix else key
            if value == "":
                yield child
            else:
                yield from empty_field_paths(value, child)
    elif isinstance(obj, list):
        for item in obj:
            yield from empty_field_paths(item, prefix)


def empty_sent(engine):
    """Return {"<request-path> <field-path>"} for body fields sent with value ""."""
    out = set()
    for req in read_json_many(Path(f"out.requests.{engine}.json").read_text()):
        for field in empty_field_paths(req.get("body", {}), ""):
            out.add(f"{req.get('path', '')} {field}")
    return out


def config_empty_fields():
    """Return {"<create-path> <field-path>"} for every "" leaf in databricks.yml.

    Uses a minimal indentation parser (the harness python is stdlib-only, no
    PyYAML). databricks.yml is generator output with a fixed 2-space style, block
    mappings, and "- " list items -- not arbitrary YAML.
    """
    out = set()
    # stack of (indent, key) for the current mapping path within a resource
    stack = []
    for raw in Path("databricks.yml").read_text().splitlines():
        if not raw.strip() or raw.lstrip().startswith("#"):
            continue
        indent = len(raw) - len(raw.lstrip(" "))
        line = raw.strip()
        # A "- " list item shares its parent's field path; treat the inline
        # "key: value" after it at the item indent.
        if line.startswith("- "):
            line = line[2:]
            indent += 2
        if ":" not in line:
            continue
        key, _, value = line.partition(":")
        key, value = key.strip(), value.strip()
        while stack and stack[-1][0] >= indent:
            stack.pop()
        path = [k for _, k in stack] + [key]
        # resources.<rtype>.<name>.<field...>: the create path is fixed by rtype.
        if len(path) >= 2 and path[0] == "resources" and path[1] in CREATE_PATHS:
            if value in ("''", '""'):
                field = ".".join(path[3:])  # drop resources.<rtype>.<name>
                out.add(f"{CREATE_PATHS[path[1]]} {field}")
        if not value:
            stack.append((indent, key))
    return out


def sent_fields(*engines):
    """Return {"<request-path> <field>"} for ALL body fields (any value) sent."""
    out = set()
    for engine in engines:
        for req in read_json_many(Path(f"out.requests.{engine}.json").read_text()):
            for field in all_field_paths(req.get("body", {}), ""):
                out.add(f"{req.get('path', '')} {field}")
    return out


def all_field_paths(obj, prefix):
    """Yield every dotted field path present in a body, regardless of value."""
    if isinstance(obj, dict):
        for key, value in obj.items():
            child = f"{prefix}.{key}" if prefix else key
            yield child
            yield from all_field_paths(value, child)
    elif isinstance(obj, list):
        for item in obj:
            yield from all_field_paths(item, prefix)


def sent_somewhere(entry, sent):
    """True if entry (create-path substring + field) matches any sent full-path entry."""
    create_path, field = entry.split(" ", 1)
    return any(create_path in path and field == f for path, f in (s.split(" ", 1) for s in sent))


def main():
    tf = empty_sent("terraform")
    direct = empty_sent("direct")

    _write("out.empty_sent_by_terraform_only.txt", tf - direct)
    _write("out.empty_sent_by_direct_only.txt", direct - tf)
    _write("out.empty_sent_by_both.txt", tf & direct)

    # Dropped by both: set to "" in config but present in neither request body
    # (present with any value = not dropped, so match on field presence).
    present = sent_fields("terraform", "direct")
    dropped_by_both = {e for e in config_empty_fields() if not sent_somewhere(e, present)}
    _write("out.empty_dropped_by_both.txt", dropped_by_both)


def _write(name, entries):
    Path(name).write_text("".join(entry + "\n" for entry in sorted(entries)))


if __name__ == "__main__":
    main()
