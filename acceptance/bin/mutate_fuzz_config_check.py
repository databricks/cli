#!/usr/bin/env python3
"""
Contract check for mutate_fuzz_config's minimal YAML loader and mutation engine
(the harness diffs stdout; a non-zero exit marks a violation on stderr):

- The loader round-trips every curated base config: load -> to_yaml -> load is a
  fixed point, so a base template the loader can't represent is caught here rather
  than as a confusing fuzz failure.
- Mutation is deterministic for a fixed seed (reproducible repros).

It also prints a few mutated configs so an accidental change to the algorithm shows up
as an output diff.
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from envsubst import substitute_variables
from fuzz_gen_config import MUTATE_BASES
from mutate_fuzz_config import load_yaml, mutate, to_yaml

CONFIGS = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "bundle", "invariant", "configs")


def render(name):
    with open(os.path.join(CONFIGS, name + ".yml.tmpl")) as f:
        return substitute_variables(f.read())


def main():
    # Fixed so the printed configs are stable regardless of the harness's unique name.
    os.environ["UNIQUE_NAME"] = "check"
    failed = False

    for name in MUTATE_BASES:
        text = render(name)
        parsed = load_yaml(text)
        if not isinstance(parsed, dict) or "resources" not in parsed:
            sys.stderr.write(f"{name}: base did not parse to a config with resources\n")
            failed = True
            continue
        # load -> emit -> load must be a fixed point.
        if load_yaml(to_yaml(parsed)) != parsed:
            sys.stderr.write(f"{name}: loader is not a round-trip fixed point\n")
            failed = True

    # Mutation must be reproducible for a fixed seed.
    for seed in range(5):
        a = to_yaml(mutate(load_yaml(render("volume")), seed))
        b = to_yaml(mutate(load_yaml(render("volume")), seed))
        if a != b:
            sys.stderr.write(f"seed {seed}: mutation is not deterministic\n")
            failed = True

    for seed in range(3):
        sys.stdout.write(f"=== volume seed={seed} ===\n")
        sys.stdout.write(to_yaml(mutate(load_yaml(render("volume")), seed)))

    if failed:
        sys.exit(1)


if __name__ == "__main__":
    main()
