"""
Schema-driven value generation for the invariant fuzzer.

mutate_fuzz_config.py injects optional fields into curated configs and asks Generator for each
field's value. Free-form scalars are sometimes replaced with dangerous values
(DANGEROUS_STRINGS/INTS) to probe input handling.

A seed is tied to schema iteration order, so adding a field moves every later draw.
"""

import os
import sys

# Depth past which optional properties are no longer emitted, to keep configs from exploding.
MAX_DEPTH = 6

# Hard cap on object/array nesting, which MAX_DEPTH leaves unbounded for required fields: a
# required-only cycle (task -> for_each_task -> task) would exhaust the stack. Branch descent and
# $ref chains are not counted.
MAX_RECURSION = 30

# The ${...} interpolation branch the schema wraps every field in (see
# bundle/internal/schema/main.go addInterpolationPatterns); we emit concrete values.
INTERPOLATION_MARKER = "\\$\\{"

# The standard seeded catalog/schema. A random name deploys on the fake server but real UC rejects
# it (CATALOG_DOES_NOT_EXIST), dropping the config.
DEFAULT_CATALOG = "main"
DEFAULT_SCHEMA = "default"

# "account users" exists on every workspace. A random principal or a privilege that does not apply
# to the securable deploys on the fake server but fails on UC.
DEFAULT_PRINCIPAL = "account users"
# Only types in mutate_fuzz_config.MUTATE_BASES that declare grants.
GRANT_PRIVILEGE = {
    "catalogs": "USE_CATALOG",
    "schemas": "USE_SCHEMA",
    "volumes": "READ_VOLUME",
    "registered_models": "EXECUTE",
    "external_locations": "READ_FILES",
}

# Permissions take no variable refs: a concrete principal and a level valid for the resource type.
DEFAULT_PERMISSION_GROUP = "users"
# Only types in mutate_fuzz_config.MUTATE_BASES that declare permissions.
PERMISSION_LEVEL = {
    "jobs": "CAN_VIEW",
    "model_serving_endpoints": "CAN_VIEW",
    "models": "CAN_READ",
    "pipelines": "CAN_VIEW",
    "secret_scopes": "READ",
    "sql_warehouses": "CAN_VIEW",
}

# Backend-computed fields, mirroring output_only in dresources/resources.yml: emitting them causes
# false drift after migrate. Only what the schema's own annotations miss, since it already drops
# bundle:"readonly" and OUTPUT_ONLY fields. Blocked by name everywhere, so a writable exception
# (an external volume's storage_location) needs a curated config.
SKIP_PROPERTY_NAMES = frozenset(
    {
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

# Absolute means already-remote, skipping the local-notebook check a bare token would fail.
NOTEBOOK_PATH = "/Shared/notebook"

# The CLI re-adds the /Workspace prefix on read, so a mismatched folder plans a spurious recreate.
PARENT_PATH = "/Workspace/Shared"

# String in the schema, parsed as protobuf.Duration at load; a bare token fails to parse.
DURATION_VALUE = "3600s"

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

# Only sometimes, so the config usually still deploys and exercises the invariant, not rejection.
DANGEROUS_PROB = 0.15


def token(rng):
    return "fuzz_" + "".join(rng.choice("abcdefghijklmnopqrstuvwxyz0123456789") for _ in range(8))


def is_empty(value):
    # Empty containers are not neutral: they are the shape behind several already-fixed drift bugs,
    # so emitting one spends seeds re-finding them. mutate_once still injects them deliberately.
    return value is None or value == {} or value == []


class Generator:
    def __init__(self, schema, rng, unique):
        self.root = schema
        self.rng = rng
        self.unique = unique
        # Set before generating a value, so grants/permissions can pick a valid value for the type.
        self.rtype = None
        # Distinguishes the pinned name/display_name values within one value; see gen_scalar.
        self.name_count = 0

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

    def should_skip_property(self, prop_name):
        return prop_name in SKIP_PROPERTY_NAMES

    def gen(self, schema, depth, name=""):
        if depth > MAX_RECURSION:
            sys.exit(f"gen_fuzz_config: schema walk exceeded {MAX_RECURSION} levels at {name!r}")

        schema = self.resolve(schema)
        if not isinstance(schema, dict) or not schema:
            return self.gen_scalar({"type": "string"}, name)

        # By name at any depth, which is safe because only resource elements declare either.
        if name == "grants":
            return self.gen_grants()
        if name == "permissions":
            return self.gen_permissions()

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
            if self.should_skip_property(prop_name):
                continue
            # Sampled, and dropped past MAX_DEPTH, so configs stay deployable within a seed's time.
            keep = prop_name in required or (depth < MAX_DEPTH and self.rng.random() < 0.35)
            if not keep:
                continue
            value = self.gen(prop_schema, depth + 1, prop_name)
            # Required properties included: a deep enough one can go missing here or in gen_array,
            # and the CLI then rejects the config. A normal fuzz outcome, not a lost seed.
            if is_empty(value):
                continue
            result[prop_name] = value

        # Map type: synthesize a few random keys, e.g. string maps like tags.
        if self.is_map(schema):
            for _ in range(self.rng.randint(1, 2)):
                key = token(self.rng)
                value = self.gen(schema["additionalProperties"], depth + 1, key)
                if not is_empty(value):
                    result[key] = value

        return result

    def gen_array(self, schema, depth, name):
        items = schema.get("items")
        if not items or depth >= MAX_DEPTH:
            return None
        values = [self.gen(items, depth + 1, name) for _ in range(self.rng.randint(1, 3))]
        values = [v for v in values if not is_empty(v)]
        return values or None

    def gen_grants(self):
        # No valid privilege means no grants node: UC rejects a wrong one, and an empty one only
        # reproduces the known drift bugs.
        privilege = GRANT_PRIVILEGE.get(self.rtype)
        if privilege is None:
            return None
        return [{"principal": DEFAULT_PRINCIPAL, "privileges": [privilege]}]

    def gen_permissions(self):
        # As gen_grants: no valid level means no permissions node.
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
        # A string, or no type at all ("any"). Pin the typed-string fields: a random token fails
        # format or existence validation.
        if name == "catalog_name":
            return DEFAULT_CATALOG
        if name == "schema_name":
            return DEFAULT_SCHEMA
        if name == "warehouse_id":
            # Always set by the harness; a KeyError beats silently rejecting every seed that uses one.
            return os.environ["TEST_DEFAULT_WAREHOUSE_ID"]
        if name == "notebook_path":
            return NOTEBOOK_PATH
        if name == "parent_path":
            return PARENT_PATH
        if name.endswith("_duration") or name == "ttl":
            return DURATION_VALUE
        if name in ("name", "display_name"):
            # Numbered: this pins by leaf name at any depth, and an array of named objects (job
            # parameters) would otherwise repeat one value and be rejected as a duplicate.
            self.name_count += 1
            return f"fuzz-{name}-{self.unique}-{self.name_count}"
        # No pinned meaning (description, comment, tag), so safe to probe here.
        if self.rng.random() < DANGEROUS_PROB:
            return self.rng.choice(DANGEROUS_STRINGS)
        return token(self.rng)


def object_branch(schema, what):
    for branch in schema["oneOf"]:
        if branch.get("type") == "object":
            return branch
    sys.exit(f"gen_fuzz_config: no object branch in {what}")


def resource_types(gen):
    # resources is oneOf[{ object with one property per resource type }].
    resources = gen.resolve(gen.root["properties"]["resources"])
    return object_branch(resources, "resources")["properties"]


def resource_element(gen, type_schema):
    # Each type is a map; the element schema is the object branch's additionalProperties.
    map_schema = gen.resolve(type_schema)
    return object_branch(map_schema, "resource type map")["additionalProperties"]
