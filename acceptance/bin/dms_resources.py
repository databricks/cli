#!/usr/bin/env python3
"""
Read resource ids and state from the deployment metadata service.

While a bundle records deployment history the service owns the resource set, so the state file is
not where ids and state come from. The service is asked instead, which takes two lookups: the CLI
keeps no deployment id locally, and the id is the object id of the workspace node the service
registers under <state path>/resources.deployment.json (see libs/dms/resolve.go).
"""

import functools
import json
import os
import subprocess

CLI = os.environ.get("CLI", "databricks")

# Must match dms.DeploymentNodeName.
DEPLOYMENT_NODE_NAME = "resources.deployment.json"


def run_json(cmd):
    """Run cmd and parse its stdout. stderr is captured rather than inherited: these lookups are
    plumbing, and a CLI warning like "no files to sync" would otherwise land in the test output."""
    result = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, encoding="utf-8")
    if result.returncode != 0:
        raise SystemExit(f"{cmd} failed with code {result.returncode}\n{result.stdout}{result.stderr}".strip())
    return json.loads(result.stdout)


def records_deployment_history():
    """Whether this run records deployment history, so the service is what to ask."""
    return os.environ.get("DATABRICKS_BUNDLE_RECORD_DEPLOYMENT_HISTORY") == "true"


@functools.cache
def get_resources(target):
    """Map every recorded resource key ("jobs.foo") to its {"id", "state"}.

    Empty when the bundle has no deployment recorded yet. Cached because a lookup costs three
    round trips and a script asks for one resource at a time.
    """
    args = [CLI, "bundle", "validate", "--output", "json"]
    if target:
        args += ["-t", target]
    state_path = run_json(args)["workspace"]["state_path"]

    node = run_json([CLI, "workspace", "get-status", f"{state_path}/{DEPLOYMENT_NODE_NAME}"])
    deployment_id = node.get("object_id")
    if not deployment_id:
        return {}

    listed = run_json([CLI, "api", "get", f"/api/2.0/bundle/deployments/{deployment_id}/resources"])

    result = {}
    for resource in listed.get("resources") or []:
        # The service stores state as the opaque envelope the CLI wrote (dstate.RecordedState), so
        # unwrap it to the resource state itself.
        envelope = json.loads(resource["state"]) if resource.get("state") else {}
        result[resource["resource_key"]] = {
            "id": resource.get("resource_id"),
            "state": envelope.get("state") or {},
        }
    return result
