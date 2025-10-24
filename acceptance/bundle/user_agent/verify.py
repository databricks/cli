#!/usr/bin/env python3
"""
Summarize requests logs. For each log write OK if User-Agent has the right engine/ tag and MISS if not.
"""

import os
import sys
import json
import subprocess
from pathlib import Path


TESTROOT = Path(os.environ["TESTROOT"])


def extract_engine(s):
    result = []
    for item in s.split():
        if item.startswith("engine/"):
            result.append(item)
    return ",".join(result)


def check_user_agent(fname):
    if ".terraform." in fname:
        expected = "engine/terraform"
    elif ".direct" in fname:
        expected = "engine/direct-exp"
    else:
        return

    printer = TESTROOT / "bin" / "print_requests.py"

    result = subprocess.run(
        [sys.executable, printer, "--get", "--oneline", "--fname", fname, "--keep"], capture_output=True, text=True
    )

    for line in result.stdout.strip().split("\n"):
        try:
            data = json.loads(line)
        except Exception:
            print(f"Failed to parse: {line!r}")
            raise

        user_agent = data["headers"]["User-Agent"][0]
        path = data["path"]
        engine = extract_engine(user_agent)

        if engine == expected:
            status = "OK  "
        else:
            status = "MISS"

        fname = fname.replace("\\", "/")
        short_fname = fname.removeprefix("simple/out.requests.").removesuffix(".json")
        print(f"{status}\t{short_fname}\t{path}\t{engine or repr(user_agent)}")


def main():
    cwd = Path.cwd()
    for fname in sorted(cwd.rglob("*.json")):
        fname = fname.relative_to(cwd)
        check_user_agent(str(fname))


if __name__ == "__main__":
    main()
