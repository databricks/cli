"""
Mutate a known-good bundle config by deleting, perturbing, and adding random fields.

Complements gen_fuzz_config.py: instead of building from the schema, this perturbs a curated
invariant config that already deploys, so it reaches a much higher deploy rate.

Two mutation kinds, chosen per step:

- destructive (always): delete a field or replace it with a token, a dangerous value, or an empty
  container. Stays within the base's fields, so it finds only reject/panic bugs.
- additive (with a schema): inject a valid optional field the base omits, valued by the schema
  generator. This is what reaches reconcile/drift bugs.

The harness only asserts no-panic on fuzzed configs, so an invalid mutation is fine: the CLI must
reject it cleanly, not crash.

Used as a library by emit_fuzz_config.py.
"""

import os
import random
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from gen_fuzz_config import DANGEROUS_INTS, DANGEROUS_STRINGS, Generator, resource_element, resource_types

DANGEROUS = DANGEROUS_STRINGS + DANGEROUS_INTS

# Chance a step injects a field rather than perturbing one. Biased high: injection is the path to
# drift bugs, and destructive coverage is already dense.
ADD_PROB = 0.6


def tokenize(text):
    # (indent, content) per non-blank, non-comment line. Only full-line comments are stripped; the
    # curated bases have no trailing "#" in values.
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
    # to_yaml emits empty containers in flow form; read them back so load -> emit -> load holds.
    if text == "[]":
        return []
    if text == "{}":
        return {}
    # Populated flow style is the one shape this loader cannot represent: "[id]" would read back as
    # the string "[id]", turning a list into a scalar, and load -> emit -> load stays a fixed point,
    # so the round-trip check cannot see it either. Exit so a new MUTATE_BASES entry fails the
    # selftest rather than silently costing coverage.
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
        # The item is its own block: the inline remainder (re-indented to child_indent) plus any
        # deeper continuation lines that belong to it.
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


def collect_insertions(gen, node, schema, rtype, out):
    # Record every writable optional field absent from an object, walking node and schema together
    # so nested objects are candidates too.
    schema = gen.resolve(schema)
    if not isinstance(schema, dict):
        return

    branches = schema.get("oneOf") or schema.get("anyOf")
    if branches:
        # Pick the branch matching the node we have, not a random one.
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
    types = resource_types(gen)
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
    # rtype drives grants/permissions/typed-string generation.
    gen.rtype = rtype
    value = gen.gen(prop_schema, 1, name)
    if value is not None:
        node[name] = value


def mutate(config, seed, schema=None, unique="fuzz"):
    # Without a schema only the destructive mutations run; the selftest uses that path to print
    # configs that don't churn as the schema grows.
    rng = random.Random(seed)
    gen = Generator(schema, rng, unique) if schema is not None else None

    # Mutate only inside resource instances: keep the bundle/name and resources skeleton so there
    # is always something to deploy, while every instance field is fair game.
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
