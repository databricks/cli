#!/usr/bin/env python3
"""
Mutate a curated, deploy-verified bundle config for the invariant fuzzer.

Destructive: delete/replace a field (token, dangerous scalar, or empty container).
Additive: inject one optional from INJECT that the base omits (deploy-proven shapes).

Emits JSON on stdout (valid YAML 1.2; no PyYAML in the harness). load_yaml covers the
bases' block style only.
"""

import json
import os
import random
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

# Single-resource invariant configs (data/ staged by script.prepare).
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

# Optional absences keyed by resources.<type>; shapes that deploy or showed drift.
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
        ("git_source", {"branch": "main"}),
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
        ("aliases", [{"alias_name": "champion", "id": "alias-champion"}]),
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

# Audited-empty types (a missing INJECT key would look the same without this).
NO_INJECT = {
    "secret_scopes": "only keyvault_metadata remains: Azure-only, conflicts with backend_type DATABRICKS",
}


def token(rng):
    return "fuzz_" + "".join(rng.choice("abcdefghijklmnopqrstuvwxyz0123456789") for _ in range(8))


def dump_config(config):
    # ensure_ascii=False: default escapes become invalid YAML surrogates before the bundle sees them.
    return json.dumps(config, indent=2, ensure_ascii=False) + "\n"


def tokenize(text):
    # Full-line comments only; curated bases never use trailing "#".
    out = []
    for raw in text.splitlines():
        stripped = raw.lstrip(" ")
        if not stripped or stripped.startswith("#"):
            continue
        out.append((len(raw) - len(stripped), stripped.rstrip()))
    return out


def scalar(text):
    if text in ("", "null", "~"):
        return None
    if text == "[]":
        return []
    if text == "{}":
        return {}
    # Flow style would round-trip as a string; fail loud so a new base is caught.
    if text[0] in "[{":
        sys.exit(f"mutate_fuzz_config: flow-style value is not supported: {text!r}")
    if text == "true":
        return True
    if text == "false":
        return False
    try:
        return int(text)
    except ValueError:
        pass
    try:
        return float(text)
    except ValueError:
        pass
    if len(text) >= 2 and text[0] == text[-1] and text[0] in "\"'":
        return text[1:-1]
    return text


def parse_block(tokens, i, indent):
    if i >= len(tokens):
        return {}, i
    first = tokens[i][1]
    if first.startswith("- ") or first == "-":
        return parse_seq(tokens, i, indent)
    if ": " in first or first.endswith(":"):
        return parse_map(tokens, i, indent)
    return scalar(first), i + 1


def parse_map(tokens, i, indent):
    result = {}
    while i < len(tokens) and tokens[i][0] == indent:
        content = tokens[i][1]
        if content.startswith("- "):
            break
        if ": " in content:
            key, _, rest = content.partition(": ")
            result[key.strip()] = scalar(rest)
            i += 1
        elif content.endswith(":"):
            key = content[:-1].strip()
            i += 1
            if i < len(tokens) and tokens[i][0] > indent:
                value, i = parse_block(tokens, i, tokens[i][0])
            else:
                value = None
            result[key] = value
        else:
            break
    return result, i


def parse_seq(tokens, i, indent):
    result = []
    while i < len(tokens) and tokens[i][0] == indent and (tokens[i][1].startswith("- ") or tokens[i][1] == "-"):
        after = tokens[i][1][2:] if tokens[i][1].startswith("- ") else ""
        child_indent = indent + 2
        item = []
        if after:
            item.append((child_indent, after))
        i += 1
        while i < len(tokens) and tokens[i][0] >= child_indent:
            item.append(tokens[i])
            i += 1
        result.append(parse_block(item, 0, child_indent)[0] if item else None)
    return result, i


def load_yaml(text):
    tokens = tokenize(text)
    value, _ = parse_block(tokens, 0, 0)
    return value


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
        return
    instance, name, value = rng.choice(candidates)
    # Deep copy so later destructive steps do not mutate the shared catalog entry.
    instance[name] = json.loads(json.dumps(value))


def mutate(config, seed):
    rng = random.Random(seed)
    # Resource instances only: keep the bundle/resources skeleton intact.
    roots = [instance for _, instance in resource_instances(config)]

    for _ in range(rng.randint(1, 3)):
        if rng.random() < ADD_PROB:
            add_field(rng, config)
        else:
            mutate_once(rng, roots)

    return config


def main():
    # Windows stdout is often ANSI; UTF-8 probes need an explicit encoding.
    sys.stdout.reconfigure(encoding="utf-8")

    seed = int(os.environ["FUZZ_SEED"])
    name = MUTATE_BASES[seed % len(MUTATE_BASES)]
    path = os.path.join(os.environ["INVARIANT_DIR"], "configs", name + ".yml.tmpl")
    with open(path) as f:
        config = load_yaml(substitute_variables(f.read()))
    sys.stdout.write(dump_config(mutate(config, seed)))


if __name__ == "__main__":
    main()
