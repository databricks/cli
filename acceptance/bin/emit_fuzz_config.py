#!/usr/bin/env python3
"""
Emit a fuzz databricks.yml on stdout for the current seed, picking the strategy from FUZZ_MODE so
the invariant scripts don't each duplicate the branch:

  generate - build from scratch by walking `bundle schema` (gen_fuzz_config.py).
  mutate   - perturb a curated invariant config (mutate_fuzz_config.py).

Reads FUZZ_SEED, FUZZ_SCHEMA, FUZZ_MODE, UNIQUE_NAME and INVARIANT_DIR, which fuzz/script and the
invariant prologue set.
"""

import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from envsubst import substitute_variables
from gen_fuzz_config import gen_config, to_yaml
from mutate_fuzz_config import load_yaml, mutate

# Curated single-resource configs that deploy standalone (only $UNIQUE_NAME, no init script). All
# are in the invariant INPUT_CONFIG matrix, so they stay deploy-verified.
MUTATE_BASES = [
    "catalog",
    "external_location",
    "job",
    "model",
    "model_serving_endpoint",
    "pipeline",
    "registered_model",
    "schema",
    "secret_scope",
    "sql_warehouse",
    "volume",
]


def generate(seed):
    with open(os.environ["FUZZ_SCHEMA"]) as f:
        schema = json.load(f)
    return to_yaml(gen_config(schema, seed, os.environ["UNIQUE_NAME"]))


def mutate_base(seed):
    name = MUTATE_BASES[seed % len(MUTATE_BASES)]
    path = os.path.join(os.environ["INVARIANT_DIR"], "configs", name + ".yml.tmpl")
    unique = os.environ["UNIQUE_NAME"]
    with open(path) as f:
        rendered = substitute_variables(f.read())
    config = load_yaml(rendered)
    # The schema lets mutate inject valid optional fields, not just perturb existing ones.
    with open(os.environ["FUZZ_SCHEMA"]) as f:
        schema = json.load(f)
    return to_yaml(mutate(config, seed, schema=schema, unique=unique))


def main():
    seed = int(os.environ["FUZZ_SEED"])
    mode = os.environ["FUZZ_MODE"]
    if mode == "generate":
        sys.stdout.write(generate(seed))
    elif mode == "mutate":
        sys.stdout.write(mutate_base(seed))
    else:
        sys.exit(f"emit_fuzz_config: unknown FUZZ_MODE {mode!r}")


if __name__ == "__main__":
    main()
