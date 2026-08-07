#!/usr/bin/env python3
"""
Emit a fuzz databricks.yml on stdout for the current seed by perturbing a curated invariant
config (mutate_fuzz_config.py). The schema is used only to inject valid optional fields the
base omits.

Reads FUZZ_SEED, FUZZ_SCHEMA, UNIQUE_NAME and INVARIANT_DIR, which fuzz/script and the
invariant prologue set.
"""

import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from envsubst import substitute_variables
from gen_fuzz_config import to_yaml
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


def main():
    # to_yaml emits non-ASCII literally, so this redirect must be UTF-8: on Windows it would
    # default to the ANSI code page and the astral-plane probe would raise UnicodeEncodeError.
    sys.stdout.reconfigure(encoding="utf-8")

    seed = int(os.environ["FUZZ_SEED"])
    name = MUTATE_BASES[seed % len(MUTATE_BASES)]
    path = os.path.join(os.environ["INVARIANT_DIR"], "configs", name + ".yml.tmpl")
    unique = os.environ["UNIQUE_NAME"]
    with open(path) as f:
        rendered = substitute_variables(f.read())
    config = load_yaml(rendered)
    with open(os.environ["FUZZ_SCHEMA"]) as f:
        schema = json.load(f)
    sys.stdout.write(to_yaml(mutate(config, seed, schema=schema, unique=unique)))


if __name__ == "__main__":
    main()
