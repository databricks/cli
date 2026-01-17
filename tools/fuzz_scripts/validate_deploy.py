#!/usr/bin/env python3
"""
Fuzz test script: deploy with terraform, then deploy with direct engine.

Return codes:
    0: Success
    2: bundle validate failed
    5: bundle deploy (terraform) failed - assume wrong config
    10: bundle deploy (direct) failed - BUG
    11: bundle plan (after direct deploy) failed - BUG
    12: Idempotency bug (plan after deploy shows drift)
"""

import os
import shutil
import subprocess
import sys
from pathlib import Path


def run_cli(*args, env_extra: dict | None = None) -> tuple[int, str, str]:
    """Run CLI command and return (returncode, stdout, stderr)."""
    cli = os.environ.get("CLI", "databricks")
    cmd = [cli, "bundle"] + list(args)
    print(f"+ {' '.join(cmd)}", flush=True)
    env = os.environ.copy()
    if env_extra:
        env.update(env_extra)
    result = subprocess.run(cmd, capture_output=True, text=True, env=env)
    return result.returncode, result.stdout, result.stderr


PLAN_NO_CHANGES = "0 to add, 0 to change, 0 to delete, 1 unchanged"


def destroy_bundle(engine: str, cwd: Path):
    """Destroy bundle, ignoring errors."""
    original_cwd = os.getcwd()
    try:
        os.chdir(cwd)
        code, stdout, stderr = run_cli("destroy", "--auto-approve", env_extra={"DATABRICKS_BUNDLE_ENGINE": engine})
        if code != 0:
            print(f"bundle destroy ({engine}) failed (code={code})")
            print(f"stdout: {stdout}")
            print(f"stderr: {stderr}")
    finally:
        os.chdir(original_cwd)


def main():
    root_dir = Path.cwd()
    direct_dir = root_dir / "direct_test"
    terraform_deployed = False
    direct_deployed = False

    try:
        # Step 1: Validate
        code, stdout, stderr = run_cli("validate")
        if code != 0:
            print(f"bundle validate failed (code={code})")
            print(f"stdout: {stdout}")
            print(f"stderr: {stderr}")
            return 2

        # Step 2: Deploy with terraform engine
        code, stdout, stderr = run_cli("deploy", env_extra={"DATABRICKS_BUNDLE_ENGINE": "terraform"})
        if code != 0:
            print(f"bundle deploy (terraform) failed (code={code})")
            print(f"stdout: {stdout}")
            print(f"stderr: {stderr}")
            return 5
        terraform_deployed = True

        # Step 3: Create subdirectory and copy databricks.yml
        direct_dir.mkdir(exist_ok=True)
        shutil.copy("databricks.yml", direct_dir / "databricks.yml")
        os.chdir(direct_dir)

        # Step 4: Deploy with direct engine
        code, stdout, stderr = run_cli("deploy", env_extra={"DATABRICKS_BUNDLE_ENGINE": "direct"})
        if code != 0:
            print(f"bundle deploy (direct) failed (code={code}) - BUG")
            print(f"stdout: {stdout}")
            print(f"stderr: {stderr}")
            return 10
        direct_deployed = True

        # Step 5: Plan to check for drift
        code, stdout, stderr = run_cli("plan", env_extra={"DATABRICKS_BUNDLE_ENGINE": "direct"})
        if code != 0:
            print(f"bundle plan (after direct deploy) failed (code={code})")
            print(f"stdout: {stdout}")
            print(f"stderr: {stderr}")
            return 11

        if PLAN_NO_CHANGES not in stdout + stderr:
            print(f"Drift detected: expected '{PLAN_NO_CHANGES}'")
            print(f"stdout: {stdout}")
            print(f"stderr: {stderr}")
            return 12

        print("All checks passed")
        return 0
    finally:
        # Cleanup: destroy both bundles
        if direct_deployed:
            destroy_bundle("direct", direct_dir)
        if terraform_deployed:
            destroy_bundle("terraform", root_dir)


if __name__ == "__main__":
    sys.exit(main())
