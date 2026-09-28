#!/usr/bin/env python3
"""
Seed a terraform-state fixture for the auto-migration acceptance tests.

The terraform deployment engine was removed, so these tests can no longer produce a
terraform state with a real `bundle deploy`. Instead each test checks in a
terraform.tfstate fixture (real terraform state shape) whose resource ids are
placeholder tokens "__ID_<resource-key>__" (e.g. __ID_resources.jobs.test_job__).
This helper simulates "a prior terraform deployment of the same config":

  1. deploys the bundle on the direct engine, which creates every resource on the
     backend exactly as a deploy would (so the migrated state's first plan is a
     no-op, not a spurious update), and records their real ids and resolved state;
  2. substitutes each real id/attribute into its placeholder in the fixture and writes
     the result to .databricks/bundle/<target>/terraform/terraform.tfstate; and
  3. removes the direct state (local + remote) so the bundle looks like it is still
     on the terraform engine.

The next `bundle deploy` then reads the fixture and migrates it to the direct engine.

Fixture placeholders (substituted after the direct deploy):
  __ID_<resource-key>__            the resource's backend id, e.g. __ID_resources.jobs.foo__
  __ATTR{<resource-key>}{<field>}__ a resolved top-level state field (JSON-encoded), for a
                                    computed value a fixture cannot hardcode, e.g. a volume's
                                    __ATTR{resources.volumes.v}{storage_location}__. Write it
                                    WITHOUT surrounding quotes; the JSON value is inserted.

Any deploy args after the known ones (e.g. --var) are passed through to the seeding deploy,
so the created resources match the config the test migrates.

Usage: seed_tfstate.py FIXTURE [-t TARGET] [-- DEPLOY_ARG...]
"""

import argparse
import json
import os
import re
import subprocess
import sys

sys.path.insert(0, os.path.dirname(__file__))
from print_state import get_remote_state_path, get_state_file

CLI = os.environ["CLI"]


def run_quiet(cmd):
    """Run cmd, capturing its output so this setup step does not leak into the test's
    output.txt. Fails loudly (with the captured output) on a non-zero exit."""
    result = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, encoding="utf-8")
    if result.returncode != 0:
        sys.exit(f"seed_tfstate.py: {cmd} failed:\n{result.stdout}")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("fixture")
    parser.add_argument("-t", "--target")
    # Remaining args (e.g. --var "x=y") are forwarded to the seeding deploy verbatim.
    args, deploy_args = parser.parse_known_args()

    # Read the fixture and drop it from the bundle root before deploying, so the deploy
    # does not sync it as a bundle file.
    with open(args.fixture) as f:
        raw = f.read()
    os.remove(args.fixture)

    # Create the resources on the backend, matching config, via a direct deploy.
    deploy = [CLI, "bundle", "deploy"]
    if args.target:
        deploy += ["-t", args.target]
    deploy += deploy_args
    run_quiet(deploy)

    # Substitute each resource's id and requested attributes from the direct state.
    state_file = get_state_file(args.target, backup=False)
    state = json.loads(open(state_file).read())["state"]
    for key, entry in state.items():
        rid = entry.get("__id__")
        if rid is not None:
            raw = raw.replace(f"__ID_{key}__", str(rid))

    def attr(match):
        key, field = match.group(1), match.group(2)
        entry = state.get(key)
        if entry is None:
            sys.exit(f"seed_tfstate.py: __ATTR of unknown resource {key!r}")
        return json.dumps(entry.get("state", {}).get(field))

    raw = re.sub(r"__ATTR\{([^}]+)\}\{([^}]+)\}__", attr, raw)

    if "__ID_" in raw or "__ATTR{" in raw:
        sys.exit(f"seed_tfstate.py: unresolved placeholders remain in {args.fixture}:\n{raw}")

    # Write the terraform state where the migration reads it.
    tf_path = os.path.join(os.path.dirname(state_file), "terraform", "terraform.tfstate")
    os.makedirs(os.path.dirname(tf_path), exist_ok=True)
    with open(tf_path, "w") as f:
        f.write(raw)

    # Remove the direct state, local and remote, so the bundle resolves to the terraform
    # engine again and the next deploy migrates.
    for path in (state_file, state_file + ".wal"):
        try:
            os.remove(path)
        except FileNotFoundError:
            pass
    remote_state = get_remote_state_path(args.target)
    if remote_state:
        run_quiet([CLI, "workspace", "delete", f"{remote_state}/resources.json"])


if __name__ == "__main__":
    main()
