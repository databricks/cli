#!/usr/bin/env python3
"""
This script implements functionality equivalent to envsubst. We use a python
script instead because that way we do not need to install it when we run integration
tests on DBR.

Usage: envsubst.py

Substitutes environment variables in shell format strings.
Reads from stdin and writes to stdout.

Examples:
  echo 'Hello $USER' | python envsubst.py
  echo 'Hello $USER from $HOME' | python envsubst.py
"""

import sys
import os
import re


def substitute_variables(text):
    """
    Substitute environment variables in text.

    Args:
        text: Input text containing variable references

    Returns:
        Text with variables substituted
    """

    def replace_var(match):
        var_name = match.group(1) or match.group(2)
        return os.environ.get(var_name, "")

    # Match both $VAR and ${VAR} formats
    pattern = r"\$\{([A-Za-z_][A-Za-z0-9_]*)\}|\$([A-Za-z_][A-Za-z0-9_]*)"
    return re.sub(pattern, replace_var, text)


def main():
    # Read from stdin
    input_text = sys.stdin.read()

    # Substitute variables
    output_text = substitute_variables(input_text)

    # Write to stdout
    sys.stdout.write(output_text)


if __name__ == "__main__":
    main()
