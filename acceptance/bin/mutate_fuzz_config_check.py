#!/usr/bin/env python3
"""
Contract checks for mutate_fuzz_config. Failures go to stderr; stdout is a few mutated configs
so an algorithm change shows up as an acceptance output diff.

- load -> dump -> load is a fixed point for every MUTATE_BASES entry
- mutate(seed) is deterministic
- sample volume seeds stay pairwise distinct (so output.txt catches algorithm drift)
- INJECT eventually lands on a sparse base (registered_model)
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from envsubst import substitute_variables
from mutate_fuzz_config import INJECT, MUTATE_BASES, dump_yaml, load_yaml, mutate

CONFIGS = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "bundle", "invariant", "configs")


def render(name):
    with open(os.path.join(CONFIGS, name + ".yml.tmpl")) as f:
        return substitute_variables(f.read())


def load(name):
    return load_yaml(render(name))


def instance(config):
    (instances,) = config["resources"].values()
    (value,) = instances.values()
    return value


def resource_type(config):
    (rtype,) = config["resources"]
    return rtype


def main():
    # Stable printed configs regardless of the harness UNIQUE_NAME.
    os.environ["UNIQUE_NAME"] = "check"
    os.environ.setdefault("CURRENT_USER_NAME", "check-user")
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
        rtype = resource_type(parsed)
        if rtype not in INJECT:
            sys.stderr.write(f"{name}: resources.{rtype} has no INJECT entry\n")
            failed = True

    for seed in range(5):
        a = dump_yaml(mutate(load("volume"), seed))
        b = dump_yaml(mutate(load("volume"), seed))
        if a != b:
            sys.stderr.write(f"seed {seed}: mutation is not deterministic\n")
            failed = True

    # Distinct dumps: consecutive seeds can collide and hide algorithm changes in output.txt.
    samples = [0, 1, 5]
    dumps = []
    for seed in samples:
        out = dump_yaml(mutate(load("volume"), seed))
        dumps.append(out)
        sys.stdout.write(f"=== volume seed={seed} ===\n")
        sys.stdout.write(out)
    if len(set(dumps)) != len(dumps):
        sys.stderr.write(f"sample seeds {samples} are not pairwise distinct\n")
        failed = True

    # Sparse base: any new field must come from INJECT.
    base_fields = set(instance(load("registered_model")))
    inject_names = {name for name, _ in INJECT["registered_models"]}
    injected = False
    for seed in range(30):
        fields = set(instance(mutate(load("registered_model"), seed)))
        added = fields - base_fields
        if added:
            injected = True
        unexpected = added - inject_names
        if unexpected:
            sys.stderr.write(f"seed {seed}: injected non-catalog field(s): {sorted(unexpected)}\n")
            failed = True
    if not injected:
        sys.stderr.write("mutation never injected a curated optional field\n")
        failed = True

    if failed:
        sys.exit(1)


if __name__ == "__main__":
    main()
