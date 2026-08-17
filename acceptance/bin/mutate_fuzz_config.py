#!/usr/bin/env python3
"""
Mutate a curated, deploy-verified bundle config for the invariant fuzzer.

Destructive: delete/replace a field (token, dangerous scalar, or empty container).
Additive: inject one optional from INJECT that the base omits.
Each seed picks exactly one mode so an additive finding maps to one catalog entry.

Bases are deploy-verified YAML templates in bundle/invariant/configs/, parsed via
$YAML2JSON (stdlib Python cannot read YAML). Emits JSON on stdout; the bundle
reads it as YAML 1.2.
"""

import json
import os
import random
import subprocess
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from envsubst import substitute_variables

# Prefer inject: that is how reconcile/drift bugs are reached.
ADD_PROB = 0.6

# Hostile free-form scalars; the CLI must reject or round-trip without panicking.
DANGEROUS_STRINGS = [
    "",
    " ",
    "a" * 300,
    "line1\nline2",
    "tab\there",
    "\U0001f680-unicode-\u00e9",
    "quote\"and'apostrophe",
    "${resources.jobs.does_not_exist.id}",
    "../../etc/passwd",
]
DANGEROUS_INTS = [
    2**31 - 1,
    2**31,
    -(2**31),
    2**63 - 1,
    -(2**63),
    -1,
]
DANGEROUS = DANGEROUS_STRINGS + DANGEROUS_INTS

# Single-resource invariant configs.
MUTATE_BASES = [
    "app",
    "catalog",
    "experiment",
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

CONFIGS_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "bundle", "invariant", "configs")

# Schema-valid optionals from past drift/reconcile findings (may still fail to deploy).
INJECT = {
    "apps": [
        ("description", "fuzz-app-description"),
        (
            "config",
            {
                "command": ["python", "app.py"],
                "env": [{"name": "FUZZ_ENV", "value": "1"}],
            },
        ),
        ("lifecycle", {"started": False}),
    ],
    "catalogs": [
        ("custom_max_retention_hours", 168),
        (
            "managed_encryption_settings",
            {"customer_managed_key_id": "00000000-0000-0000-0000-000000000000"},
        ),
        ("properties", {"fuzz_key": "fuzz_val"}),
    ],
    "experiments": [
        ("artifact_location", "dbfs:/databricks/mlflow-tracking/fuzz"),
        ("tags", [{"key": "fuzz", "value": "1"}]),
    ],
    "external_locations": [
        ("read_only", True),
        ("skip_validation", True),
    ],
    "jobs": [
        ("description", "fuzz-job"),
        ("max_concurrent_runs", 1),
        (
            "webhook_notifications",
            {"on_success": [{"id": "alpha"}, {"id": "beta"}]},
        ),
        ("tags", {"fuzz": "1"}),
    ],
    "models": [
        ("description", "fuzz-model"),
    ],
    "model_serving_endpoints": [
        ("description", "fuzz-endpoint"),
        ("route_optimized", True),
        (
            "config",
            {
                "served_entities": [
                    {
                        "name": "prod",
                        "burst_scaling_enabled": True,
                        "external_model": {
                            "name": "gpt-4o-mini",
                            "provider": "openai",
                            "task": "llm/v1/chat",
                            "openai_config": {
                                "openai_api_key_plaintext": "sk-test-plaintext-key",
                            },
                        },
                    }
                ],
                "traffic_config": {
                    "routes": [{"served_model_name": "prod", "traffic_percentage": 100}],
                },
            },
        ),
    ],
    "pipelines": [
        ("allow_duplicate_names", True),
        ("parameters", {"fuzz_param": "1"}),
        ("development", True),
        ("photon", False),
    ],
    "registered_models": [
        ("comment", "fuzz-registered-model"),
    ],
    "schemas": [
        ("comment", "fuzz-schema"),
        ("properties", {"fuzz_key": "fuzz_val"}),
    ],
    "sql_warehouses": [
        ("enable_photon", True),
        ("lifecycle", {"started": False}),
        ("tags", {"fuzz": "1"}),
    ],
    "volumes": [
        ("comment", "fuzz-volume"),
    ],
}

# Audited-empty types (an absent INJECT key would look identical).
NO_INJECT = {
    "secret_scopes": "only keyvault_metadata remains: Azure-only, conflicts with backend_type DATABRICKS",
}


def token(rng):
    return "fuzz_" + "".join(rng.choice("abcdefghijklmnopqrstuvwxyz0123456789") for _ in range(8))


def dump_config(config):
    # ensure_ascii=False: default escapes become invalid YAML surrogates before the bundle sees them.
    return json.dumps(config, indent=2, ensure_ascii=False) + "\n"


def load_base(name):
    path = os.path.join(CONFIGS_DIR, name + ".yml.tmpl")
    # Same loader as the bundle; substitute after parse so placeholders stay JSON strings.
    result = subprocess.run([os.environ["YAML2JSON"], path], capture_output=True, check=True, text=True)
    return json.loads(substitute_variables(result.stdout))


def collect(node, out):
    if isinstance(node, dict):
        for k, v in node.items():
            out.append((node, k))
            collect(v, out)
    elif isinstance(node, list):
        for idx, v in enumerate(node):
            out.append((node, idx))
            collect(v, out)


def mutate_once(rng, roots):
    refs = []
    for root in roots:
        collect(root, refs)
    if not refs:
        return
    container, key = rng.choice(refs)
    op = rng.choice(["delete", "scalar", "dangerous", "empty"])
    if op == "delete":
        del container[key]
    elif op == "scalar":
        container[key] = token(rng)
    elif op == "dangerous":
        container[key] = rng.choice(DANGEROUS)
    else:
        container[key] = rng.choice([{}, [], None])


def resource_instances(config):
    for rtype, instances in config.get("resources", {}).items():
        if isinstance(instances, dict):
            for instance in instances.values():
                if isinstance(instance, dict):
                    yield rtype, instance


def add_field(rng, config):
    candidates = []
    for rtype, instance in resource_instances(config):
        for name, value in INJECT.get(rtype, []):
            if name not in instance:
                candidates.append((instance, name, value))
    if not candidates:
        return False
    instance, name, value = rng.choice(candidates)
    # Copy: INJECT values are shared across seeds.
    instance[name] = json.loads(json.dumps(value))
    return True


def mutate(config, seed):
    rng = random.Random(seed)
    # Resource instances only: keep the bundle/resources skeleton intact.
    roots = [instance for _, instance in resource_instances(config)]

    # Fall through when add has nothing to inject (e.g. NO_INJECT types).
    if rng.random() < ADD_PROB and add_field(rng, config):
        return config
    for _ in range(rng.randint(1, 3)):
        mutate_once(rng, roots)
    return config


def main():
    # Windows stdout is often ANSI; UTF-8 probes need an explicit encoding.
    sys.stdout.reconfigure(encoding="utf-8")

    seed = int(os.environ["FUZZ_SEED"])
    name = MUTATE_BASES[seed % len(MUTATE_BASES)]
    sys.stdout.write(dump_config(mutate(load_base(name), seed)))


if __name__ == "__main__":
    main()
