#!/usr/bin/env python3
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from print_requests import read_json_many


def gron(obj, path="json"):
    """Flatten JSON into greppable assignments.

    The path parameter defaults to "json" to match the original gron tool,
    which treats the input as a JavaScript variable named "json".

    Container declarations are only printed for empty dicts/lists.
    This differs from https://github.com/tomnomnom/gron which always prints
    container declarations to allow reconstruction via ungron. We don't need
    reversibility - we only care about making JSON greppable.

    >>> gron({"name": "Tom", "age": 30})
    json.name = "Tom";
    json.age = 30;

    >>> gron({"items": ["apple", "banana"]})
    json.items[0] = "apple";
    json.items[1] = "banana";

    >>> gron({"tasks": [{"libraries": [{"whl": "file.whl"}]}]})
    json.tasks[0].libraries[0].whl = "file.whl";

    >>> gron({"empty": {}, "items": []})
    json.empty = {};
    json.items = [];
    """
    if isinstance(obj, dict):
        if not obj:
            print(f"{path} = {{}};")
        else:
            for key in obj:
                gron(obj[key], f"{path}.{key}")
    elif isinstance(obj, list):
        if not obj:
            print(f"{path} = [];")
        else:
            for i, item in enumerate(obj):
                gron(item, f"{path}[{i}]")
    else:
        print(f"{path} = {json.dumps(obj)};")


def main():
    if len(sys.argv) > 1:
        with open(sys.argv[1]) as f:
            content = f.read()
        data = read_json_many(content)
        if len(data) == 1:
            data = data[0]
    else:
        content = sys.stdin.read()
        data = read_json_many(content)
        if len(data) == 1:
            data = data[0]

    gron(data)


if __name__ == "__main__":
    main()
