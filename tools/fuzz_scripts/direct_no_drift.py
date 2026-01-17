#!/usr/bin/env python3
"""
Fuzz test: deploy with direct engine, then plan to check for drift.
Tests that plan after deploy shows no changes (idempotency).

Return codes:
    0: Success
    2: bundle validate failed
    5: bundle deploy (direct) failed - assume wrong config
    10: bundle plan failed - BUG
    11: Drift detected (plan shows changes) - BUG

Environment variables (optional, for testserver mode):
    TESTSERVER_DIRECT_URL: URL for direct engine testserver
"""

import os
import subprocess
import sys
from pathlib import Path

# Env vars to hide from trace output
HIDDEN_ENV_VARS = {"DATABRICKS_HOST", "DATABRICKS_TOKEN"}

PLAN_NO_CHANGES = "0 to add, 0 to change, 0 to delete, 1 unchanged"


def get_env_for_direct() -> dict:
    """Get environment variables for direct engine."""
    env = {"DATABRICKS_BUNDLE_ENGINE": "direct"}

    url = os.environ.get("TESTSERVER_DIRECT_URL")
    if url:
        env["DATABRICKS_HOST"] = url
        env["DATABRICKS_TOKEN"] = "test-token"

    return env


def run_cli(*args, env_extra: dict | None = None) -> tuple[int, str, str]:
    """Run CLI command and return (returncode, stdout, stderr)."""
    cli = os.environ.get("CLI", "databricks")
    cmd = [cli, "bundle"] + list(args)
    env_str = ""
    if env_extra:
        visible = {k: v for k, v in env_extra.items() if k not in HIDDEN_ENV_VARS}
        if visible:
            env_str = " ".join(f"{k}={v}" for k, v in visible.items()) + " "
    print(f"+ {env_str}{' '.join(cmd)}", flush=True)
    env = os.environ.copy()
    if env_extra:
        env.update(env_extra)
    result = subprocess.run(cmd, capture_output=True, text=True, env=env)
    return result.returncode, result.stdout, result.stderr


def destroy_bundle(cwd: Path):
    """Destroy bundle, ignoring errors."""
    original_cwd = os.getcwd()
    try:
        os.chdir(cwd)
        code, stdout, stderr = run_cli("destroy", "--auto-approve", env_extra=get_env_for_direct())
        if code != 0:
            print(f"bundle destroy failed (code={code})")
            print(f"stdout: {stdout}")
            print(f"stderr: {stderr}")
    finally:
        os.chdir(original_cwd)


def main():
    root_dir = Path.cwd()
    deployed = False

    try:
        # Step 1: Validate
        code, stdout, stderr = run_cli("validate")
        if code != 0:
            print(f"bundle validate failed (code={code})")
            print(f"stdout: {stdout}")
            print(f"stderr: {stderr}")
            return 2

        # Step 2: Deploy with direct engine
        code, stdout, stderr = run_cli("deploy", env_extra=get_env_for_direct())
        if code != 0:
            print(f"bundle deploy (direct) failed (code={code})")
            print(f"stdout: {stdout}")
            print(f"stderr: {stderr}")
            return 5
        deployed = True

        # Step 3: Plan to check for drift
        code, stdout, stderr = run_cli("plan", env_extra=get_env_for_direct())
        if code != 0:
            print(f"bundle plan failed (code={code}) - BUG")
            print(f"stdout: {stdout}")
            print(f"stderr: {stderr}")
            return 10

        if PLAN_NO_CHANGES not in stdout + stderr:
            print(f"Drift detected: expected '{PLAN_NO_CHANGES}'")
            print(f"stdout: {stdout}")
            print(f"stderr: {stderr}")
            return 11

        print("All checks passed")
        return 0
    finally:
        if deployed:
            destroy_bundle(root_dir)


if __name__ == "__main__":
    sys.exit(main())
