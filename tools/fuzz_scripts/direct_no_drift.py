#!/usr/bin/env python3
"""
Fuzz test: deploy with direct engine, then plan to check for drift.
Tests that plan after deploy shows no changes (idempotency).

Return codes:
    0: Success
    2: bundle validate failed
    5: bundle deploy (direct) failed - assume wrong config
    10: bundle plan failed - BUG
    11: Drift detected (plan shows non-skip actions) - BUG
"""

import json
import os
import subprocess
import sys
from pathlib import Path


def run_cli(*args, env_extra: dict | None = None) -> tuple[int, str, str]:
    """Run CLI command and return (returncode, stdout, stderr)."""
    cli = os.environ.get("CLI", "databricks")
    cmd = [cli, "bundle"] + list(args)
    env_str = ""
    if env_extra:
        env_str = " ".join(f"{k}={v}" for k, v in env_extra.items()) + " "
    print(f"+ {env_str}{' '.join(cmd)}", flush=True)
    env = os.environ.copy()
    if env_extra:
        env.update(env_extra)
    result = subprocess.run(cmd, capture_output=True, text=True, env=env)
    return result.returncode, result.stdout, result.stderr


def reset_testserver():
    """Reset testserver state via API call."""
    cli = os.environ.get("CLI", "databricks")
    subprocess.run(
        [cli, "api", "post", "/testserver-reset-state"],
        capture_output=True,
    )


def destroy_bundle(cwd: Path):
    """Destroy bundle, ignoring errors."""
    original_cwd = os.getcwd()
    try:
        os.chdir(cwd)
        code, stdout, stderr = run_cli("destroy", "--auto-approve", env_extra={"DATABRICKS_BUNDLE_ENGINE": "direct"})
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
        # Reset testserver state before starting
        reset_testserver()

        # Step 1: Validate
        code, stdout, stderr = run_cli("validate")
        if code != 0:
            print(f"bundle validate failed (code={code})")
            print(f"stdout: {stdout}")
            print(f"stderr: {stderr}")
            return 2

        # Step 2: Deploy with direct engine
        code, stdout, stderr = run_cli("deploy", env_extra={"DATABRICKS_BUNDLE_ENGINE": "direct"})
        if code != 0:
            print(f"bundle deploy (direct) failed (code={code})")
            print(f"stdout: {stdout}")
            print(f"stderr: {stderr}")
            return 5
        deployed = True

        # Step 3: Plan with JSON output to check for drift
        code, stdout, stderr = run_cli("plan", "-o", "json", env_extra={"DATABRICKS_BUNDLE_ENGINE": "direct"})
        if code != 0:
            print(f"bundle plan failed (code={code}) - BUG")
            print(f"stdout: {stdout}")
            print(f"stderr: {stderr}")
            return 10

        # Parse JSON and check all actions are "skip"
        try:
            plan = json.loads(stdout)
        except json.JSONDecodeError as e:
            print(f"Failed to parse plan JSON: {e}")
            print(f"stdout: {stdout}")
            print(f"stderr: {stderr}")
            return 10

        non_skip_actions = []
        for item in plan:
            action = item.get("action")
            if action != "skip":
                non_skip_actions.append(item)

        if non_skip_actions:
            print(f"Drift detected: {len(non_skip_actions)} non-skip actions")
            for item in non_skip_actions:
                print(f"  - {item.get('resource_type')}: {item.get('action')}")
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
