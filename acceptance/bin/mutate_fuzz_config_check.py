#!/usr/bin/env python3
"""
Contract check for mutate_fuzz_config (the harness diffs stdout; a non-zero exit marks a
violation on stderr):

- The loader round-trips every curated base: load -> dump -> load is a fixed point.
- Mutation is deterministic for a fixed seed (reproducible repros).

It also prints a few mutated configs so an algorithm change shows up as an output diff.
"""

import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from envsubst import substitute_variables
from gen_fuzz_config import SKIP_PROPERTY_NAMES
from mutate_fuzz_config import MUTATE_BASES, dump_yaml, load_yaml, mutate

CONFIGS = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "bundle", "invariant", "configs")
SCHEMA = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", "bundle", "schema", "jsonschema.json")


def render(name):
    with open(os.path.join(CONFIGS, name + ".yml.tmpl")) as f:
        return substitute_variables(f.read())


def load(name):
    return load_yaml(render(name))


def instance(config):
    # The curated bases are single-resource; return that one resource instance.
    (instances,) = config["resources"].values()
    (value,) = instances.values()
    return value


def main():
    # Fixed so the printed configs are stable regardless of the harness's unique name.
    os.environ["UNIQUE_NAME"] = "check"
    failed = False

    for name in MUTATE_BASES:
        parsed = load(name)
        if not isinstance(parsed, dict) or "resources" not in parsed:
            sys.stderr.write(f"{name}: base did not parse to a config with resources\n")
            failed = True
            continue
        if load_yaml(dump_yaml(parsed)) != parsed:
            sys.stderr.write(f"{name}: loader is not a round-trip fixed point\n")
            failed = True

    for seed in range(5):
        a = dump_yaml(mutate(load("volume"), seed))
        b = dump_yaml(mutate(load("volume"), seed))
        if a != b:
            sys.stderr.write(f"seed {seed}: mutation is not deterministic\n")
            failed = True

    for seed in range(3):
        sys.stdout.write(f"=== volume seed={seed} ===\n")
        sys.stdout.write(dump_yaml(mutate(load("volume"), seed)))

    # Assert-only (no stdout) so printed output stays stable as the schema grows. The
    # registered_model base sets no optional fields, so any added field must have been injected.
    with open(SCHEMA) as f:
        schema = json.load(f)

    for seed in range(5):
        a = dump_yaml(mutate(load("registered_model"), seed, schema=schema, unique="check"))
        b = dump_yaml(mutate(load("registered_model"), seed, schema=schema, unique="check"))
        if a != b:
            sys.stderr.write(f"seed {seed}: schema-aware mutation is not deterministic\n")
            failed = True

    base_fields = set(instance(load("registered_model")))
    injected = False
    for seed in range(30):
        fields = set(instance(mutate(load("registered_model"), seed, schema=schema, unique="check")))
        added = fields - base_fields
        if added:
            injected = True
        # Injecting an output-only field would manufacture false drift.
        leaked = SKIP_PROPERTY_NAMES & added
        if leaked:
            sys.stderr.write(f"seed {seed}: injected output-only field(s): {sorted(leaked)}\n")
            failed = True
    if not injected:
        sys.stderr.write("schema-aware mutation never injected an optional field\n")
        failed = True

    if failed:
        sys.exit(1)


if __name__ == "__main__":
    main()
