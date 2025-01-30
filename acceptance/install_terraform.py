#!/usr/bin/env python3
"""
Script to set up terraform and databricks terraform provider in a local directory:

- Download terraform.
- Download databricks provider.
- Write a .terraformrc config file that uses this directory.
- The config file contains env vars that need to be set so that databricks CLI uses this terraform and provider.
"""

import os
import platform
import zipfile
import argparse
import json
from pathlib import Path
from urllib.request import urlretrieve

os_name = platform.system().lower()

arch = platform.machine().lower()
arch = {"x86_64": "amd64"}.get(arch, arch)
if os_name == "windows" and arch not in ("386", "amd64"):
    # terraform 1.5.5 only has builds for these two.
    arch = "amd64"

terraform_version = "1.5.5"
terraform_file = f"terraform_{terraform_version}_{os_name}_{arch}.zip"
terraform_url = f"https://releases.hashicorp.com/terraform/{terraform_version}/{terraform_file}"
terraform_binary = "terraform.exe" if os_name == "windows" else "terraform"


def retrieve(url, path):
    if not path.exists():
        print(f"Downloading {url} -> {path}")
        urlretrieve(url, path)


def read_version(path):
    for line in path.open():
        if "ProviderVersion" in line:
            # Expecting 'const ProviderVersion = "1.64.1"'
            items = line.strip().split()
            assert len(items) >= 3, items
            assert items[-3:-1] == ["ProviderVersion", "="], items
            version = items[-1].strip('"')
            assert version, items
            return version
    raise SystemExit(f"Could not find ProviderVersion in {path}")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--targetdir", default="build", type=Path)
    parser.add_argument("--provider-version")
    args = parser.parse_args()
    target = args.targetdir

    if not args.provider_version:
        version_file = Path(__file__).parent.parent / "bundle/internal/tf/codegen/schema/version.go"
        assert version_file.exists(), version_file
        terraform_provider_version = read_version(version_file)
        print(f"Read version {terraform_provider_version} from {version_file}")
    else:
        terraform_provider_version = args.provider_version

    terraform_provider_file = f"terraform-provider-databricks_{terraform_provider_version}_{os_name}_{arch}.zip"
    terraform_provider_url = (
        f"https://github.com/databricks/terraform-provider-databricks/releases/download/v{terraform_provider_version}/{terraform_provider_file}"
    )

    target.mkdir(exist_ok=True, parents=True)

    zip_path = target / terraform_file
    terraform_path = target / terraform_binary
    terraform_provider_path = target / terraform_provider_file

    retrieve(terraform_url, zip_path)
    retrieve(terraform_provider_url, terraform_provider_path)

    if not terraform_path.exists():
        print(f"Extracting {zip_path} -> {terraform_path}")

        with zipfile.ZipFile(zip_path, "r") as zip_ref:
            zip_ref.extractall(target)

        terraform_path.chmod(0o755)

    tfplugins_path = target / "tfplugins"
    provider_dir = Path(tfplugins_path / f"registry.terraform.io/databricks/databricks/{terraform_provider_version}/{os_name}_{arch}")
    if not provider_dir.exists():
        print(f"Extracting {terraform_provider_path} -> {provider_dir}")
        os.makedirs(provider_dir, exist_ok=True)
        with zipfile.ZipFile(terraform_provider_path, "r") as zip_ref:
            zip_ref.extractall(provider_dir)

        files = list(provider_dir.iterdir())
        assert files, provider_dir

        for f in files:
            f.chmod(0o755)

    terraformrc_path = target / ".terraformrc"
    if not terraformrc_path.exists():
        path = json.dumps(str(tfplugins_path.absolute()))
        text = f"""# Set these env variables before running databricks cli:
# export DATABRICKS_TF_CLI_CONFIG_FILE={terraformrc_path.absolute()}
# export DATABRICKS_TF_EXEC_PATH={terraform_path.absolute()}

provider_installation {{
    filesystem_mirror {{
        path = {path}
        include = ["registry.terraform.io/databricks/databricks"]
    }}
}}
"""
        print(f"Writing {terraformrc_path}:\n{text}")
        terraformrc_path.write_text(text)


if __name__ == "__main__":
    main()
