"""
Generate a random bundle config from the bundle JSON schema.

Walks `databricks bundle schema` (resolving $ref, picking concrete oneOf/anyOf branches) and emits
one random resource, seeded by the caller. Free-form scalars are sometimes replaced with dangerous
values (DANGEROUS_STRINGS/INTS) to probe input handling. The harness drops configs the CLI
rejects, so output may be structurally random but invalid.

Used as a library by emit_fuzz_config.py and mutate_fuzz_config.py.
"""

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

# Keep in sync with libs/jsonschema.Type. gen_scalar exits on anything else.
SCALAR_TYPES = {"boolean", "integer", "number", "string"}

# Cross-resource refs must resolve on every workspace (fake server and real UC). "main"/"default"
# are the standard seeded catalog/schema; a random name deploys on the fake server but real UC
# rejects it (CATALOG_DOES_NOT_EXIST), dropping the config.
DEFAULT_CATALOG = "main"
DEFAULT_SCHEMA = "default"

# "account users" exists on every workspace; each securable gets one privilege UC accepts. A random
# principal or inapplicable privilege deploys on the fake server but fails on UC.
DEFAULT_PRINCIPAL = "account users"
GRANT_PRIVILEGE = {
    "catalogs": "USE_CATALOG",
    "schemas": "USE_SCHEMA",
    "volumes": "READ_VOLUME",
    "registered_models": "EXECUTE",
    "external_locations": "READ_FILES",
    "vector_search_indexes": "SELECT",
}

# Permissions can't be variable refs; each entry needs a concrete principal and a level valid for
# the resource type.
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

# Fields the backend computes; emitting them causes false drift after migrate. Mirrors
# output_only/backend_defaults in dresources/resources.yml. Blocked by name everywhere, so writable
# exceptions (an external volume's storage_location) need a curated config.
SKIP_PROPERTY_NAMES = frozenset(
    {
        "browse_only",
        "created_at",
        "created_by",
        "creator_name",
        # Backend-assigned; the CLI rejects an etag set in bundle config.
        "etag",
        "full_name",
        "metastore_id",
        "owner",
        "storage_location",
        "updated_at",
        "updated_by",
    }
)

# Fields these resources need to deploy but that the schema's required[] omits (or can't express in
# YAML). Values come from the *_BY_RESOURCE tables below.
RESOURCE_REQUIRED_FIELDS = {
    "registered_models": frozenset({"catalog_name", "name", "schema_name"}),
    "dashboards": frozenset({"display_name", "file_path", "warehouse_id"}),
    "alerts": frozenset({"display_name", "file_path", "warehouse_id"}),
    "apps": frozenset({"name", "source_code_path"}),
    "genie_spaces": frozenset({"serialized_space", "title", "warehouse_id"}),
}

# Fields that conflict with the set we emit. Dashboards/Genie spaces take their body from file_path
# XOR an inline serialized_* field; emitting both is rejected.
RESOURCE_SKIP_FIELDS = {
    "dashboards": frozenset({"serialized_dashboard"}),
    "genie_spaces": frozenset({"file_path"}),
    "apps": frozenset({"git_repository", "git_source"}),
}

# Resources allowing only a fixed field set in YAML. Alerts read their spec from the
# .dbalert.json at file_path; the CLI rejects other fields (load_dbalert_files.go).
RESOURCE_FIELD_ALLOWLIST = {
    "alerts": frozenset({"display_name", "file_path", "lifecycle", "permissions", "warehouse_id"}),
}

# Serialized-body fixtures copied into each seed dir from invariant/data; the extension selects the
# parser.
FILE_PATH_BY_RESOURCE = {
    "dashboards": "./dashboard.lvdash.json",
    "alerts": "./alert.dbalert.json",
}

# A local directory holding app source, also copied in from data/.
APP_SOURCE_CODE_PATH = "./app"

# An absolute workspace path is treated as already-remote, skipping the local-notebook
# existence/extension check a bare token would fail.
NOTEBOOK_PATH = "/Shared/notebook"

# parent_path is a workspace folder; pin it to a valid one. The CLI re-adds the /Workspace prefix
# on read, so a mismatched value plans a spurious recreate.
PARENT_PATH = "/Workspace/Shared"

# String in the schema but parsed as protobuf.Duration at load (suspend_timeout_duration, ttl); a
# bare token fails to parse.
DURATION_VALUE = "3600s"

# Dangerous/near-range-end probes for free-form scalars. The CLI must reject or round-trip these
# without panicking; mutate_fuzz_config reuses them.
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

# Inject a dangerous value only sometimes, so the config usually still deploys and exercises the
# invariant, not just the reject path.
DANGEROUS_PROB = 0.15


class Generator:
    def __init__(self, schema, rng, unique):
        self.root = schema
        self.rng = rng
        self.unique = unique
        # Top-level resource type, set before generating its element so grants/permissions can pick
        # a value valid for that securable.
        self.rtype = None

    def resolve(self, schema):
        # Follow $ref chains ("#/$defs/.../resources.Job"), indexing $defs by path segment.
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
        # A Genie space body is free-form but the backend rejects unknown keys, so emit the minimal
        # accepted body instead of a random object.
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
            # A restricted resource (e.g. alerts) rejects any field outside its allow-list, even a
            # schema-required one it reads from the file instead.
            if allowlist is not None and prop_name not in allowlist:
                continue
            if self.should_skip_property(prop_name, prop_schema):
                continue
            # Emit optional fields less often deeper down to keep configs from exploding.
            keep = prop_name in required or (depth < MAX_DEPTH and self.rng.random() < 0.35)
            if not keep:
                continue
            value = self.gen(prop_schema, depth + 1, prop_name)
            # Drop an object whose every property was skipped: `{}` carries no information and some
            # fields reject it outright.
            if value is None or value == {}:
                continue
            result[prop_name] = value

        # Map type: synthesize a few random keys, e.g. resources.<type> or string maps like tags.
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
        # One known-good grant for the securable. No valid privilege means no grants node: UC
        # rejects a wrong one, and an empty one only reproduces the known drift bugs.
        privilege = GRANT_PRIVILEGE.get(self.rtype)
        if privilege is None:
            return None
        return [{"principal": DEFAULT_PRINCIPAL, "privileges": [privilege]}]

    def gen_permissions(self):
        # As gen_grants: no valid level means no permissions node, rather than a random principal, a
        # ${...} ref, or an empty list.
        level = PERMISSION_LEVEL.get(self.rtype)
        if level is None:
            return None
        return [{"level": level, "group_name": DEFAULT_PERMISSION_GROUP}]

    def gen_scalar(self, schema, name):
        t = schema.get("type")
        if t == "boolean":
            # Cleanup must be able to destroy the bundle.
            if name == "prevent_destroy":
                return False
            return self.rng.choice([True, False])
        if t == "integer":
            # In hours, but UC accepts only a window of 0 or 7-30 days (0 or 168-720 hours).
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
        # Pin cross-resource refs and typed-string fields to accepted values; a random token fails
        # format/existence validation and drops the config.
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
        if name == "parent_path":
            return PARENT_PATH
        if name == "file_path":
            return FILE_PATH_BY_RESOURCE.get(self.rtype, self.token())
        if name.endswith("_duration") or name == "ttl":
            return DURATION_VALUE
        if name == "name" and self.rtype == "vector_search_indexes":
            # UC requires the full catalog.schema.table name; each part is alphanumeric+_.
            table = re.sub(r"[^0-9a-zA-Z_]", "_", f"fuzz_index_{self.unique}")
            return f"{DEFAULT_CATALOG}.{DEFAULT_SCHEMA}.{table}"
        if name in ("name", "display_name"):
            return f"fuzz-{name}-{self.unique}"
        # Free-form string with no pinned meaning (description, comment, tag): safe to probe
        # dangerous input here, unlike the pinned fields above.
        if self.rng.random() < DANGEROUS_PROB:
            return self.rng.choice(DANGEROUS_STRINGS)
        return self.token()

    def token(self):
        return "fuzz_" + "".join(self.rng.choice("abcdefghijklmnopqrstuvwxyz0123456789") for _ in range(8))


def resource_types(gen):
    # resources is oneOf[{ object with one property per resource type }].
    resources = gen.resolve(gen.root["properties"]["resources"])
    obj = next(b for b in resources["oneOf"] if b.get("type") == "object")
    return obj["properties"]


def resource_element(gen, type_schema):
    # Each type is a map; the element schema is the object branch's additionalProperties.
    map_schema = gen.resolve(type_schema)
    obj = next(b for b in map_schema["oneOf"] if b.get("type") == "object")
    return obj["additionalProperties"]


def gen_config(schema, seed, unique, allowed=frozenset()):
    gen = Generator(schema, random.Random(seed), unique)

    types = resource_types(gen)
    candidates = [t for t in types if not allowed or t in allowed]
    if not candidates:
        sys.exit(f"no resource types to generate from (allowed={sorted(allowed)})")

    rtype = gen.rng.choice(sorted(candidates))
    gen.rtype = rtype
    instance = gen.gen(resource_element(gen, types[rtype]), 0)

    return {
        # Same name shape as the curated configs, so targets that derive workspace paths from the
        # bundle name work unchanged.
        "bundle": {"name": f"test-bundle-{unique}"},
        "resources": {rtype: {f"fuzz_{rtype}_{seed}": instance}},
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
            return f"{pad}- []\n" if list_item else f"{pad}[]\n"
        # A list inside a list: the marker needs its own line, else the two flatten into one.
        if list_item:
            return f"{pad}-\n" + to_yaml(obj, indent + 1)
        out = ""
        for item in obj:
            if isinstance(item, (dict, list)):
                out += to_yaml(item, indent, list_item=True)
            else:
                out += f"{pad}- {dump_scalar(item)}\n"
        return out
    return f"{pad}{dump_scalar(obj)}\n"


def dump_scalar(v):
    # ensure_ascii=False keeps non-ASCII as literal UTF-8. The default escapes astral chars (e.g.
    # the rocket probe) into surrogate pairs that YAML rejects, killing the config at parse time
    # before it reaches bundle logic. Control chars stay escaped by json.dumps (YAML ok).
    return json.dumps(v, ensure_ascii=False)
