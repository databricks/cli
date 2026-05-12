#!/usr/bin/env python3
import argparse
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from print_requests import read_json_many


def gron(obj, path="json", noindex=False):
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

    >>> gron({"items": ["apple", "banana"]}, noindex=True)
    json.items[] = "apple";
    json.items[] = "banana";

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
                gron(obj[key], f"{path}.{key}", noindex=noindex)
    elif isinstance(obj, list):
        if not obj:
            print(f"{path} = [];")
        else:
            for i, item in enumerate(obj):
                index = "[]" if noindex else f"[{i}]"
                gron(item, f"{path}{index}", noindex=noindex)
    else:
        print(f"{path} = {json.dumps(obj)};")


def sort_arrays(obj, keys):
    """Recursively sort arrays whose dict key matches one in `keys`.

    Sort uses a canonical JSON repr so the order is content-determined and stable
    across runs. Arrays not under a matching key keep their original order.
    """
    if isinstance(obj, dict):
        for k, v in obj.items():
            if isinstance(v, list):
                items = [sort_arrays(item, keys) for item in v]
                if k in keys:
                    items.sort(key=lambda x: json.dumps(x, sort_keys=True))
                obj[k] = items
            else:
                obj[k] = sort_arrays(v, keys)
        return obj
    elif isinstance(obj, list):
        return [sort_arrays(item, keys) for item in obj]
    return obj


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--noindex", action="store_true")
    parser.add_argument(
        "--sort-arrays",
        default="",
        help="Comma-separated dict keys whose array values should be sorted by content (e.g. acls,access_control_list)",
    )
    parser.add_argument("file", nargs="?")
    args = parser.parse_args()

    if args.file:
        content = Path(args.file).read_text()
    else:
        content = sys.stdin.read()

    data = read_json_many(content)
    if len(data) == 1:
        data = data[0]

    if args.sort_arrays:
        keys = set(args.sort_arrays.split(","))
        data = sort_arrays(data, keys)

    gron(data, noindex=args.noindex)


if __name__ == "__main__":
    main()
