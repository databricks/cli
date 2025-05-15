#!/usr/bin/env python

import os
from datetime import datetime
import argparse


def main():
    # Get current timestamp
    now = datetime.now().strftime("%Y-%m-%d %H:%M:%S")

    # Print job information
    print(f"Example Python job started at: {now}")

    # Read a secret from a passed secret scope
    try:
        parser = argparse.ArgumentParser()
        parser.add_argument("-s", "--scope_name", help="Name of the secret scope")
        args = parser.parse_args()
        scope_name = args.scope_name

        secret_value = dbutils.secrets.get(scope=scope_name, key="example-key")
        print(
            f"Successfully retrieved secret. First few characters: {secret_value[:3]}***"
        )
    except Exception as e:
        print(f"Could not access secret: {str(e)}")

    print("Example Python job completed successfully")


if __name__ == "__main__":
    main()
