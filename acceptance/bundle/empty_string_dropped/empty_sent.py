#!/usr/bin/env python3
"""Classify empty-string handling by diffing terraform vs direct create requests.

Both engines deploy the same config, so their create requests have the same
shape. From the request logs alone we collect, per engine, the set of body
fields sent with an empty-string value ("<request-path> <field>"), then split:

  out.empty_sent_by_terraform_only.txt  sent "" by terraform, not by direct
  out.empty_sent_by_direct_only.txt     sent "" by direct, not by terraform
  out.empty_sent_by_both.txt            sent "" by both

Terraform drops empty optional strings via omitempty (SDKv2 resources), so today
direct_only lists the gap; both/terraform_only capture the exceptions (e.g. a
required field with no omitempty, or a plugin-framework resource that serializes
empty strings).
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


def empty_sent(engine):
    """Return {"<request-path> <field-path>"} for body fields sent with value ""."""
    out = set()
    for req in read_json_many(Path(f"out.requests.{engine}.json").read_text()):
        for field in empty_field_paths(req.get("body", {}), ""):
            out.add(f"{req.get('path', '')} {field}")
    return out


def empty_field_paths(obj, prefix):
    """Yield dotted body-field paths whose value is an empty string."""
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


def main():
    tf = empty_sent("terraform")
    direct = empty_sent("direct")

    _write("out.empty_sent_by_terraform_only.txt", tf - direct)
    _write("out.empty_sent_by_direct_only.txt", direct - tf)
    _write("out.empty_sent_by_both.txt", tf & direct)


def _write(name, entries):
    Path(name).write_text("".join(entry + "\n" for entry in sorted(entries)))


if __name__ == "__main__":
    main()
