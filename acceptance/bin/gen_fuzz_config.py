#!/usr/bin/env python3
"""
Generate a random bundle config from the bundle JSON schema.

Walks `databricks bundle schema` (resolving $ref, picking concrete oneOf/anyOf
branches) and emits one random resource as databricks.yml, seeded by --seed. Feeds the
invariant tests; the harness filters out configs the CLI rejects, so output may be
structurally-random but sometimes invalid.
"""

import argparse
import json
import random
import sys

# The schema is recursive (e.g. task -> for_each_task -> task); cap the walk.
MAX_DEPTH = 6

# The ${...} interpolation branch the schema wraps every field in (see
# bundle/internal/schema/main.go addInterpolationPatterns); we emit concrete values.
INTERPOLATION_MARKER = "\\$\\{"


class Generator:
    def __init__(self, schema, rng, unique):
        self.root = schema
        self.rng = rng
        self.unique = unique

    def resolve(self, schema):
        # Follow $ref chains, e.g. "#/$defs/github.com/.../resources.Job", nested
        # under $defs by "/"-separated path segments.
        while isinstance(schema, dict) and "$ref" in schema:
            cur = self.root["$defs"]
            for part in schema["$ref"].split("/")[2:]:
                cur = cur[part]
            schema = cur
        return schema

    def is_interpolation(self, branch):
        return branch.get("type") == "string" and INTERPOLATION_MARKER in branch.get("pattern", "")

    def choose_branch(self, branches):
        # Prefer concrete branches over the ${...} alternatives.
        concrete = [b for b in branches if not self.is_interpolation(b)]
        return self.rng.choice(concrete or branches)

    def gen(self, schema, depth, name=""):
        schema = self.resolve(schema)
        if not isinstance(schema, dict) or not schema:
            return self.gen_scalar({"type": "string"}, name)

        if "const" in schema:
            return schema["const"]
        if schema.get("enum"):
            return self.rng.choice(schema["enum"])

        for key in ("oneOf", "anyOf"):
            if schema.get(key):
                return self.gen(self.choose_branch(schema[key]), depth, name)

        t = schema.get("type")
        if t == "object" or "properties" in schema or self.is_map(schema):
            return self.gen_object(schema, depth)
        if t == "array":
            return self.gen_array(schema, depth, name)
        return self.gen_scalar(schema, name)

    def is_map(self, schema):
        return isinstance(schema.get("additionalProperties"), dict) and not schema.get("properties")

    def gen_object(self, schema, depth):
        props = schema.get("properties", {})
        required = set(schema.get("required", []))
        result = {}

        for prop_name, prop_schema in props.items():
            # Always emit required fields; emit optional ones less often as we go
            # deeper to keep configs from exploding.
            keep = prop_name in required or (depth < MAX_DEPTH and self.rng.random() < 0.35)
            if not keep:
                continue
            value = self.gen(prop_schema, depth + 1, prop_name)
            if value is not None:
                result[prop_name] = value

        # Map type (additionalProperties, no fixed properties): synthesize a few
        # random keys, e.g. resources.<type> or string maps like tags.
        if self.is_map(schema):
            for _ in range(self.rng.randint(1, 2)):
                key = self.token()
                result[key] = self.gen(schema["additionalProperties"], depth + 1, key)

        return result

    def gen_array(self, schema, depth, name):
        items = schema.get("items")
        if not items or depth >= MAX_DEPTH:
            return []
        return [self.gen(items, depth + 1, name) for _ in range(self.rng.randint(1, 3))]

    def gen_scalar(self, schema, name):
        t = schema.get("type")
        if t == "boolean":
            return self.rng.choice([True, False])
        if t == "integer":
            # The field is in hours, but UC validates it as a window of 0 or 7-30
            # days; only 0 or 168-720 (hours) are accepted.
            if name == "custom_max_retention_hours":
                return self.rng.choice([0, self.rng.randint(168, 720)])
            return self.rng.choice([0, 1, self.rng.randint(2, 1000)])
        if t == "number":
            return round(self.rng.uniform(0, 1000), 2)
        # string (default)
        if name in ("name", "display_name"):
            return f"fuzz-{name}-{self.unique}"
        return self.token()

    def token(self):
        return "fuzz_" + "".join(self.rng.choice("abcdefghijklmnopqrstuvwxyz0123456789") for _ in range(8))


def resource_types(schema, gen):
    # resources is oneOf[{ object with one property per resource type }].
    resources = gen.resolve(schema["properties"]["resources"])
    obj = next(b for b in resources["oneOf"] if b.get("type") == "object")
    return obj["properties"]


def gen_config(schema, seed, unique, allowed):
    rng = random.Random(seed)
    gen = Generator(schema, rng, unique)

    types = resource_types(schema, gen)
    candidates = [t for t in types if not allowed or t in allowed]
    if not candidates:
        sys.exit(f"no resource types to generate from (allowed={sorted(allowed)})")
    rtype = rng.choice(sorted(candidates))

    # Each resource type is a map ref; the element schema is the object branch's
    # additionalProperties.
    map_schema = gen.resolve(types[rtype])
    obj = next(b for b in map_schema["oneOf"] if b.get("type") == "object")
    element = obj["additionalProperties"]

    key = f"fuzz_{rtype}_{seed}"
    instance = gen.gen(element, 0)
    return {
        "bundle": {"name": f"fuzz-{unique}"},
        "resources": {rtype: {key: instance}},
    }


def to_yaml(obj, indent=0, list_item=False):
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
                out += f"{prefix}{k}:\n" + to_yaml(v, child_indent)
            else:
                out += f"{prefix}{k}: {json.dumps(v)}\n"
            first = False
        return out
    if isinstance(obj, list):
        if not obj:
            return f"{pad}[]\n"
        out = ""
        for item in obj:
            if isinstance(item, (dict, list)):
                out += to_yaml(item, indent, list_item=True)
            else:
                out += f"{pad}- {json.dumps(item)}\n"
        return out
    return f"{pad}{json.dumps(obj)}\n"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--schema", required=True, help="Path to bundle JSON schema")
    parser.add_argument("--seed", type=int, required=True, help="RNG seed (for reproducibility)")
    parser.add_argument("--unique", default="local", help="Unique suffix for resource names")
    parser.add_argument(
        "--resources",
        default="",
        help="Comma-separated allow-list of resource types (default: all)",
    )
    args = parser.parse_args()

    with open(args.schema) as f:
        schema = json.load(f)

    allowed = {r.strip() for r in args.resources.split(",") if r.strip()}
    config = gen_config(schema, args.seed, args.unique, allowed)
    sys.stdout.write(to_yaml(config))


if __name__ == "__main__":
    main()
