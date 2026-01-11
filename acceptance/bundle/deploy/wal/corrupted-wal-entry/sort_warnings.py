#!/usr/bin/env python3
"""Sort warning blocks in CLI output to make test output deterministic.

Warning blocks look like:
Warning: Single node cluster is not correctly configured
  at resources.jobs.XXX.tasks[0].new_cluster
  in databricks.yml:NN:NN

num_workers should be 0 only for single-node clusters...
  spark_conf:
    ...
  custom_tags:
    ...

This script groups consecutive warning blocks, sorts them by job name, and outputs.
"""

import re
import sys


def main():
    content = sys.stdin.read()
    lines = content.split("\n")

    result = []
    i = 0

    while i < len(lines):
        line = lines[i]

        # Check if this is the start of a warning block
        if line.startswith("Warning:"):
            # Collect all consecutive warning blocks
            warnings = []
            while i < len(lines) and (
                lines[i].startswith("Warning:")
                or (
                    warnings
                    and not lines[i].startswith("Uploading")
                    and not lines[i].startswith("Deploying")
                    and not lines[i].startswith(">>>")
                    and not lines[i].startswith("===")
                )
            ):
                # Collect one complete warning block
                block = []
                if lines[i].startswith("Warning:"):
                    block.append(lines[i])
                    i += 1
                    # Collect until next Warning or end marker
                    while i < len(lines):
                        if lines[i].startswith("Warning:"):
                            break
                        if lines[i].startswith("Uploading") or lines[i].startswith("Deploying"):
                            break
                        if lines[i].startswith(">>>") or lines[i].startswith("==="):
                            break
                        block.append(lines[i])
                        i += 1
                    warnings.append(block)
                else:
                    i += 1

            # Sort warnings by the job name in "at resources.jobs.XXX"
            def get_sort_key(block):
                for line in block:
                    match = re.search(r"at resources\.jobs\.(\w+)", line)
                    if match:
                        return match.group(1)
                return ""

            warnings.sort(key=get_sort_key)

            # Output sorted warnings
            for block in warnings:
                for line in block:
                    result.append(line)
        else:
            result.append(line)
            i += 1

    print("\n".join(result), end="")


if __name__ == "__main__":
    main()
