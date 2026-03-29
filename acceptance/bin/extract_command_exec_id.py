#!/usr/bin/env python3
import json
import re
import sys
from pathlib import Path


def extract_cmd_exec_id():
    requests_file = Path("out.requests.txt")

    # Read JSON objects one at a time and find the first one with a cmd-exec-id
    # in the User-Agent header. Some requests (e.g. .well-known/databricks-config)
    # are made before the command execution context is set and lack cmd-exec-id.
    with requests_file.open("r") as f:
        json_str = ""
        while True:
            line = f.readline()
            if not line:
                break

            json_str += line
            try:
                data = json.loads(json_str)
            except json.JSONDecodeError:
                continue

            # Reset for next JSON object
            json_str = ""

            user_agent = data.get("headers", {}).get("User-Agent", [""])[0]
            match = re.search(r"cmd-exec-id/([^\s]+)", user_agent)
            if match:
                return match.group(1)

    raise SystemExit("No command execution ID found in any request in out.requests.txt")


if __name__ == "__main__":
    exec_id = extract_cmd_exec_id()
    print(exec_id)
