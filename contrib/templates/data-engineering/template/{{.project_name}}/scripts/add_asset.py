#!/usr/bin/env python3
#
# add_asset.py is used to initialize a new asset from the data-engineering template.
#
import sys
import subprocess
from typing import Literal

VALID_ASSETS = ["etl-pipeline", "job", "ingest-pipeline"]
AssetType = Literal["etl-pipeline", "job", "ingest-pipeline"]


def init_bundle(asset_type: AssetType) -> None:
    cmd = f"databricks bundle init https://github.com/databricks/bundle-examples --template-dir contrib/templates/data-engineering/assets/{asset_type} " + " ".join(sys.argv[2:])
    subprocess.run(cmd, shell=True)


def show_menu() -> AssetType:
    print("\nSelect asset type to initialize:")
    for i, asset in enumerate(VALID_ASSETS, 1):
        print(f"{i}. {asset}")

    while True:
        try:
            choice = int(input("\nEnter number (1-3): "))
            if 1 <= choice <= len(VALID_ASSETS):
                return VALID_ASSETS[choice - 1]
            print("Invalid choice. Please try again.")
        except ValueError:
            print("Please enter a number.")


def main():
    if len(sys.argv) > 1:
        asset_type = sys.argv[1]
        if asset_type not in VALID_ASSETS:
            print(f"Error: Asset type must be one of {VALID_ASSETS}")
            sys.exit(1)
    else:
        asset_type = show_menu()

    init_bundle(asset_type)


if __name__ == "__main__":
    main()
