#!/usr/bin/env python3
"""
Mutate a known-good bundle config by deleting, perturbing, and adding random fields.

Perturbs a curated invariant config that already deploys.

Two mutation kinds, chosen per step:

- destructive (always): delete a field or replace it with a token, a dangerous value, or an empty
  container.
- additive (with a schema): inject a valid optional field the base omits, valued by the schema
  generator in gen_fuzz_config.py.

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
from gen_fuzz_config import (
    DANGEROUS_INTS,
    DANGEROUS_STRINGS,
    Generator,
    is_empty,
    resource_element,
    resource_types,
    token,
)

DANGEROUS = DANGEROUS_STRINGS + DANGEROUS_INTS

# Biased high: injection is the path to drift bugs.
ADD_PROB = 0.6

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


def collect_insertions(gen, node, schema, out):
    # Every writable optional field absent from the node, walking node and schema together so
    # nested objects are candidates too. Each point carries gen.rtype, which add_field restores
    # before generating a value for the one it picks.
    schema = gen.resolve(schema)
    if not isinstance(schema, dict):
        return

    branches = schema.get("oneOf") or schema.get("anyOf")
    if branches:
        # Pick the branch matching the node we have.
        picked = None
        for branch in branches:
            resolved = gen.resolve(branch)
            if isinstance(node, dict) and (
                resolved.get("type") == "object" or "properties" in resolved or gen.is_map(resolved)
            ):
                picked = resolved
                break
            if isinstance(node, list) and resolved.get("type") == "array":
                picked = resolved
                break
        if picked is None:
            return
        schema = picked

    if isinstance(node, dict):
        props = schema.get("properties", {})
        for name, prop_schema in props.items():
            if name not in node and not gen.should_skip_property(name):
                out.append((node, name, prop_schema, gen.rtype))
        for key, value in node.items():
            if key in props and isinstance(value, (dict, list)):
                collect_insertions(gen, value, props[key], out)
        if gen.is_map(schema):
            for value in node.values():
                if isinstance(value, (dict, list)):
                    collect_insertions(gen, value, schema["additionalProperties"], out)
    elif isinstance(node, list):
        items = schema.get("items")
        if items:
            for value in node:
                if isinstance(value, (dict, list)):
                    collect_insertions(gen, value, items, out)


def add_field(gen, rng, config):
    # Inject one valid optional field, absent from the base, into a random insertion point.
    types = resource_types(gen)
    points = []
    for rtype, instances in config.get("resources", {}).items():
        if rtype not in types or not isinstance(instances, dict):
            continue
        element = resource_element(gen, types[rtype])
        gen.rtype = rtype
        for instance in instances.values():
            if isinstance(instance, dict):
                collect_insertions(gen, instance, element, points)
    if not points:
        return
    node, name, prop_schema, rtype = rng.choice(points)
    # rtype drives grants/permissions/typed-string generation.
    gen.rtype = rtype
    value = gen.gen(prop_schema, 1, name)
    if not is_empty(value):
        node[name] = value


def mutate(config, seed, schema=None, unique="fuzz"):
    # Without a schema only the destructive mutations run; the selftest uses that path for
    # configs that stay stable as the schema grows.
    rng = random.Random(seed)
    gen = Generator(schema, rng, unique) if schema is not None else None

    # Only inside resource instances, so the bundle/resources skeleton survives.
    roots = []
    for instances in config.get("resources", {}).values():
        if isinstance(instances, dict):
            roots.extend(v for v in instances.values() if isinstance(v, (dict, list)))

    for _ in range(rng.randint(1, 3)):
        if gen is not None and rng.random() < ADD_PROB:
            add_field(gen, rng, config)
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
    unique = os.environ["UNIQUE_NAME"]
    with open(path) as f:
        config = load_yaml(substitute_variables(f.read()))
    with open(os.environ["FUZZ_SCHEMA"]) as f:
        schema = json.load(f)
    sys.stdout.write(dump_yaml(mutate(config, seed, schema=schema, unique=unique)))


if __name__ == "__main__":
    main()
