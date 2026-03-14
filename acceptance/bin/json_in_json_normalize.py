#!/usr/bin/env python3
r"""
Process json fields in the following format and for matching fields re-normalize JSON if possible.

For example, if target field is serialized_dashboard and we see this:

 "serialized_dashboard": "{\n  \"pages\": [\n    {\n      \"displayName\": \"Test Dashboard\",\n      \"name\": \"test-page\",\n      \"pageType\": \"PAGE_TYPE_CANVAS\"\n    }\n  ]\n}\n",


we'll convert it to this:

 "serialized_dashboard": "{\"pages\":[{\"displayName\":\"Test Dashboard\",\"name\": \"test-page\",\"pageType\":\"PAGE_TYPE_CANVAS\"}]}",
"""

import argparse
import json
import re

line_re = re.compile(r'^(\s*"(\w+)":\s*)"(.*)"(,?)$')


def normalize_json_field(line, fields):
    r"""
    Normalize JSON in a field if the field name matches.

    >>> normalize_json_field('  "foo": "{\\"a\\": 1}",', {'foo'})
    '  "foo": "{\\"a\\":1}",'
    >>> normalize_json_field('  "foo": "{\\"a\\": 1}",', {'bar'})
    '  "foo": "{\\"a\\": 1}",'
    >>> normalize_json_field('  "other": "plain string",', {'foo'})
    '  "other": "plain string",'
    """
    m = line_re.match(line.rstrip("\n"))
    if not m:
        return line

    prefix, field_name, value, suffix = m.groups()

    if fields and field_name not in fields:
        return line

    # if specific fields are not requested, process all where values start with { or [
    if not fields and value[:1] not in "{[":
        return line

    try:
        unescaped = json.loads('"' + value + '"')
        parsed = json.loads(unescaped)
        normalized = json.dumps(parsed, separators=(",", ":"), sort_keys=True)
        escaped = json.dumps(normalized)[1:-1]
    except json.JSONDecodeError:
        return line
    else:
        return f'{prefix}"{escaped}"{suffix}\n'


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("filename")
    parser.add_argument("fields", nargs="*")
    args = parser.parse_args()

    fields = set(args.fields)
    result = []

    with open(args.filename) as fobj:
        for line in fobj:
            result.append(normalize_json_field(line, fields))

    with open(args.filename, "w") as fobj:
        fobj.writelines(result)


if __name__ == "__main__":
    main()
