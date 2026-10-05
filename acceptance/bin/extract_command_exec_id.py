#!/usr/bin/env python3
"""
Print the cmd-exec-id from the first recorded request that has one in its User-Agent, and update
ACC_REPLS to replace it with [CMD_EXEC_ID_HASH]. The telemetry cmd_exec_id field is then only
replaced if it matches the id the CLI actually sent.

Requires IncludeRequestHeaders = ["User-Agent"].
"""

import json
import os
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from add_repl import add_repl


def extract_cmd_exec_id():
    requests_file = Path(os.environ["OUT_REQUESTS"])

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

    raise SystemExit(f"No command execution ID found in any request in {requests_file}")


if __name__ == "__main__":
    exec_id = extract_cmd_exec_id()
    add_repl(exec_id, "CMD_EXEC_ID_HASH")
    print(exec_id)
