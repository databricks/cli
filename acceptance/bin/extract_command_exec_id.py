#!/usr/bin/env python3
import json
import re
import sys
from pathlib import Path


def extract_cmd_exec_id():
    requests_file = Path("out.requests.txt")

    if not requests_file.exists():
        return f"Requests file '{requests_file}' not found"

    try:
        # Read until we find a complete JSON object
        with requests_file.open("r") as f:
            json_str = ""
            while True:
                line = f.readline()
                if not line:
                    return "Requests file is empty"

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
                return "User-Agent header is empty"

            match = re.search(r"cmd-exec-id/([^\s]+)", user_agent)
            if match:
                print(match.group(1))
                return None

            return f"No command execution ID found in User-Agent: {user_agent}"

    except KeyError as e:
        return f"Missing required field in JSON: {e}"
    except IndexError:
        return "No User-Agent headers found in requests"


if __name__ == "__main__":
    error = extract_cmd_exec_id()
    if error:
        print(f"Error: {error}", file=sys.stderr)
        sys.exit(1)
