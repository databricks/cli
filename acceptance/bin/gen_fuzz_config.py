#!/usr/bin/env python3
"""
Generate a random bundle config from the bundle JSON schema.

Walks `databricks bundle schema` (resolving $ref, picking concrete oneOf/anyOf
branches) and emits one or more random resources as databricks.yml, seeded by --seed.
With --resource-count > 1 it also links resources with ${resources.*} references (each
resource referencing an earlier one) so the interpolation and deploy-ordering machinery is
exercised. Free-form scalars are occasionally replaced with dangerous / near-range-end
values (DANGEROUS_STRINGS, DANGEROUS_INTS) to probe the CLI's input handling. Feeds the
invariant tests; the harness filters out configs the CLI rejects, so output may be
structurally-random but sometimes invalid.
"""

import argparse
import json
import os
import random
import re
import sys

# The schema is recursive (e.g. task -> for_each_task -> task); cap the walk.
MAX_DEPTH = 6

# The ${...} interpolation branch the schema wraps every field in (see
# bundle/internal/schema/main.go addInterpolationPatterns); we emit concrete values.
INTERPOLATION_MARKER = "\\$\\{"

# Types the generator can produce; keep in sync with libs/jsonschema.Type.
SCALAR_TYPES = {"boolean", "integer", "number", "string"}
HANDLED_TYPES = SCALAR_TYPES | {"object", "array"}

# Cross-resource references must resolve to objects that exist on every
# workspace (the fake test server and real UC alike). "main"/"default" are the
# standard seeded catalog/schema; these mirror acceptance/bundle/invariant/configs.
# Without pinning, the generator emits random names that the fake server accepts
# but real UC rejects (e.g. CATALOG_DOES_NOT_EXIST), so the config is dropped at
# deploy and never exercises the invariant.
DEFAULT_CATALOG = "main"
DEFAULT_SCHEMA = "default"

# "account users" is a group present on every workspace, plus one privilege UC
# accepts for each grant-bearing securable type (from the curated configs). Real
# UC rejects an unknown principal or a privilege that doesn't apply to the
# securable, so a random grant would deploy on the fake server yet fail on cloud.
DEFAULT_PRINCIPAL = "account users"
GRANT_PRIVILEGE = {
    "catalogs": "USE_CATALOG",
    "schemas": "USE_SCHEMA",
    "volumes": "READ_VOLUME",
    "registered_models": "EXECUTE",
    "external_locations": "READ_FILES",
    "vector_search_indexes": "SELECT",
}

# Permissions blocks cannot be variable references; each entry needs a concrete
# principal and a level valid for the resource type (see invariant configs).
DEFAULT_PERMISSION_GROUP = "users"
PERMISSION_LEVEL = {
    "alerts": "CAN_MANAGE",
    "apps": "CAN_USE",
    "clusters": "CAN_ATTACH_TO",
    "dashboards": "CAN_READ",
    "database_instances": "CAN_USE",
    "experiments": "CAN_READ",
    "genie_spaces": "CAN_READ",
    "jobs": "CAN_VIEW",
    "model_serving_endpoints": "CAN_VIEW",
    "models": "CAN_READ",
    "pipelines": "CAN_VIEW",
    "postgres_projects": "CAN_USE",
    "secret_scopes": "READ",
    "sql_warehouses": "CAN_VIEW",
    "vector_search_endpoints": "CAN_USE",
}

# Fields the bundle schema still lists but the user never sets (backend output /
# computed). Emitting them causes false drift after terraform→direct migrate.
# Keep in sync with bundle/direct/dresources/resources.yml output_only and
# backend_defaults where the field is not user-writable.
SKIP_PROPERTY_NAMES = frozenset(
    {
        "browse_only",
        "created_at",
        "created_by",
        "creator_name",
        # An etag is a read value the backend assigns; the CLI rejects one set in
        # bundle config (e.g. "genie space ... has an etag set. Etags must not be set").
        "etag",
        "full_name",
        "metastore_id",
        "owner",
        "storage_location",
        "updated_at",
        "updated_by",
    }
)

# Resource types whose schema omits required[] (or whose required[] can't be honored
# in bundle YAML) but which need these fields to deploy. See RESOURCE_FIELD_ALLOWLIST
# and the *_BY_RESOURCE tables below for the values these fields take.
RESOURCE_REQUIRED_FIELDS = {
    "registered_models": frozenset({"catalog_name", "name", "schema_name"}),
    "dashboards": frozenset({"display_name", "file_path", "warehouse_id"}),
    "alerts": frozenset({"display_name", "file_path", "warehouse_id"}),
    "apps": frozenset({"name", "source_code_path"}),
    "genie_spaces": frozenset({"serialized_space", "title", "warehouse_id"}),
}

# Fields to drop for a specific resource type because they conflict with the field
# set we do emit. Dashboards and Genie spaces take their body from file_path XOR an
# inline serialized_* field, so emitting both is rejected ("both ... are set").
RESOURCE_SKIP_FIELDS = {
    "dashboards": frozenset({"serialized_dashboard"}),
    "genie_spaces": frozenset({"file_path"}),
    "apps": frozenset({"git_repository", "git_source"}),
}

# Resource types where only a fixed field set is allowed in bundle YAML. Alerts read
# their spec from the .dbalert.json referenced by file_path; the CLI rejects any other
# field (see bundle/config/mutator/load_dbalert_files.go allowedInYAML).
RESOURCE_FIELD_ALLOWLIST = {
    "alerts": frozenset({"display_name", "file_path", "lifecycle", "permissions", "warehouse_id"}),
}

# file_path points at a serialized-body fixture copied into every seed dir from
# acceptance/bundle/invariant/data (see fuzz/script). The extension selects the parser.
FILE_PATH_BY_RESOURCE = {
    "dashboards": "./dashboard.lvdash.json",
    "alerts": "./alert.dbalert.json",
}

# A local directory holding app source, also copied in from data/.
APP_SOURCE_CODE_PATH = "./app"

# An absolute workspace path is treated as already-remote, skipping the local-notebook
# existence/extension check a bare token would fail.
NOTEBOOK_PATH = "/Shared/notebook"

# Fields declared as string in the schema but parsed as google.protobuf.Duration at
# config load (e.g. suspend_timeout_duration, ttl); a bare token fails to parse.
DURATION_VALUE = "3600s"

# Dangerous / near-range-end probes injected into free-form scalars: empty and
# whitespace-only strings, an over-long string, embedded newlines/tabs, non-ASCII, quotes,
# a dangling ${...} reference, a path-traversal string, and int32/int64 boundaries. The CLI
# must reject or round-trip these without panicking; mutate_fuzz_config.py reuses both lists.
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
    2**31,
    -(2**31),
    2**63 - 1,
    -1,
]

# Only inject a dangerous value some of the time: a fuzzed field mostly keeps a plausible
# value so the config still deploys and exercises the invariant, not just the reject path.
DANGEROUS_PROB = 0.15


class Generator:
    def __init__(self, schema, rng, unique):
        self.root = schema
        self.rng = rng
        self.unique = unique
        # Set to the top-level resource type before generating its element, so
        # grants can pick a privilege valid for that securable.
        self.rtype = None

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

    def field_behaviors(self, schema):
        if not isinstance(schema, dict):
            return []
        resolved = self.resolve(schema)
        behaviors = list(schema.get("x-databricks-field-behaviors", []))
        if resolved is not schema:
            behaviors.extend(resolved.get("x-databricks-field-behaviors", []))
        return behaviors

    def should_skip_property(self, prop_name, prop_schema):
        if prop_name in SKIP_PROPERTY_NAMES:
            return True
        if prop_name in RESOURCE_SKIP_FIELDS.get(self.rtype, ()):
            return True
        resolved = self.resolve(prop_schema)
        if "OUTPUT_ONLY" in self.field_behaviors(prop_schema):
            return True
        if resolved.get("readOnly"):
            return True
        return False

    def gen(self, schema, depth, name=""):
        # A Genie space body is a free-form interface{}; the backend rejects unknown
        # keys, so emit the minimal accepted body instead of a random object.
        if name == "serialized_space":
            return {"version": 1}

        schema = self.resolve(schema)
        if not isinstance(schema, dict) or not schema:
            return self.gen_scalar({"type": "string"}, name)

        if name == "grants":
            return self.gen_grants()
        if name == "permissions":
            return self.gen_permissions()

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
        allowlist = None
        if depth == 0 and self.rtype:
            required |= RESOURCE_REQUIRED_FIELDS.get(self.rtype, set())
            allowlist = RESOURCE_FIELD_ALLOWLIST.get(self.rtype)
        result = {}

        for prop_name, prop_schema in props.items():
            # A restricted resource (e.g. alerts) rejects any field outside its
            # allow-list, even a schema-required one supplied via the file instead.
            if allowlist is not None and prop_name not in allowlist:
                continue
            if self.should_skip_property(prop_name, prop_schema):
                continue
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

    def gen_grants(self):
        # One known-good grant for the current securable. Skip grants for a type
        # we have no valid privilege for, rather than emit one real UC rejects.
        privilege = GRANT_PRIVILEGE.get(self.rtype)
        if privilege is None:
            return []
        return [{"principal": DEFAULT_PRINCIPAL, "privileges": [privilege]}]

    def gen_permissions(self):
        # One known-good permission for the current resource. Skip types we have
        # no valid level for, rather than emit a random principal or ${...} ref.
        level = PERMISSION_LEVEL.get(self.rtype)
        if level is None:
            return []
        return [{"level": level, "group_name": DEFAULT_PERMISSION_GROUP}]

    def gen_scalar(self, schema, name):
        t = schema.get("type")
        if t == "boolean":
            # destroy_recreate invariant requires destroy to succeed.
            if name == "prevent_destroy":
                return False
            return self.rng.choice([True, False])
        if t == "integer":
            # The field is in hours, but UC validates it as a window of 0 or 7-30
            # days; only 0 or 168-720 (hours) are accepted.
            if name == "custom_max_retention_hours":
                return self.rng.choice([0, self.rng.randint(168, 720)])
            if self.rng.random() < DANGEROUS_PROB:
                return self.rng.choice(DANGEROUS_INTS)
            return self.rng.choice([0, 1, self.rng.randint(2, 1000)])
        if t == "number":
            return round(self.rng.uniform(0, 1000), 2)
        # Fail loud on an unknown type; a missing type is "any" and falls through to string.
        if t is not None and t not in SCALAR_TYPES:
            sys.exit(f"gen_fuzz_config: unhandled schema type {t!r}")
        # string (default)
        # Pin cross-resource references and typed-string fields to values the backend
        # accepts; a random token fails format/existence validation and drops the config.
        if name == "catalog_name":
            return DEFAULT_CATALOG
        if name == "schema_name":
            return DEFAULT_SCHEMA
        if name == "warehouse_id":
            return os.environ.get("TEST_DEFAULT_WAREHOUSE_ID", "")
        if name == "notebook_path":
            return NOTEBOOK_PATH
        if name == "source_code_path":
            return APP_SOURCE_CODE_PATH
        if name == "file_path":
            return FILE_PATH_BY_RESOURCE.get(self.rtype, self.token())
        if name.endswith("_duration") or name == "ttl":
            return DURATION_VALUE
        if name == "name" and self.rtype == "vector_search_indexes":
            # UC requires the full three-level catalog.schema.table name, and each
            # part accepts only alphanumerics and underscores.
            table = re.sub(r"[^0-9a-zA-Z_]", "_", f"fuzz_index_{self.unique}")
            return f"{DEFAULT_CATALOG}.{DEFAULT_SCHEMA}.{table}"
        if name in ("name", "display_name"):
            return f"fuzz-{name}-{self.unique}"
        # A free-form string with no pinned meaning (e.g. description, comment, tag value):
        # probe dangerous / near-range-end input here, where a rejected or normalized value
        # doesn't just fail the field-format check a pinned field above guards against.
        if self.rng.random() < DANGEROUS_PROB:
            return self.rng.choice(DANGEROUS_STRINGS)
        return self.token()

    def token(self):
        return "fuzz_" + "".join(self.rng.choice("abcdefghijklmnopqrstuvwxyz0123456789") for _ in range(8))


def resource_types(schema, gen):
    # resources is oneOf[{ object with one property per resource type }].
    resources = gen.resolve(schema["properties"]["resources"])
    obj = next(b for b in resources["oneOf"] if b.get("type") == "object")
    return obj["properties"]


def gen_resource(schema, gen, types, candidates, seed, unique, index):
    rtype = gen.rng.choice(sorted(candidates))

    # Each resource type is a map ref; the element schema is the object branch's
    # additionalProperties.
    map_schema = gen.resolve(types[rtype])
    obj = next(b for b in map_schema["oneOf"] if b.get("type") == "object")
    element = obj["additionalProperties"]

    # The first resource keeps the bare key/name so a seed produces the same first
    # resource regardless of --resource-count; later resources are index-suffixed to
    # stay unique within the config.
    if index == 0:
        key = f"fuzz_{rtype}_{seed}"
        gen.unique = unique
    else:
        key = f"fuzz_{rtype}_{seed}_{index}"
        gen.unique = f"{unique}-{index}"
    gen.rtype = rtype
    instance = gen.gen(element, 0)
    return rtype, key, instance, gen.resolve(element)


def object_properties(gen, schema):
    # The resource element is oneOf[object, ${...} string]; return the object
    # branch's properties, matching the branch gen() picks to build the instance.
    schema = gen.resolve(schema)
    if "properties" in schema:
        return schema["properties"]
    for key in ("oneOf", "anyOf"):
        for branch in schema.get(key, []):
            resolved = gen.resolve(branch)
            if "properties" in resolved:
                return resolved["properties"]
    return {}


def cross_ref_field(gen, element):
    # A free-text scalar safe to overwrite with a reference; both names cover most
    # resource types (jobs use "description", UC resources use "comment").
    props = object_properties(gen, element)
    for field in ("description", "comment"):
        if field in props:
            return field
    return None


def target_ref_field(instance):
    # Reference the target's identity field: a string, so the type stays compatible
    # with the description/comment field it lands in, and an input (not an output like
    # ".id") so it resolves for every resource type and converges without drift.
    # Output-field references are covered by the curated cross-ref configs.
    for field in ("name", "display_name"):
        if isinstance(instance.get(field), str):
            return field
    return None


def inject_cross_ref(gen, records):
    # Link resources so deploy has to order them and resolve the references. A
    # record may only reference an earlier one, so the reference graph stays
    # acyclic: deploy must topologically order resources, and a cycle can't be
    # ordered (the config would be rejected instead of exercising the invariant).
    if len(records) < 2:
        return
    for i, source in enumerate(records):
        if not source["ref_field"]:
            continue
        targets = [t for t in records[:i] if target_ref_field(t["instance"])]
        if not targets:
            continue
        target = gen.rng.choice(targets)
        field = target_ref_field(target["instance"])
        source["instance"][source["ref_field"]] = f"${{resources.{target['rtype']}.{target['key']}.{field}}}"


def gen_config(schema, seed, unique, allowed, resource_count=1):
    if resource_count < 1:
        sys.exit(f"gen_fuzz_config: --resource-count must be >= 1, got {resource_count}")

    gen = Generator(schema, random.Random(seed), unique)

    types = resource_types(schema, gen)
    candidates = [t for t in types if not allowed or t in allowed]
    if not candidates:
        sys.exit(f"no resource types to generate from (allowed={sorted(allowed)})")

    records = []
    for index in range(resource_count):
        rtype, key, instance, element = gen_resource(schema, gen, types, candidates, seed, unique, index)
        records.append({"rtype": rtype, "key": key, "instance": instance, "ref_field": cross_ref_field(gen, element)})

    inject_cross_ref(gen, records)

    resources = {}
    for record in records:
        resources.setdefault(record["rtype"], {})[record["key"]] = record["instance"]

    return {
        "bundle": {"name": f"fuzz-{unique}"},
        "resources": resources,
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
                out += f"{prefix}{k}: {dump_scalar(v)}\n"
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
                out += f"{pad}- {dump_scalar(item)}\n"
        return out
    return f"{pad}{dump_scalar(obj)}\n"


def dump_scalar(v):
    # ensure_ascii=False keeps non-ASCII as literal UTF-8. The default would escape an
    # astral char (e.g. the 🚀 probe) into a UTF-16 surrogate pair (\ud83d\ude80), which
    # YAML's parser rejects as an "invalid Unicode character escape code" -- so the config
    # dies at parse time and never reaches bundle logic. A literal UTF-8 scalar is valid
    # YAML and exercises the CLI's actual unicode handling instead. Control chars (\n, \t)
    # are still escaped by json.dumps regardless, which YAML accepts.
    return json.dumps(v, ensure_ascii=False)


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
    parser.add_argument(
        "--resource-count",
        type=int,
        default=1,
        help="Number of resources to emit (default: 1)",
    )
    args = parser.parse_args()

    with open(args.schema) as f:
        schema = json.load(f)

    allowed = {r.strip() for r in args.resources.split(",") if r.strip()}
    config = gen_config(schema, args.seed, args.unique, allowed, args.resource_count)
    sys.stdout.write(to_yaml(config))


if __name__ == "__main__":
    main()
