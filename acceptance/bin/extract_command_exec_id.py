#!/usr/bin/env python3
import json
import re
import sys
from pathlib import Path


def extract_cmd_exec_id():
    requests_file = Path("out.requests.txt")

    # Read until we find a complete JSON object. This is required because we pretty
    # print the JSON object (with new lines) in the out.requests.txt file.
    with requests_file.open("r") as f:
        json_str = ""
        while True:
            line = f.readline()
            if not line:
                raise SystemExit("Requests file is empty")

            json_str += line
            try:
                # Try to parse the accumulated string as JSON
                data = json.loads(json_str)
                break
            except json.JSONDecodeError:
                # If incomplete, continue reading
                continue

        user_agent = data["headers"]["User-Agent"][0]

        if not user_agent:
            raise SystemExit("User-Agent header is empty")

        match = re.search(r"cmd-exec-id/([^\s]+)", user_agent)
        if match:
            return match.group(1)

        raise SystemExit(f"No command execution ID found in User-Agent: {user_agent}")


if __name__ == "__main__":
    exec_id = extract_cmd_exec_id()
    print(exec_id)
