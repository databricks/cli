#!/usr/bin/env python3
"""
Fuzz test: deploy with direct engine, then plan to check for drift.
Tests that plan after deploy shows no changes (idempotency).

Return codes:
    0: Success
    2: bundle validate failed
    5: bundle deploy (direct) failed - assume wrong config
    10: bundle plan failed - BUG
    11: Drift detected - BUG
    12: panic in validate - BUG
    13: internal error in validate - BUG
    14: panic in deploy - BUG
    15: internal error in deploy - BUG
    16: panic in plan - BUG
    17: internal error in plan - BUG
    18: Timeout (set by fuzzer) - BUG
    19: JSON parse error in plan output - BUG
"""

import json
import os
import re
import subprocess
import sys
from pathlib import Path

PANIC_RE = re.compile(r"panic", re.IGNORECASE)
INTERNAL_ERROR_RE = re.compile(r"internal error", re.IGNORECASE)


def check_for_bugs(stdout, stderr, panic_code, internal_error_code):
    """Check output for panic or internal error. Returns error code or None."""
    combined = stdout + stderr
    if PANIC_RE.search(combined):
        return panic_code
    if INTERNAL_ERROR_RE.search(combined):
        return internal_error_code
    return None


def run_cli(*args, env_extra=None):
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
        if bug_code := check_for_bugs(stdout, stderr, 12, 13):
            print(f"BUG in validate (code={bug_code})\nstdout: {stdout}\nstderr: {stderr}")
            return bug_code
        if code != 0:
            print(f"bundle validate failed (code={code})\nstdout: {stdout}\nstderr: {stderr}")
            return 2

        # Step 2: Deploy with direct engine
        code, stdout, stderr = run_cli("deploy", env_extra={"DATABRICKS_BUNDLE_ENGINE": "direct"})
        if bug_code := check_for_bugs(stdout, stderr, 14, 15):
            print(f"BUG in deploy (code={bug_code})\nstdout: {stdout}\nstderr: {stderr}")
            return bug_code
        if code != 0:
            print(f"bundle deploy (direct) failed (code={code})\nstdout: {stdout}\nstderr: {stderr}")
            return 5
        deployed = True

        # Step 3: Plan with JSON output to check for drift
        code, stdout, stderr = run_cli("plan", "-o", "json", env_extra={"DATABRICKS_BUNDLE_ENGINE": "direct"})
        if bug_code := check_for_bugs(stdout, stderr, 16, 17):
            print(f"BUG in plan (code={bug_code})\nstdout: {stdout}\nstderr: {stderr}")
            return bug_code
        if code != 0:
            print(f"bundle plan failed (code={code}) - BUG\nstdout: {stdout}\nstderr: {stderr}")
            return 10

        # Parse JSON and check all actions are "skip"
        try:
            data = json.loads(stdout)
        except json.JSONDecodeError as e:
            print(f"Failed to parse plan JSON: {e}\nstdout: {stdout}\nstderr: {stderr}")
            return 19

        plan = data.get("plan", {})
        non_skip_actions = []
        for resource_key, resource_plan in plan.items():
            action = resource_plan.get("action")
            if action != "skip":
                non_skip_actions.append((resource_key, resource_plan))

        if non_skip_actions:
            print(f"Drift detected: {len(non_skip_actions)} non-skip actions")
            for resource_key, resource_plan in non_skip_actions:
                action = resource_plan.get("action")
                # Collect field names from changes where action != skip
                changed_fields = []
                for field_name, field_info in resource_plan.get("changes", {}).items():
                    if isinstance(field_info, dict) and field_info.get("action") != "skip":
                        changed_fields.append(field_name)
                fields_str = ", ".join(changed_fields) if changed_fields else "(no changes details)"
                print(f"  {resource_key}: {action} [{fields_str}]")
            print(f"stderr: {stderr}")
            return 11

        print("All checks passed")
        return 0
    finally:
        if deployed:
            destroy_bundle(root_dir)


if __name__ == "__main__":
    sys.exit(main())
