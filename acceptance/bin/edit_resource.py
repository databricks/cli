#!/usr/bin/env python3
import sys
import os
import subprocess
import argparse
import json
import pprint

sys.path.insert(0, os.path.dirname(__file__))
import util
from util import run_json, run


CLI = os.environ["CLI"]


class jobs:

    def get(self, job_id):
        return run_json([CLI, "jobs", "get", job_id])["settings"]

    def set(self, job_id, value):
        payload = {"job_id": job_id, "new_settings": value}
        return run([CLI, "jobs", "reset", job_id, "--json", json.dumps(payload)])


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("type")
    parser.add_argument("id")
    parser.add_argument("-v", "--verbose", action="store_true")
    args = parser.parse_args()

    util.VERBOSE = args.verbose

    script = sys.stdin.read()

    klass = globals()[args.type]
    instance = klass()

    data = instance.get(args.id)
    my_locals = {"r": data}

    try:
        exec(script, locals=my_locals)
    except Exception:
        pprint.pprint(my_locals)
        raise

    instance.set(args.id, my_locals["r"])


if __name__ == "__main__":
    main()
