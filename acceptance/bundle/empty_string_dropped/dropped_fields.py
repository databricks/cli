#!/usr/bin/env python3
"""Compute which empty-string fields each engine dropped from create requests.

Reads recorded requests (out.requests.<engine>.json) for both engines plus the
list of fields set to "" in config (empty_fields.txt, one "<create-path> <field>"
per line). For each engine a field counts as "dropped" when it was set empty in
config but is absent from that engine's matching create-request body. Emits four
files so the direct/terraform difference is visible and reviewable in a diff.
"""

import json
from pathlib import Path


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


def present_fields(engine):
    """Map create-request path (e.g. "clusters/create") -> set of body field paths."""
    result = {}
    path = Path(f"out.requests.{engine}.json")
    if not path.exists():
        return result
    for req in read_json_many(path.read_text()):
        # Normalize "/api/2.1/clusters/create" -> "clusters/create".
        key = "/".join(req.get("path", "").rsplit("/", 2)[-2:])
        fields = result.setdefault(key, set())
        _collect(req.get("body", {}), "", fields)
    return result


def _collect(obj, prefix, out):
    if isinstance(obj, dict):
        for key, value in obj.items():
            child = f"{prefix}.{key}" if prefix else key
            out.add(child)
            _collect(value, child, out)
    elif isinstance(obj, list):
        for item in obj:
            _collect(item, prefix, out)


def dropped(empty_fields, present):
    """Return "<path> <field>" entries that were set empty but not sent."""
    out = []
    for create_path, field in empty_fields:
        if field not in present.get(create_path, set()):
            out.append(f"{create_path} {field}")
    return out


def main():
    empty_fields = []
    for line in Path("empty_fields.txt").read_text().splitlines():
        line = line.strip()
        if line and not line.startswith("#"):
            create_path, field = line.split()
            empty_fields.append((create_path, field))
    empty_fields.sort()

    tf = present_fields("terraform")
    direct = present_fields("direct")

    dropped_tf = dropped(empty_fields, tf)
    dropped_direct = dropped(empty_fields, direct)

    tf_only = [f for f in dropped_tf if f not in dropped_direct]
    direct_only = [f for f in dropped_direct if f not in dropped_tf]

    _write("out.empty_fields_dropped_by_terraform.txt", dropped_tf)
    _write("out.empty_fields_dropped_by_direct.txt", dropped_direct)
    _write("out.empty_fields_dropped_by_terraform_only.txt", tf_only)
    _write("out.empty_fields_dropped_direct_only.txt", direct_only)


def _write(name, fields):
    Path(name).write_text("".join(f + "\n" for f in fields))


if __name__ == "__main__":
    main()
