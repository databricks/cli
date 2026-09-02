#!/usr/bin/env python3
"""
Read resource ids and state from the deployment metadata service.

While a bundle records deployment history the service owns the resource set, so the state file is
not where ids and state come from. The service is asked instead, which takes two lookups: the CLI
keeps no deployment id locally, and the id is the object id of the workspace node the service
registers under <state path>/resources.deployment.json (see libs/dms/resolve.go).
"""

import functools
import glob
import json
import os
import posixpath
import subprocess
import sys

sys.path.insert(0, os.path.dirname(__file__))
from print_state import get_state_file

CLI = os.environ.get("CLI", "databricks")

# Must match dms.DeploymentNodeName.
DEPLOYMENT_NODE_NAME = "resources.deployment.json"


def run_json(cmd, allow_failure=False):
    """Run cmd and parse its stdout, or return None if it fails and allow_failure is set. stderr is
    captured rather than inherited: these lookups are plumbing, and a CLI warning like "no files to
    sync" would otherwise land in the test output."""
    result = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, encoding="utf-8")
    if result.returncode != 0:
        if allow_failure:
            return None
        raise SystemExit(f"{cmd} failed with code {result.returncode}\n{result.stdout}{result.stderr}".strip())
    return json.loads(result.stdout)


def records_deployment_history():
    """Whether this run records deployment history, so the service is what to ask."""
    return os.environ.get("DATABRICKS_BUNDLE_RECORD_DEPLOYMENT_HISTORY") == "true"


def get_remote_state_path(target):
    """The bundle's remote state directory.

    Preferred source is the sync snapshot, because it needs no CLI call: re-running the config
    load would need whatever --var and flags the test deployed with, which a helper cannot know.
    A bundle with no files to sync writes no snapshot, so fall back to asking the CLI - those
    bundles are the ones with nothing to parameterize."""
    target_dir = os.path.dirname(get_state_file(target, False))
    snapshots = glob.glob(f"{target_dir}/sync-snapshots/*.json")
    if snapshots:
        remote_path = json.loads(open(snapshots[0]).read())["remote_path"]
        # state and files are siblings under the bundle root.
        return posixpath.join(posixpath.dirname(remote_path), "state")

    args = [CLI, "bundle", "validate", "--output", "json"]
    if target:
        args += ["-t", target]
    return run_json(args)["workspace"]["state_path"]


@functools.cache
def get_resources(target):
    """Map every recorded resource key ("jobs.foo") to its {"id", "state"}.

    Empty when the bundle has no deployment recorded yet. Cached because a lookup costs three
    round trips and a script asks for one resource at a time.
    """
    state_path = get_remote_state_path(target)
    if not state_path:
        return {}

    # No node means nothing has been recorded, the conclusion dms.resolveDeploymentID also draws
    # from a 404 - the deployment is gone once the bundle is destroyed.
    node = run_json([CLI, "workspace", "get-status", f"{state_path}/{DEPLOYMENT_NODE_NAME}"], allow_failure=True)
    if not node or not node.get("object_id"):
        return {}
    deployment_id = node["object_id"]

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
