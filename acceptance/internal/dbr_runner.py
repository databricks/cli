#!/usr/bin/env python3

import sys
import os
import subprocess

def main():
    if len(sys.argv) < 3:
        print("Usage: dbr_runner.py <directory> <program> [args...]", file=sys.stderr)
        sys.exit(1)

    directory = sys.argv[1]
    command = sys.argv[2:]

    # Check if directory exists
    if not os.path.isdir(directory):
        print(f"Error: Directory '{directory}' does not exist", file=sys.stderr)
        sys.exit(1)

    # Change to the specified directory
    os.chdir(directory)

    # Execute the provided command
    try:
        with open("_internal_stdout", "w") as stdout_file, open("_internal_stderr", "w") as stderr_file:
            result = subprocess.run(
                command,
                stdout=stdout_file,
                stderr=stderr_file,
                text=True
            )

        # Exit with the same code as the command
        sys.exit(result.returncode)

    except Exception as e:
        print(f"Error executing command: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
