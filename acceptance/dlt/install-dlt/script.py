#!/usr/bin/env python3
import os
import shutil
import subprocess
import sys


# Helper to format the command for printing
def format_cmd(cmd):
    return " ".join(cmd)


def trace(*args):
    print(f"\n>>> {' '.join(args)}", file=sys.stderr)
    return run_command(args)


def run_command(cmd):
    try:
        result = subprocess.run(cmd, check=True)
        return result.returncode
    except subprocess.CalledProcessError as e:
        print(f"\nExit code: {e.returncode}", file=sys.stderr)
        return e.returncode


def main():
    CLI = os.environ["CLI"]

    # 0. Basic install
    trace(CLI, "install-dlt")

    # 1. dlt file exists, should not overwrite
    os.makedirs("tmpdir", exist_ok=True)
    open(os.path.join("tmpdir", "dlt"), "a").close()
    trace(CLI, "install-dlt", "-d", "tmpdir")
    shutil.rmtree("tmpdir", ignore_errors=True)

    # 2. no permissions to create symlink
    os.makedirs("nopermdir", exist_ok=True)
    try:
        os.chmod("nopermdir", 0o500)
    except Exception:
        pass  # May not work on Windows, ignore
    trace(CLI, "install-dlt", "-d", "nopermdir")
    try:
        os.chmod("nopermdir", 0o700)
    except Exception:
        pass
    shutil.rmtree("nopermdir", ignore_errors=True)

    # 3. program not called as databricks (simulate by copying binary)
    cli_path = shutil.which(CLI)
    if cli_path:
        shutil.copy(cli_path, "notdatabricks")
        try:
            trace(os.path.abspath("notdatabricks"), "install-dlt")
        except Exception:
            pass
        # Cleanup
        os.remove("notdatabricks")
        os.remove("dlt")


if __name__ == "__main__":
    main()
