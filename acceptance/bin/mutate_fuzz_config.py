#!/usr/bin/env python3
"""
Mutate a known-good bundle config by deleting, perturbing, and adding random fields.

Complements gen_fuzz_config.py (generate-from-scratch via schema walk): instead of
building a config from the schema, this starts from a curated invariant config that
already deploys and applies a few seeded mutations. It exercises the CLI's handling of
perturbed-but-realistic input, and reaches a much higher deploy rate than the schema
walk, since the base already resolves.

Two kinds of mutation, chosen per step:

- destructive (always): delete a field, or replace it with a fuzz token, a
  boundary/dangerous value, or an empty container. Probes the reject/no-panic path.
- additive (only with a schema): inject a valid optional field the base omits, valued by
  the schema generator. Destructive ops stay within the base's field set, so they only
  find reject/panic bugs; adding a valid optional field to a still-deploying config is
  what reaches reconcile/drift bugs (the field space the schema walk explores).

Reads the base databricks.yml (already envsubst-rendered) from stdin, writes the mutated
config to stdout. --seed makes the mutation reproducible; --schema enables the additive op.

The invariant harness only asserts no-panic on fuzzed configs (SKIP_DRIFT_CHECK), so a
mutation that makes the config invalid is fine: the CLI must reject it cleanly, not crash.
"""

import argparse
import json
import os
import random
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from gen_fuzz_config import DANGEROUS_INTS, DANGEROUS_STRINGS, Generator, resource_types, to_yaml

# Same near-range-end and dangerous-character probes the schema-walk generator injects into
# free-form scalars; here we drop them onto any field (see mutate_once).
DANGEROUS = DANGEROUS_STRINGS + DANGEROUS_INTS

# Chance a step injects a field rather than perturbing one. Biased high: injection is the
# path to drift bugs, and destructive coverage is already dense (1-3 steps per seed).
ADD_PROB = 0.6


def tokenize(text):
    # (indent, content) per non-blank, non-comment line. Only full-line comments are
    # stripped; the curated bases don't use trailing "#" in values.
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
    # Bare scalar: the whole block is a single value (e.g. a list scalar item).
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
        # The item is its own block: the inline remainder (re-indented to child_indent)
        # plus any deeper continuation lines that belong to it.
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
    # (container, key) for every child, so a mutation can delete or replace it in place.
    if isinstance(node, dict):
        for k, v in node.items():
            out.append((node, k))
            collect(v, out)
    elif isinstance(node, list):
        for idx, v in enumerate(node):
            out.append((node, idx))
            collect(v, out)


def token(rng):
    return "fuzz_" + "".join(rng.choice("abcdefghijklmnopqrstuvwxyz0123456789") for _ in range(8))


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


def resource_element(gen, type_schema):
    # The instance schema is the map's object-branch additionalProperties (as gen_resource).
    map_schema = gen.resolve(type_schema)
    obj = next(b for b in map_schema["oneOf"] if b.get("type") == "object")
    return obj["additionalProperties"]


def collect_insertions(gen, node, schema, rtype, out):
    # Record every writable optional field absent from an existing object, walking the node
    # alongside its schema so nested objects (not just the top level) are candidates too.
    schema = gen.resolve(schema)
    if not isinstance(schema, dict):
        return

    branches = schema.get("oneOf") or schema.get("anyOf")
    if branches:
        # Pick the branch matching the node we actually have, not a random one.
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
            if name not in node and not gen.should_skip_property(name, prop_schema):
                out.append((node, name, prop_schema, rtype))
        for key, value in node.items():
            if key in props and isinstance(value, (dict, list)):
                collect_insertions(gen, value, props[key], rtype, out)
        if gen.is_map(schema):
            for value in node.values():
                if isinstance(value, (dict, list)):
                    collect_insertions(gen, value, schema["additionalProperties"], rtype, out)
    elif isinstance(node, list):
        items = schema.get("items")
        if items:
            for value in node:
                if isinstance(value, (dict, list)):
                    collect_insertions(gen, value, items, rtype, out)


def add_field(gen, rng, config):
    # Inject one valid optional field, absent from the base, into a random insertion point.
    types = resource_types(gen.root, gen)
    points = []
    for rtype, instances in config.get("resources", {}).items():
        if rtype not in types or not isinstance(instances, dict):
            continue
        element = resource_element(gen, types[rtype])
        for instance in instances.values():
            if isinstance(instance, dict):
                gen.rtype = rtype
                collect_insertions(gen, instance, element, rtype, points)
    if not points:
        return
    node, name, prop_schema, rtype = rng.choice(points)
    # rtype drives grants/permissions/typed-string generation (see gen_scalar/gen_grants).
    gen.rtype = rtype
    value = gen.gen(prop_schema, 1, name)
    if value is not None:
        node[name] = value


def mutate(config, seed, schema=None, unique="fuzz"):
    rng = random.Random(seed)
    gen = Generator(schema, rng, unique) if schema is not None else None

    # Mutate only inside resource instances: keep bundle/name and the
    # resources.<type>.<key> skeleton so there is always something to deploy, while
    # every field of the instance (including required ones) is fair game.
    roots = []
    for instances in config.get("resources", {}).values():
        if isinstance(instances, dict):
            roots.extend(v for v in instances.values() if isinstance(v, (dict, list)))

    for _ in range(rng.randint(1, 3)):
        # gen is None short-circuits before rng is touched, so the no-schema path keeps its
        # exact RNG stream (and committed selftest output) unchanged.
        if gen is not None and rng.random() < ADD_PROB:
            add_field(gen, rng, config)
        else:
            mutate_once(rng, roots)

    return config


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--seed", type=int, required=True, help="RNG seed (for reproducibility)")
    parser.add_argument("--schema", help="Path to bundle JSON schema; enables valid-optional-field injection")
    parser.add_argument("--unique", default="fuzz", help="Unique suffix for injected field values")
    args = parser.parse_args()

    config = load_yaml(sys.stdin.read())
    if not isinstance(config, dict):
        sys.exit("mutate_fuzz_config: base config did not parse to a mapping")

    schema = None
    if args.schema:
        with open(args.schema) as f:
            schema = json.load(f)

    sys.stdout.write(to_yaml(mutate(config, args.seed, schema=schema, unique=args.unique)))


if __name__ == "__main__":
    main()
