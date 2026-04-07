#!/usr/bin/env python3
"""
Fuzz-test bundle field handling by iterating over all INPUT fields from refschema,
generating modified configs, and running the no_drift test against each.

Usage:
    python3 tools/fuzz/test_fields.py --config job --field description -n 5
    python3 tools/fuzz/test_fields.py --seed 42 -n 100
"""

import argparse
import json
import os
import random
import re
import shutil
import subprocess
import sys
import tempfile
import uuid
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent.parent
FIELDS_FILE = REPO_ROOT / "acceptance" / "bundle" / "refschema" / "out.fields.txt"
CONFIGS_DIR = REPO_ROOT / "acceptance" / "bundle" / "invariant" / "configs"
DATA_DIR = REPO_ROOT / "acceptance" / "bundle" / "invariant" / "data"
GENERATED_PREFIX = "generated__"

# Configs that need special init/cleanup scripts or external setup — skip in fuzzer.
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
    # Skip fields that involve array indexing in the middle
    for p in remainder:
        if "[*]" in p:
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


def substitute_env_vars(text):
    """Substitute $VAR and ${VAR} patterns with environment variables.

    >>> os.environ['_TEST_VAR'] = 'hello'
    >>> substitute_env_vars('$_TEST_VAR world')
    'hello world'
    """

    def replace_var(match):
        var_name = match.group(1) or match.group(2)
        return os.environ.get(var_name, "")

    return re.sub(r"\$\{([A-Za-z_][A-Za-z0-9_]*)\}|\$([A-Za-z_][A-Za-z0-9_]*)", replace_var, text)


def set_yaml_value(lines, key_path, value):
    """Insert or modify a YAML value given a key path within a resource instance.

    This is a simple line-based YAML manipulation (not a full parser).
    Returns modified lines.
    """
    # Find the resource instance block (indented under resource type)
    # We look for the first resource instance (4-space indent under resource type)
    instance_indent = None
    instance_start = None
    for i, line in enumerate(lines):
        stripped = line.rstrip()
        if not stripped or stripped.startswith("#"):
            continue
        indent = len(line) - len(line.lstrip())
        # Resource instances are at 4-space indent (under "  <resource_type>:")
        if indent == 4 and ":" in stripped and not stripped.startswith("-"):
            instance_indent = indent
            instance_start = i
            break

    if instance_start is None or instance_indent is None:
        return lines

    # Find the end of this instance block
    instance_end = len(lines)
    for i in range(instance_start + 1, len(lines)):
        stripped = lines[i].rstrip()
        if not stripped or stripped.startswith("#"):
            continue
        indent = len(lines[i]) - len(lines[i].lstrip())
        if indent <= instance_indent:
            instance_end = i
            break

    # Navigate/create the key path within the instance
    current_indent = instance_indent + 2  # 6 spaces for first level under instance
    insert_pos = instance_end

    for key in key_path[:-1]:
        # Look for existing key at current indent level
        found = False
        for i in range(instance_start + 1, instance_end):
            stripped = lines[i].rstrip()
            if not stripped:
                continue
            indent = len(lines[i]) - len(lines[i].lstrip())
            if indent == current_indent and stripped.startswith(key + ":"):
                insert_pos = i + 1
                current_indent += 2
                found = True
                break
        if not found:
            # Insert intermediate key
            new_line = " " * current_indent + key + ":\n"
            lines.insert(insert_pos, new_line)
            instance_end += 1
            insert_pos += 1
            current_indent += 2

    # Insert or replace the final key
    final_key = key_path[-1]
    for i in range(instance_start + 1, instance_end):
        stripped = lines[i].rstrip()
        if not stripped:
            continue
        indent = len(lines[i]) - len(lines[i].lstrip())
        if indent == current_indent and stripped.startswith(final_key + ":"):
            # Replace existing value
            lines[i] = " " * current_indent + final_key + ": " + yaml_encode(value) + "\n"
            return lines

    # Insert new key
    new_line = " " * current_indent + final_key + ": " + yaml_encode(value) + "\n"
    lines.insert(insert_pos, new_line)
    return lines


def remove_yaml_key(lines, key_path):
    """Remove a YAML key from the resource instance. Returns (modified_lines, was_removed)."""
    instance_indent = None
    instance_start = None
    for i, line in enumerate(lines):
        stripped = line.rstrip()
        if not stripped or stripped.startswith("#"):
            continue
        indent = len(line) - len(line.lstrip())
        if indent == 4 and ":" in stripped and not stripped.startswith("-"):
            instance_indent = indent
            instance_start = i
            break

    if instance_start is None or instance_indent is None:
        return lines, False

    instance_end = len(lines)
    for i in range(instance_start + 1, len(lines)):
        stripped = lines[i].rstrip()
        if not stripped or stripped.startswith("#"):
            continue
        indent = len(lines[i]) - len(lines[i].lstrip())
        if indent <= instance_indent:
            instance_end = i
            break

    current_indent = instance_indent + 2
    search_start = instance_start + 1
    search_end = instance_end

    # Navigate to parent keys
    for key in key_path[:-1]:
        found = False
        for i in range(search_start, search_end):
            stripped = lines[i].rstrip()
            if not stripped:
                continue
            indent = len(lines[i]) - len(lines[i].lstrip())
            if indent == current_indent and stripped.startswith(key + ":"):
                search_start = i + 1
                current_indent += 2
                # Find end of this key's block
                for j in range(i + 1, search_end):
                    s = lines[j].rstrip()
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
        stripped = lines[i].rstrip()
        if not stripped:
            continue
        indent = len(lines[i]) - len(lines[i].lstrip())
        if indent == current_indent and stripped.startswith(final_key + ":"):
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
        if config_name in SKIP_CONFIGS:
            continue
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
            field_present = is_field_in_config(config_content, yaml_path)

            # Generate "add" test cases with interesting values
            for value in interesting_values(field_type):
                cases.append((config_name, field_path, field_type, value, "set"))

            # Generate "remove" test case if field is present
            if field_present:
                cases.append((config_name, field_path, field_type, None, "remove"))

    return cases


def is_field_in_config(content, yaml_path):
    """Check if a field path exists in the config content."""
    final_key = yaml_path[-1]
    for line in content.splitlines():
        stripped = line.strip()
        if stripped.startswith(final_key + ":"):
            return True
    return False


def generate_modified_config(config_content, yaml_path, value, action):
    """Generate a modified config with the field set or removed."""
    lines = config_content.splitlines(keepends=True)
    if not lines[-1].endswith("\n"):
        lines[-1] += "\n"

    if action == "remove":
        lines, removed = remove_yaml_key(lines, yaml_path)
        if not removed:
            return None
    else:
        lines = set_yaml_value(lines, yaml_path, value)

    return "".join(lines)


def run_cli(cli_path, args, cwd, env, timeout=300):
    """Run a CLI command and return (returncode, stdout, stderr)."""
    cmd = [str(cli_path)] + args
    try:
        result = subprocess.run(
            cmd,
            cwd=cwd,
            env=env,
            capture_output=True,
            text=True,
            timeout=timeout,
        )
        return result.returncode, result.stdout, result.stderr
    except subprocess.TimeoutExpired:
        return -1, "", "timeout"


def run_test_case(cli_path, config_content, unique_name, verbose=False):
    """Run the no_drift test against a single config.

    Returns: (status, details) where status is one of:
        "skip" — config failed validation or deploy
        "pass" — no drift detected
        "drift" — drift detected (bug found!)
        "error" — unexpected error
    """
    tmpdir = tempfile.mkdtemp(prefix="fuzz_field_")

    try:
        # Copy data files
        if DATA_DIR.exists():
            for item in DATA_DIR.iterdir():
                dest = Path(tmpdir) / item.name
                if item.is_dir():
                    shutil.copytree(item, dest)
                else:
                    shutil.copy2(item, dest)

        # Write config with env var substitution
        os.environ["UNIQUE_NAME"] = unique_name
        config_resolved = substitute_env_vars(config_content)
        config_path = Path(tmpdir) / "databricks.yml"
        config_path.write_text(config_resolved)

        # Initialize git repo (required by bundle)
        subprocess.run(
            ["git", "init", "-qb", "main"],
            cwd=tmpdir,
            capture_output=True,
        )
        subprocess.run(
            ["git", "config", "user.name", "Fuzzer"],
            cwd=tmpdir,
            capture_output=True,
        )
        subprocess.run(
            ["git", "config", "user.email", "fuzzer@test.com"],
            cwd=tmpdir,
            capture_output=True,
        )
        subprocess.run(
            ["git", "add", "."],
            cwd=tmpdir,
            capture_output=True,
        )
        subprocess.run(
            ["git", "commit", "-qm", "init"],
            cwd=tmpdir,
            capture_output=True,
        )

        env = dict(os.environ)
        env["DATABRICKS_BUNDLE_ENGINE"] = "direct"

        # Validate
        rc, stdout, stderr = run_cli(cli_path, ["bundle", "validate"], tmpdir, env)
        if rc != 0:
            if verbose:
                print(f"    validate failed: {stderr[:200]}", file=sys.stderr)
            return "skip", f"validate failed (rc={rc})"

        # Check for panics/internal errors
        if "panic" in stderr.lower() or "internal error" in stderr.lower():
            return "error", f"validate: panic/internal error: {stderr[:200]}"

        # Deploy
        rc, stdout, stderr = run_cli(cli_path, ["bundle", "deploy"], tmpdir, env, timeout=600)
        if rc != 0:
            if verbose:
                print(f"    deploy failed: {stderr[:200]}", file=sys.stderr)
            return "skip", f"deploy failed (rc={rc})"

        if "panic" in stderr.lower() or "internal error" in stderr.lower():
            return "error", f"deploy: panic/internal error: {stderr[:200]}"

        # Plan (JSON)
        rc, stdout, stderr = run_cli(cli_path, ["bundle", "plan", "-o", "json"], tmpdir, env)
        if "panic" in stderr.lower() or "internal error" in stderr.lower():
            return "error", f"plan: panic/internal error: {stderr[:200]}"

        if rc != 0:
            return "error", f"plan failed (rc={rc}): {stderr[:200]}"

        # Check plan for drift
        try:
            plan_data = json.loads(stdout)
            drift_actions = {}
            for key, value in plan_data.get("plan", {}).items():
                action = value.get("action")
                if action != "skip":
                    drift_actions[key] = action
            if drift_actions:
                return "drift", f"drift detected: {drift_actions}"
        except (json.JSONDecodeError, KeyError) as e:
            return "error", f"plan parse error: {e}"

        return "pass", "no drift"

    finally:
        # Destroy (cleanup)
        env = dict(os.environ)
        env["DATABRICKS_BUNDLE_ENGINE"] = "direct"
        run_cli(cli_path, ["bundle", "destroy", "--auto-approve"], tmpdir, env, timeout=300)
        shutil.rmtree(tmpdir, ignore_errors=True)


def generate_unique_name():
    """Generate a unique name similar to acceptance test framework."""
    return "fuzz-" + uuid.uuid4().hex[:20]


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
    parser.add_argument("--cli", default=None, help="Path to CLI binary")
    parser.add_argument("--verbose", "-v", action="store_true", help="Verbose output")
    parser.add_argument("--dry-run", action="store_true", help="List test cases without running")
    args = parser.parse_args()

    # Find CLI binary
    cli_path = args.cli
    if cli_path is None:
        cli_path = REPO_ROOT / "cli"
        if not cli_path.exists():
            # Try common build locations
            for p in [
                REPO_ROOT / "build" / "databricks",
                REPO_ROOT / "databricks",
            ]:
                if p.exists():
                    cli_path = p
                    break
    cli_path = Path(cli_path)
    if not args.dry_run and not cli_path.exists():
        print(f"CLI binary not found at {cli_path}. Run 'make build' first or pass --cli.", file=sys.stderr)
        sys.exit(1)

    # Parse fields
    if not FIELDS_FILE.exists():
        print(f"Fields file not found: {FIELDS_FILE}", file=sys.stderr)
        print("Run the refschema acceptance test first to generate it.", file=sys.stderr)
        sys.exit(1)

    fields = parse_fields(FIELDS_FILE)
    print(f"Loaded {len(fields)} fields from {FIELDS_FILE.name}")

    # Parse configs
    configs = {}
    for config_path in sorted(CONFIGS_DIR.glob("*.yml.tmpl")):
        if config_path.name.startswith(GENERATED_PREFIX):
            continue
        resource_types = parse_config_resource_types(config_path)
        configs[config_path.name] = {
            "resource_types": resource_types,
            "content": config_path.read_text(),
        }
    print(f"Loaded {len(configs)} configs from {CONFIGS_DIR.name}")

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
    drift_configs = []
    gen_index = 0

    for i, (config_name, field_path, field_type, value, action) in enumerate(cases[:n]):
        unique_name = generate_unique_name()
        label = f"[{i + 1}/{n}] {config_name} | {field_path} | {action}"
        if action == "set":
            label += f"={yaml_encode(value)}"
        print(f"{label} ... ", end="", flush=True)

        # Generate modified config
        config_content = configs[config_name]["content"]
        yaml_path = field_yaml_path(field_path)
        modified = generate_modified_config(config_content, yaml_path, value, action)
        if modified is None:
            print("SKIP (modification failed)")
            stats["skip"] += 1
            continue

        status, details = run_test_case(cli_path, modified, unique_name, verbose=args.verbose)
        stats[status] += 1
        print(f"{status.upper()}: {details}")

        if status == "drift":
            gen_index += 1
            saved = save_generated_config(config_name, field_path, gen_index, modified)
            drift_configs.append((config_name, field_path, value, action, saved))
            print(f"    -> Saved to {saved.relative_to(REPO_ROOT)}")

    # Summary
    print(f"\n{'=' * 60}")
    print(f"Results: {stats['pass']} pass, {stats['drift']} drift, {stats['skip']} skip, {stats['error']} error")

    if drift_configs:
        print(f"\nDrift detected in {len(drift_configs)} cases:")
        for config_name, field_path, value, action, saved in drift_configs:
            print(f"  - {config_name} | {field_path} | {action}={value}")
            print(f"    Config: {saved.relative_to(REPO_ROOT)}")


if __name__ == "__main__":
    main()
