#!/usr/bin/env python3
"""
Analyzes out.requests.json in test root dir and pretty prints HTTP requests.

By default ignores requests with method==GET (unless --get option is passed).
Free standing arguments are substrings matching path.
If argument starts with ! then it's a negation filter.

Examples:
  print_requests.py //jobs                     # Show non-GET requests with /jobs in path
  print_requests.py --get //jobs               # Show all requests with /jobs in path
  print_requests.py --sort '^//import-file/'   # Show non-GET requests, exclude /import-file/, sort output
  print_requests.py --keep //jobs              # Show requests and do not delete out.requests.json afterwards

This replaces custom jq wrappers like:
  jq --sort-keys 'select(.method != "GET" and (.path | contains("/jobs")))' < out.requests.txt

>>> test_requests = [
...     {"method": "GET", "path": "/api/2.0/clusters/list"},
...     {"method": "POST", "path": "/api/2.1/jobs/create", "body": {"name": "test"}},
...     {"method": "GET", "path": "/api/2.0/jobs/123"},
...     {"method": "PUT", "path": "/api/2.0/jobs/123", "body": {"name": "updated"}},
...     {"method": "POST", "path": "/api/2.0/workspace/import-file/test.py"},
...     {"method": "DELETE", "path": "/api/2.0/jobs/123"}
... ]

>>> def short_name(x):
...   ind = test_requests.index(x)
...   return f'R{ind} {x["method"]}'
>>> def test(*args):
...   r = filter_requests(*args)
...   for x in r:
...      print(short_name(x))

>>> test(test_requests, ["//jobs"], False, False)
R1 POST
R3 PUT
R5 DELETE

>>> test(test_requests, ["//jobs"], True, False)
R1 POST
R2 GET
R3 PUT
R5 DELETE

>>> test(test_requests, ["^//import-file/"], False, False)
R1 POST
R3 PUT
R5 DELETE

>>> # Test multiple positive filters (OR logic)
>>> test(test_requests, ["//clusters", "//import-file"], True, False)
R0 GET
R4 POST

>>> # Test positive + negative filters (AND logic)
>>> test(test_requests, ["//api", "^/jobs"], False, False)
R4 POST
"""

import os
import sys
import json
import argparse
from pathlib import Path


# I've originally tried ADD_PREFIX to be empty, so you can just do "print_requests.py /jobs"
# However, that causes test to fail on Windows CI because "/jobs" becomes "C:/Program Files/Git/jobs"
# This behaviour can be disabled with MSYS_NO_PATHCONV=1 but that causes other failures, so we require extra slash here.
ADD_PREFIX = "/"
NEGATE_PREFIX = "^/"


def read_json_many(s):
    result = []

    try:
        dec = json.JSONDecoder()
        pos = 0
        n = len(s)
        while True:
            # skip whitespace between objects
            while pos < n and s[pos].isspace():
                pos += 1
            if pos >= n:
                break
            obj, idx = dec.raw_decode(s, pos)
            result.append(obj)
            pos = idx

    except Exception as ex:
        sys.exit(str(ex))

    if not result and s:
        sys.stderr.write(f"WARNING: could not parse {len(s)} chars: {s!r}\n")

    return result


# quick self-test
test = '{"method": "GET"}\n{"method":\n"POST"\n}\n'
result = read_json_many(test)
assert result == [{"method": "GET"}, {"method": "POST"}], result


def filter_requests(requests, path_filters, include_get, should_sort):
    """Filter requests based on method and path filters."""
    positive_filters = []
    negative_filters = []

    for f in path_filters:
        if f.startswith(ADD_PREFIX):
            positive_filters.append(f.removeprefix(ADD_PREFIX))
        elif f.startswith(NEGATE_PREFIX):
            negative_filters.append(f.removeprefix(NEGATE_PREFIX))
        else:
            sys.exit(f"Unrecognized filter: {f!r}")

    filtered_requests = []
    for req in requests:
        # Skip GET requests unless include_get is True
        if req.get("method") == "GET" and not include_get:
            continue

        # Apply path filters
        path = req.get("path", "")
        should_include = True

        # Check positive filters - if any exist, at least one must match (OR logic)
        if positive_filters:
            has_match = any(f in path for f in positive_filters)
            if not has_match:
                should_include = False

        # Check negative filters - if any match, exclude the request (AND logic with positive)
        if should_include and negative_filters:
            has_negative_match = any(f in path for f in negative_filters)
            if has_negative_match:
                should_include = False

        if should_include:
            filtered_requests.append(req)

    if should_sort:
        filtered_requests.sort(key=str)

    return filtered_requests


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("path_filters", nargs="*", help=f"Path substring filters")
    parser.add_argument("-v", "--verbose", action="store_true", help="Enable diagnostic messages")
    parser.add_argument("--get", action="store_true", help="Include GET requests (excluded by default)")
    parser.add_argument("--keep", action="store_true", help="Keep out.requests.json file after processing")
    parser.add_argument("--sort", action="store_true", help="Sort requests before output")
    parser.add_argument("--oneline", action="store_true", help="Print output with one request per line")
    parser.add_argument("--fname", default="out.requests.txt")
    args = parser.parse_args()

    test_tmp_dir = os.environ.get("TEST_TMP_DIR")
    if test_tmp_dir:
        requests_file = Path(test_tmp_dir) / args.fname
    else:
        requests_file = Path(args.fname)

    if not requests_file.exists():
        sys.exit(f"File {requests_file} not found")

    with open(requests_file) as fobj:
        data = fobj.read()

    if not data:
        return

    requests = read_json_many(data)
    filtered_requests = filter_requests(requests, args.path_filters, args.get, args.sort)
    if args.verbose:
        print(
            f"Read {len(data)} chars, {len(requests)} requests, {len(filtered_requests)} after filtering",
            file=sys.stderr,
            flush=True,
        )

    for req in filtered_requests:
        if args.oneline:
            print(json.dumps(req))
        else:
            print(json.dumps(req, indent=2))

    if not args.keep:
        requests_file.unlink()


if __name__ == "__main__":
    main()
