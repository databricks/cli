#!/usr/bin/env python3
"""
Mutate a known-good bundle config by deleting, perturbing, and adding curated fields.

Perturbs a curated invariant config that already deploys.

Two mutation kinds, chosen per step:

- destructive (always): delete a field or replace it with a token, a dangerous value, or an empty
  container.
- additive: inject one optional field from INJECT that the base omits. Values are hand-curated to
  deploy (and to cover fields that previously reached reconcile/drift bugs).

As a script, emits one mutated databricks.yml on stdout for the current seed (see main).

YAML I/O is stdlib-only (acceptance python has no PyYAML): dump uses JSON scalars so dangerous
probes stay one line; load understands that dialect plus the curated bases' block style.
"""

import json
import os
import random
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from envsubst import substitute_variables

# Biased high: injection is the path to drift bugs.
ADD_PROB = 0.6

# Probes for free-form scalars: the CLI must reject or round-trip these without panicking.
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

# Curated single-resource configs that deploy standalone (no init script). All are in the invariant
# INPUT_CONFIG matrix, so they stay deploy-verified. Fixture files (data/app, data/pipeline.py)
# are staged by fuzz/script.prepare.
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

# Optional fields absent from the corresponding base, keyed by resources.<type>. Only injected when
# missing. Shapes are taken from acceptance fixtures that already deploy / reproduce known drift.
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
        ("description", "fuzz-experiment"),
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
    "secret_scopes": [],
    "sql_warehouses": [
        ("enable_photon", True),
        ("lifecycle", {"started": False}),
        ("tags", {"fuzz": "1"}),
    ],
    "volumes": [
        ("comment", "fuzz-volume"),
    ],
}


def token(rng):
    return "fuzz_" + "".join(rng.choice("abcdefghijklmnopqrstuvwxyz0123456789") for _ in range(8))


def dump_scalar(v):
    # ensure_ascii=False keeps non-ASCII literal: the default escapes astral chars into surrogate
    # pairs that YAML rejects, killing the config before it reaches bundle logic. Control chars
    # stay escaped by json.dumps, which YAML accepts.
    return json.dumps(v, ensure_ascii=False)


def dump_yaml(obj, indent=0, list_item=False):
    pad = "  " * indent
    if isinstance(obj, dict):
        if not obj:
            return f"{pad}{{}}\n" if not list_item else f"{pad}- {{}}\n"
        out = ""
        first = True
        for k, v in obj.items():
            prefix = pad + "- " if list_item and first else (pad + "  " if list_item else pad)
            child_indent = indent + 2 if list_item else indent + 1
            if isinstance(v, (dict, list)) and v:
                out += f"{prefix}{k}:\n" + dump_yaml(v, child_indent)
            else:
                out += f"{prefix}{k}: {dump_scalar(v)}\n"
            first = False
        return out
    if isinstance(obj, list):
        if not obj:
            return f"{pad}- []\n" if list_item else f"{pad}[]\n"
        # A list inside a list: the marker needs its own line, else the two flatten into one.
        if list_item:
            return f"{pad}-\n" + dump_yaml(obj, indent + 1)
        out = ""
        for item in obj:
            if isinstance(item, (dict, list)):
                out += dump_yaml(item, indent, list_item=True)
            else:
                out += f"{pad}- {dump_scalar(item)}\n"
        return out
    return f"{pad}{dump_scalar(obj)}\n"


def tokenize(text):
    # (indent, content) per line. Only full-line comments: no curated base has a trailing "#".
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
    # dump_yaml emits empty containers in flow form; read them back so load -> dump -> load holds.
    if text == "[]":
        return []
    if text == "{}":
        return {}
    # The one shape this loader cannot represent: "[id]" reads back as the string "[id]", turning a
    # list into a scalar, and load -> dump -> load stays a fixed point. Exit so a new MUTATE_BASES
    # entry fails the selftest.
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
    # (container, key) per child, so a mutation can delete or replace it in place.
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
    """Yield (resource_type, instance_dict) for each resource in the config."""
    for rtype, instances in config.get("resources", {}).items():
        if isinstance(instances, dict):
            for instance in instances.values():
                if isinstance(instance, dict):
                    yield rtype, instance


def add_field(rng, config):
    # Inject one curated optional that the instance still lacks.
    candidates = []
    for rtype, instance in resource_instances(config):
        for name, value in INJECT.get(rtype, []):
            if name not in instance:
                candidates.append((instance, name, value))
    if not candidates:
        return
    instance, name, value = rng.choice(candidates)
    # Copy so later destructive steps cannot mutate the shared catalog value in place.
    instance[name] = json.loads(json.dumps(value))


def mutate(config, seed):
    rng = random.Random(seed)

    # Only inside resource instances, so the bundle/resources skeleton survives.
    roots = [instance for _, instance in resource_instances(config)]

    for _ in range(rng.randint(1, 3)):
        if rng.random() < ADD_PROB:
            add_field(rng, config)
        else:
            mutate_once(rng, roots)

    return config


def main():
    # dump_yaml emits non-ASCII literally, so this redirect must be UTF-8: on Windows it would
    # default to the ANSI code page and the astral-plane probe would raise UnicodeEncodeError.
    sys.stdout.reconfigure(encoding="utf-8")

    seed = int(os.environ["FUZZ_SEED"])
    name = MUTATE_BASES[seed % len(MUTATE_BASES)]
    path = os.path.join(os.environ["INVARIANT_DIR"], "configs", name + ".yml.tmpl")
    with open(path) as f:
        config = load_yaml(substitute_variables(f.read()))
    sys.stdout.write(dump_yaml(mutate(config, seed)))


if __name__ == "__main__":
    main()
