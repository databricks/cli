#!/usr/bin/env python3
"""Diff terraform vs direct create-request bodies to show empty-string handling.

Both engines deploy the same config, so their create requests have the same
shape; any body field one engine sends that the other omits is a field that
engine kept and the other dropped. Comparing the two request sets therefore
yields the difference directly, with no reference list of configured fields:

  out.empty_fields_dropped_by_terraform_only.txt  present in direct, not terraform
  out.empty_fields_dropped_direct_only.txt        present in terraform, not direct

(The direct engine currently sends every "" while terraform drops them, so today
the first file lists the whole gap and the second is empty.)
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


def body_fields(engine):
    """Return {request_path: {body field paths}} for each recorded create request."""
    result = {}
    path = Path(f"out.requests.{engine}.json")
    for req in read_json_many(path.read_text()):
        fields = result.setdefault(req.get("path", ""), set())
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


def only_in(a, b):
    """Return sorted "<path> <field>" present in engine a's request but not b's."""
    out = []
    for path, fields in a.items():
        for field in fields - b.get(path, set()):
            out.append(f"{path} {field}")
    return sorted(out)


def main():
    tf = body_fields("terraform")
    direct = body_fields("direct")

    _write("out.empty_fields_dropped_by_terraform_only.txt", only_in(direct, tf))
    _write("out.empty_fields_dropped_direct_only.txt", only_in(tf, direct))


def _write(name, lines):
    Path(name).write_text("".join(line + "\n" for line in lines))


if __name__ == "__main__":
    main()
