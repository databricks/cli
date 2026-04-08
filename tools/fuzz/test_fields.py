#!/usr/bin/env python3
"""
Fuzz-test bundle field handling by iterating over all INPUT fields from refschema,
generating modified configs, and running invariant tests via the acceptance test harness.

Usage:
    python3 tools/fuzz/test_fields.py --config job --field description -n 5
    python3 tools/fuzz/test_fields.py --seed 42 -n 100
    python3 tools/fuzz/test_fields.py --test migrate --config schema -n 3
"""

import argparse
import json
import os
import random
import re
import subprocess
import sys
import tomllib
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent.parent
FIELDS_FILE = REPO_ROOT / "acceptance" / "bundle" / "refschema" / "out.fields.txt"
CONFIGS_DIR = REPO_ROOT / "acceptance" / "bundle" / "invariant" / "configs"
TEST_TOML = REPO_ROOT / "acceptance" / "bundle" / "invariant" / "test.toml"
GENERATED_PREFIX = "generated__"

# Configs from test.toml that the fuzzer can't handle (e.g., pydabs needs special setup).
SKIP_CONFIGS = {
    "job_pydabs_10_tasks.yml.tmpl",
    "job_pydabs_1000_tasks.yml.tmpl",
    "synced_database_table.yml.tmpl",
}


def parse_fields(path):
    """Parse out.fields.txt into list of (field_path, field_type, tags).

    >>> fields = parse_fields(FIELDS_FILE)
    >>> any(f[0] == "resources.jobs.*.name" for f in fields)
    True
    """
    fields = []
    with open(path) as f:
        for line in f:
            line = line.rstrip("\n")
            if not line:
                continue
            parts = line.split("\t")
            if len(parts) < 3:
                continue
            field_path = parts[0]
            field_type = parts[1]
            tags = parts[2:]
            fields.append((field_path, field_type, tags))
    return fields


def is_input_field(tags):
    """Check if field is available in config (INPUT or ALL).

    >>> is_input_field(["ALL"])
    True
    >>> is_input_field(["INPUT"])
    True
    >>> is_input_field(["REMOTE"])
    False
    >>> is_input_field(["INPUT", "STATE"])
    True
    """
    return "INPUT" in tags or "ALL" in tags


def extract_resource_type(field_path):
    """Extract resource type from field path like 'resources.jobs.*.name'.

    >>> extract_resource_type("resources.jobs.*.name")
    'jobs'
    >>> extract_resource_type("resources.model_serving_endpoints.*.config")
    'model_serving_endpoints'
    """
    parts = field_path.split(".")
    if len(parts) >= 3 and parts[0] == "resources":
        return parts[1]
    return None


def field_yaml_path(field_path):
    """Convert field path to YAML key path relative to resource instance.

    'resources.jobs.*.name' -> ['name']
    'resources.jobs.*.email_notifications.on_failure' -> ['email_notifications', 'on_failure']
    'resources.jobs.*.tasks[*].task_key' -> None  (array elements not supported for simple insertion)

    >>> field_yaml_path("resources.jobs.*.name")
    ['name']
    >>> field_yaml_path("resources.jobs.*.email_notifications.on_failure")
    ['email_notifications', 'on_failure']
    >>> field_yaml_path("resources.jobs.*.tasks[*].task_key") is None
    True
    """
    # Skip the "resources.<type>.*" prefix
    parts = field_path.split(".")
    if len(parts) < 4 or parts[0] != "resources" or parts[2] != "*":
        return None
    remainder = parts[3:]
    # Skip fields that involve array indexing or map wildcards
    for p in remainder:
        if "[*]" in p or p == "*":
            return None
    return remainder


def interesting_values(field_type):
    """Return interesting test values for a given field type.

    >>> interesting_values("bool")
    [True, False]
    >>> interesting_values("string")
    ['test-fuzz-value', '']
    >>> interesting_values("int")
    [0, 1, -1]
    """
    if field_type == "bool":
        return [True, False]
    if field_type in ("int", "int64"):
        return [0, 1, -1]
    if field_type == "float64":
        return [0.0, 1.5]
    if field_type == "string":
        return ["test-fuzz-value", ""]
    if field_type == "map[string]string":
        return [{"fuzz_key": "fuzz_value"}]
    if field_type.startswith("[]"):
        return [[]]
    if field_type.startswith("*"):
        # Pointer to struct — try empty struct
        return [{}]
    if field_type == "any":
        return ["test-fuzz-value"]
    # Enum or struct types — try empty string (will often fail validation, which is fine)
    if "." in field_type:
        return [""]
    return ["test-fuzz-value"]


def load_config_names_from_toml(toml_path):
    """Load INPUT_CONFIG list from test.toml EnvMatrix.

    >>> names = load_config_names_from_toml(TEST_TOML)
    >>> "job.yml.tmpl" in names
    True
    >>> len(names) > 10
    True
    """
    with open(toml_path, "rb") as f:
        data = tomllib.load(f)
    return data.get("EnvMatrix", {}).get("INPUT_CONFIG", [])


def load_exclude_rules(toml_path):
    """Load EnvMatrixExclude rules from test.toml.

    >>> rules = load_exclude_rules(TEST_TOML)
    >>> any("INPUT_CONFIG=alert.yml.tmpl" in r for r in rules.values())
    True
    """
    with open(toml_path, "rb") as f:
        data = tomllib.load(f)
    return data.get("EnvMatrixExclude", {})


def should_exclude_config(config_name, exclude_rules, is_cloud):
    """Check if a config should be excluded based on EnvMatrixExclude rules.

    Each rule is a list of conditions that must ALL match for the config to be excluded.
    CONFIG_Cloud=true matches only when is_cloud is True.

    >>> rules = {"no_alert": ["CONFIG_Cloud=true", "INPUT_CONFIG=alert.yml.tmpl"]}
    >>> should_exclude_config("alert.yml.tmpl", rules, is_cloud=True)
    True
    >>> should_exclude_config("alert.yml.tmpl", rules, is_cloud=False)
    False
    >>> should_exclude_config("job.yml.tmpl", rules, is_cloud=True)
    False
    """
    for conditions in exclude_rules.values():
        all_match = True
        for cond in conditions:
            key, value = cond.split("=", 1)
            if key == "CONFIG_Cloud":
                if not is_cloud:
                    all_match = False
                    break
            elif key == "INPUT_CONFIG":
                if value != config_name:
                    all_match = False
                    break
            # Other conditions (e.g., DATABRICKS_BUNDLE_ENGINE) are not relevant for config filtering
        if all_match:
            return True
    return False


def parse_config_resource_types(config_path):
    """Extract resource type keys from a config template.

    Returns set of resource type names (e.g., {'jobs', 'schemas'}).
    """
    resource_types = set()
    with open(config_path) as f:
        content = f.read()
    # Look for top-level keys under 'resources:' section
    in_resources = False
    for line in content.splitlines():
        stripped = line.strip()
        if stripped == "resources:":
            in_resources = True
            continue
        if in_resources:
            if line and not line[0].isspace():
                # Left-aligned line = new top-level section
                break
            # Check for resource type key (2-space indented under resources)
            m = re.match(r"^  (\w+):", line)
            if m:
                resource_types.add(m.group(1))
    return resource_types


def find_resource_instance(lines, resource_type):
    """Find the first resource instance under the given resource type section.

    Returns (instance_start, instance_indent, instance_end) or (None, None, None).

    >>> lines = "resources:\\n  jobs:\\n    foo:\\n      name: bar\\n".splitlines(keepends=True)
    >>> find_resource_instance(lines, "jobs")
    (2, 4, 4)
    >>> find_resource_instance(lines, "schemas")
    (None, None, None)
    """
    # First, find the resource type key at indent 2 (e.g., "  jobs:")
    type_line = None
    for i, line in enumerate(lines):
        content = line.strip()
        if not content or content.startswith("#"):
            continue
        indent = len(line) - len(line.lstrip())
        if indent == 2 and content == resource_type + ":":
            type_line = i
            break

    if type_line is None:
        return None, None, None

    # Find the first instance at indent 4 under this resource type
    instance_start = None
    instance_indent = None
    for i in range(type_line + 1, len(lines)):
        content = lines[i].strip()
        if not content or content.startswith("#"):
            continue
        indent = len(lines[i]) - len(lines[i].lstrip())
        if indent <= 2:
            break  # Left the resource type section
        if indent == 4 and ":" in content and not content.startswith("-"):
            instance_indent = indent
            instance_start = i
            break

    if instance_start is None or instance_indent is None:
        return None, None, None

    # Find the end of this instance block
    instance_end = len(lines)
    for i in range(instance_start + 1, len(lines)):
        content = lines[i].strip()
        if not content or content.startswith("#"):
            continue
        indent = len(lines[i]) - len(lines[i].lstrip())
        if indent <= instance_indent:
            instance_end = i
            break

    return instance_start, instance_indent, instance_end


def set_yaml_value(lines, key_path, value, resource_type=None):
    """Insert or modify a YAML value given a key path within a resource instance.

    This is a simple line-based YAML manipulation (not a full parser).
    Returns modified lines.

    >>> lines = "resources:\\n  jobs:\\n    foo:\\n      name: bar\\n".splitlines(keepends=True)
    >>> "".join(set_yaml_value(lines, ["description"], "hello", "jobs"))
    'resources:\\n  jobs:\\n    foo:\\n      name: bar\\n      description: hello\\n'

    Nested field with intermediate key creation:
    >>> lines = "resources:\\n  jobs:\\n    foo:\\n      name: bar\\n".splitlines(keepends=True)
    >>> "".join(set_yaml_value(lines, ["email_notifications", "on_failure"], [], "jobs"))
    'resources:\\n  jobs:\\n    foo:\\n      name: bar\\n      email_notifications:\\n        on_failure: []\\n'

    Replaces existing value:
    >>> lines = "resources:\\n  jobs:\\n    foo:\\n      name: bar\\n".splitlines(keepends=True)
    >>> "".join(set_yaml_value(lines, ["name"], "baz", "jobs"))
    'resources:\\n  jobs:\\n    foo:\\n      name: baz\\n'

    Sibling collision: setting email_notifications.on_failure when webhook_notifications.on_failure exists:
    >>> lines = "resources:\\n  jobs:\\n    foo:\\n      webhook_notifications:\\n        on_failure: old\\n      name: bar\\n".splitlines(keepends=True)
    >>> result = "".join(set_yaml_value(lines, ["email_notifications", "on_failure"], "new", "jobs"))
    >>> "webhook_notifications:" in result and "on_failure: old" in result
    True
    >>> "email_notifications:" in result and "on_failure: new" in result
    True
    """
    instance_start, instance_indent, instance_end = find_resource_instance(lines, resource_type)

    if instance_start is None or instance_indent is None:
        return lines

    # Navigate/create the key path within the instance
    current_indent = instance_indent + 2  # 6 spaces for first level under instance
    search_start = instance_start + 1
    search_end = instance_end
    insert_pos = instance_end

    for key in key_path[:-1]:
        # Look for existing key at current indent level
        found = False
        for i in range(search_start, search_end):
            content = lines[i].strip()
            if not content:
                continue
            indent = len(lines[i]) - len(lines[i].lstrip())
            if indent == current_indent and content.startswith(key + ":"):
                search_start = i + 1
                current_indent += 2
                # Narrow search_end to this key's block
                for j in range(i + 1, search_end):
                    s = lines[j].strip()
                    if not s:
                        continue
                    ind = len(lines[j]) - len(lines[j].lstrip())
                    if ind <= current_indent - 2:
                        search_end = j
                        break
                insert_pos = search_end
                found = True
                break
        if not found:
            # Insert intermediate key
            new_line = " " * current_indent + key + ":\n"
            lines.insert(insert_pos, new_line)
            instance_end += 1
            search_start = insert_pos + 1
            search_end = insert_pos + 1
            insert_pos += 1
            current_indent += 2

    # Insert or replace the final key
    final_key = key_path[-1]
    for i in range(search_start, search_end):
        content = lines[i].strip()
        if not content:
            continue
        indent = len(lines[i]) - len(lines[i].lstrip())
        if indent == current_indent and content.startswith(final_key + ":"):
            # Replace existing value
            lines[i] = " " * current_indent + final_key + ": " + yaml_encode(value) + "\n"
            return lines

    # Insert new key
    new_line = " " * current_indent + final_key + ": " + yaml_encode(value) + "\n"
    lines.insert(insert_pos, new_line)
    return lines


def remove_yaml_key(lines, key_path, resource_type=None):
    """Remove a YAML key from the resource instance. Returns (modified_lines, was_removed).

    >>> lines = "resources:\\n  jobs:\\n    foo:\\n      name: bar\\n      desc: x\\n".splitlines(keepends=True)
    >>> result, removed = remove_yaml_key(lines, ["name"], "jobs")
    >>> removed
    True
    >>> "".join(result)
    'resources:\\n  jobs:\\n    foo:\\n      desc: x\\n'

    Remove a key with children:
    >>> lines = "resources:\\n  jobs:\\n    foo:\\n      email_notifications:\\n        on_failure: a\\n        on_success: b\\n      name: bar\\n".splitlines(keepends=True)
    >>> result, removed = remove_yaml_key(lines, ["email_notifications"], "jobs")
    >>> removed
    True
    >>> "".join(result)
    'resources:\\n  jobs:\\n    foo:\\n      name: bar\\n'

    Key not found:
    >>> lines = "resources:\\n  jobs:\\n    foo:\\n      name: bar\\n".splitlines(keepends=True)
    >>> _, removed = remove_yaml_key(lines, ["missing"], "jobs")
    >>> removed
    False
    """
    instance_start, instance_indent, instance_end = find_resource_instance(lines, resource_type)

    if instance_start is None or instance_indent is None:
        return lines, False

    current_indent = instance_indent + 2
    search_start = instance_start + 1
    search_end = instance_end

    # Navigate to parent keys
    for key in key_path[:-1]:
        found = False
        for i in range(search_start, search_end):
            content = lines[i].strip()
            if not content:
                continue
            indent = len(lines[i]) - len(lines[i].lstrip())
            if indent == current_indent and content.startswith(key + ":"):
                search_start = i + 1
                current_indent += 2
                # Find end of this key's block
                for j in range(i + 1, search_end):
                    s = lines[j].strip()
                    if not s:
                        continue
                    ind = len(lines[j]) - len(lines[j].lstrip())
                    if ind <= current_indent - 2:
                        search_end = j
                        break
                found = True
                break
        if not found:
            return lines, False

    # Find and remove the final key
    final_key = key_path[-1]
    for i in range(search_start, search_end):
        content = lines[i].strip()
        if not content:
            continue
        indent = len(lines[i]) - len(lines[i].lstrip())
        if indent == current_indent and content.startswith(final_key + ":"):
            # Remove this line and any indented children
            remove_end = i + 1
            for j in range(i + 1, search_end):
                s = lines[j].rstrip()
                if not s:
                    remove_end = j + 1
                    continue
                ind = len(lines[j]) - len(lines[j].lstrip())
                if ind > current_indent:
                    remove_end = j + 1
                else:
                    break
            del lines[i:remove_end]
            return lines, True

    return lines, False


def yaml_encode(value):
    """Encode a Python value as YAML scalar.

    >>> yaml_encode(True)
    'true'
    >>> yaml_encode(False)
    'false'
    >>> yaml_encode(0)
    '0'
    >>> yaml_encode("hello")
    'hello'
    >>> yaml_encode("")
    '""'
    >>> yaml_encode({})
    '{}'
    >>> yaml_encode([])
    '[]'
    """
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, (int, float)):
        return str(value)
    if isinstance(value, str):
        if value == "":
            return '""'
        # Quote if contains special chars
        if any(c in value for c in ":#{}[]&*!|>',@`\""):
            return json.dumps(value)
        return value
    if isinstance(value, dict):
        if not value:
            return "{}"
        # Simple inline dict
        items = ", ".join(f"{k}: {yaml_encode(v)}" for k, v in value.items())
        return "{" + items + "}"
    if isinstance(value, list):
        if not value:
            return "[]"
        items = ", ".join(yaml_encode(v) for v in value)
        return "[" + items + "]"
    return str(value)


def build_test_cases(fields, configs, config_filter, field_filter):
    """Build list of (config_name, field_path, field_type, value, action) test cases."""
    cases = []

    for config_name in sorted(configs.keys()):
        if config_filter and config_filter not in config_name:
            continue

        resource_types = configs[config_name]["resource_types"]
        config_content = configs[config_name]["content"]

        for field_path, field_type, tags in fields:
            if not is_input_field(tags):
                continue
            if field_filter and field_filter not in field_path:
                continue

            res_type = extract_resource_type(field_path)
            if res_type not in resource_types:
                continue

            yaml_path = field_yaml_path(field_path)
            if yaml_path is None:
                continue

            # Skip internal/meta fields that don't make sense to fuzz
            if yaml_path == ["id"] or yaml_path == ["modified_status"]:
                continue
            if yaml_path[0] == "lifecycle":
                continue

            # Check if field is already present in config
            field_present = is_field_in_config(config_content, yaml_path, resource_type=res_type)

            # Generate "add" test cases with interesting values
            for value in interesting_values(field_type):
                cases.append((config_name, field_path, field_type, value, "set"))

            # Generate "remove" test case if field is present
            if field_present:
                cases.append((config_name, field_path, field_type, None, "remove"))

    return cases


def is_field_in_config(content, yaml_path, resource_type=None):
    """Check if a field key path exists under the correct resource instance.

    Uses find_resource_instance + indent-level walking to verify the full path,
    not just the leaf key.

    >>> is_field_in_config(
    ...     "bundle:\\n  name: x\\nresources:\\n  jobs:\\n    foo:\\n      name: bar\\n",
    ...     ["name"], resource_type="jobs")
    True
    >>> is_field_in_config(
    ...     "bundle:\\n  name: x\\nresources:\\n  jobs:\\n    foo:\\n      name: bar\\n"
    ...     "      permissions:\\n        - level: CAN_VIEW\\n          group_name: users\\n",
    ...     ["run_as", "group_name"], resource_type="jobs")
    False
    >>> is_field_in_config(
    ...     "bundle:\\n  name: x\\nresources:\\n  jobs:\\n    foo:\\n      name: bar\\n"
    ...     "      run_as:\\n        group_name: admins\\n",
    ...     ["run_as", "group_name"], resource_type="jobs")
    True
    """
    lines = content.splitlines(keepends=True)
    instance_start, instance_indent, instance_end = find_resource_instance(lines, resource_type)
    if instance_start is None:
        return False

    current_indent = instance_indent + 2
    search_start = instance_start + 1
    search_end = instance_end

    for key in yaml_path:
        found = False
        for i in range(search_start, search_end):
            line = lines[i]
            stripped = line.strip()
            if not stripped or stripped.startswith("#"):
                continue
            indent = len(line) - len(line.lstrip())
            if indent == current_indent and stripped.startswith(key + ":"):
                # Found this key; narrow search to its children
                search_start = i + 1
                current_indent += 2
                # Find end of this key's block
                for j in range(i + 1, search_end):
                    s = lines[j].strip()
                    if not s:
                        continue
                    ind = len(lines[j]) - len(lines[j].lstrip())
                    if ind <= current_indent - 2:
                        search_end = j
                        break
                found = True
                break
            if indent < current_indent and indent <= instance_indent:
                break
        if not found:
            return False
    return True


def generate_modified_config(config_content, yaml_path, value, action, resource_type=None):
    """Generate a modified config with the field set or removed."""
    lines = config_content.splitlines(keepends=True)
    if not lines[-1].endswith("\n"):
        lines[-1] += "\n"

    if action == "remove":
        lines, removed = remove_yaml_key(lines, yaml_path, resource_type=resource_type)
        if not removed:
            return None
    else:
        lines = set_yaml_value(lines, yaml_path, value, resource_type=resource_type)

    return "".join(lines)


def run_test_via_harness(config_name, config_content, test_name="no_drift", verbose=False):
    """Run an invariant test against a modified config using the acceptance test harness.

    Overwrites the config in configs/ temporarily, invokes `go test` targeting the
    specific invariant test variant, then restores the original config.

    Returns: (status, details) where status is one of:
        "skip" — config failed validation or deploy
        "pass" — no drift detected
        "drift" — drift detected (bug found!)
        "error" — unexpected error
    """
    config_path = CONFIGS_DIR / config_name
    original_content = config_path.read_text()

    try:
        config_path.write_text(config_content)

        escaped_name = re.escape(config_name)
        run_filter = (
            f"TestAccept/bundle/invariant/{test_name}/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG={escaped_name}"
        )
        cmd = [
            "go",
            "test",
            "./acceptance",
            "-run",
            run_filter,
            "-v",
            "-tail",
            "-count=1",
            "-timeout",
            "10m",
        ]

        result = subprocess.run(
            cmd,
            cwd=str(REPO_ROOT),
            capture_output=True,
            text=True,
            timeout=700,
        )

        output = result.stdout + result.stderr
        has_config_ok = "INPUT_CONFIG_OK" in output

        if verbose:
            # Print last 20 lines of output for debugging
            lines = output.strip().splitlines()
            for line in lines[-20:]:
                print(f"    | {line}", file=sys.stderr)

        if result.returncode == 0 and has_config_ok:
            return "pass", "no drift"

        if not has_config_ok:
            return "skip", "config rejected (validation/deploy failed)"

        # Test failed after INPUT_CONFIG_OK — bug detected
        if "panic" in output.lower():
            return "error", "panic after deploy"
        if "Unexpected action=" in output:
            return "drift", "drift detected"
        return "error", f"test failed (rc={result.returncode})"

    except subprocess.TimeoutExpired:
        return "error", "test timed out"
    finally:
        config_path.write_text(original_content)


def save_generated_config(config_name, field_path, index, config_content):
    """Save a generated config that caused drift."""
    safe_field = field_path.replace(".", "_").replace("*", "x").replace("[", "").replace("]", "")
    base = config_name.replace(".yml.tmpl", "")
    filename = f"{GENERATED_PREFIX}{base}_{safe_field}.{index:03d}.yml"
    dest = CONFIGS_DIR / filename
    dest.write_text(config_content)
    return dest


def main():
    parser = argparse.ArgumentParser(description="Fuzz-test bundle fields for drift")
    parser.add_argument("--config", default="", help="Substring filter for config names")
    parser.add_argument("--field", default="", help="Substring filter for field names")
    parser.add_argument("--seed", type=int, default=None, help="Random seed")
    parser.add_argument("-n", type=int, default=None, help="Max test cases to run")
    parser.add_argument("--test", default="no_drift", help="Invariant test to run (default: no_drift)")
    parser.add_argument("--verbose", "-v", action="store_true", help="Verbose output")
    parser.add_argument("--dry-run", action="store_true", help="List test cases without running")
    args = parser.parse_args()

    # Parse fields
    if not FIELDS_FILE.exists():
        print(f"Fields file not found: {FIELDS_FILE}", file=sys.stderr)
        print("Run the refschema acceptance test first to generate it.", file=sys.stderr)
        sys.exit(1)

    fields = parse_fields(FIELDS_FILE)
    print(f"Loaded {len(fields)} fields from {FIELDS_FILE.name}")

    # Parse configs from test.toml EnvMatrix, applying exclusion rules
    config_names = load_config_names_from_toml(TEST_TOML)
    exclude_rules = load_exclude_rules(TEST_TOML)
    is_cloud = bool(os.environ.get("CLOUD_ENV"))

    configs = {}
    for name in sorted(config_names):
        if name in SKIP_CONFIGS:
            continue
        if should_exclude_config(name, exclude_rules, is_cloud):
            continue
        config_path = CONFIGS_DIR / name
        if not config_path.exists():
            print(f"Warning: config {name} from test.toml not found, skipping", file=sys.stderr)
            continue
        resource_types = parse_config_resource_types(config_path)
        configs[name] = {
            "resource_types": resource_types,
            "content": config_path.read_text(),
        }
    print(f"Loaded {len(configs)} configs from {TEST_TOML.name}")

    # Build test cases
    cases = build_test_cases(fields, configs, args.config, args.field)
    print(f"Generated {len(cases)} test cases")

    if not cases:
        print("No matching test cases found.")
        return

    # Shuffle
    if args.seed is not None:
        random.seed(args.seed)
    else:
        seed = random.randint(0, 2**32)
        random.seed(seed)
        print(f"Using seed: {seed}")

    random.shuffle(cases)

    # Limit
    n = len(cases)
    if args.n is not None:
        n = min(args.n, n)
    print(f"Running {n} of {len(cases)} test cases\n")

    if args.dry_run:
        for i, (config_name, field_path, field_type, value, action) in enumerate(cases[:n]):
            print(f"{i + 1:4d}. {config_name} | {field_path} ({field_type}) | {action}={value}")
        return

    # Run tests
    stats = {"skip": 0, "pass": 0, "drift": 0, "error": 0}
    saved_configs = []
    gen_index = 0

    for i, (config_name, field_path, field_type, value, action) in enumerate(cases[:n]):
        label = f"[{i + 1}/{n}] {config_name} | {field_path} | {action}"
        if action == "set":
            label += f"={yaml_encode(value)}"
        print(f"{label} ... ", end="", flush=True)

        # Generate modified config
        config_content = configs[config_name]["content"]
        yaml_path = field_yaml_path(field_path)
        res_type = extract_resource_type(field_path)
        modified = generate_modified_config(config_content, yaml_path, value, action, resource_type=res_type)
        if modified is None:
            print("SKIP (modification failed)")
            stats["skip"] += 1
            continue

        status, details = run_test_via_harness(config_name, modified, test_name=args.test, verbose=args.verbose)
        stats[status] += 1
        print(f"{status.upper()}: {details}")

        if status in ("drift", "error"):
            gen_index += 1
            saved = save_generated_config(config_name, field_path, gen_index, modified)
            saved_configs.append((config_name, field_path, value, action, status, saved))
            print(f"    -> Saved to {saved.relative_to(REPO_ROOT)}")

    # Summary
    print(f"\n{'=' * 60}")
    print(f"Results: {stats['pass']} pass, {stats['drift']} drift, {stats['skip']} skip, {stats['error']} error")

    if saved_configs:
        print(f"\nIssues detected in {len(saved_configs)} cases:")
        for config_name, field_path, value, action, status, saved in saved_configs:
            print(f"  - [{status}] {config_name} | {field_path} | {action}={value}")
            print(f"    Config: {saved.relative_to(REPO_ROOT)}")


if __name__ == "__main__":
    main()
