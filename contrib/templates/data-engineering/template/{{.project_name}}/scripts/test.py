#!/usr/bin/env python3
#
# test.py runs the unit tests for this project using pytest and serverless compute.
# To use a different form of compute, instead use 'uv run pytest' or
# use your IDE's testing panel. When using VS Code, consider using the Databricks extension.
#
import os
import subprocess


def main():
    os.environ["DATABRICKS_SERVERLESS_COMPUTE_ID"] = "auto"
    subprocess.run(["pytest"], check=True)


if __name__ == "__main__":
    main()
