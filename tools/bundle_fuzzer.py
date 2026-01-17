#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.13"
# dependencies = ["pyyaml"]
# ///
"""
Bundle Fuzzer - generates random databricks.yml configs and tests them.

Usage:
    bundle_fuzzer.py [OPTIONS] SCRIPT

Return codes from SCRIPT:
    0: No issues
    1-9: Syntax error, generate new config
    10-19: Bug detected, store results
"""

import argparse
import json
import os
import random
import subprocess
import sys
import tempfile
from collections import Counter
from pathlib import Path

import yaml

REPO_ROOT = Path(__file__).parent.parent.resolve()
SCHEMA_PATH = REPO_ROOT / "bundle" / "schema" / "jsonschema.json"
TESTSERVER_PATH = REPO_ROOT / "testserver"

# Value pools for generating realistic values
STRINGS = ["test", "my_job", "my_pipeline", "task1", "key1", "default", "dev", "prod"]
SPARK_VERSIONS = ["13.3.x-scala2.12", "14.3.x-scala2.12", "15.4.x-scala2.12"]
NODE_TYPES = ["i3.xlarge", "i3.2xlarge", "Standard_DS3_v2", "n1-standard-4"]
PYTHON_FILES = ["./main.py", "./src/task.py", "./notebook.py"]
WAREHOUSE_IDS = ["abc123def456", "warehouse_001"]
CATALOG_NAMES = ["main", "dev_catalog", "hive_metastore"]
SCHEMA_NAMES = ["default", "my_schema", "dev"]


class SchemaResolver:
    def __init__(self, schema: dict):
        self.schema = schema
        self.defs = schema.get("$defs", {})

    def resolve_ref(self, ref: str) -> dict:
        """Resolve a $ref pointer to its definition."""
        if not ref.startswith("#/$defs/"):
            return {}
        path = ref[8:]  # Remove "#/$defs/"
        parts = path.split("/")
        current = self.defs
        for part in parts:
            if isinstance(current, dict) and part in current:
                current = current[part]
            else:
                return {}
        return current if isinstance(current, dict) else {}

    def get_effective_schema(self, schema: dict) -> dict:
        """Get effective schema, resolving $ref if present."""
        if "$ref" in schema:
            return self.resolve_ref(schema["$ref"])
        return schema


class ConfigGenerator:
    def __init__(self, resolver: SchemaResolver, seed: int | None = None, max_fields: int = 10):
        self.resolver = resolver
        self.rng = random.Random(seed)
        self.depth = 0
        self.max_depth = 6
        self.max_fields = max_fields
        self.field_count = 0

    def generate_string(self, schema: dict) -> str:
        """Generate a random string value."""
        if "enum" in schema:
            return self.rng.choice(schema["enum"])
        # 10% chance of empty string
        if self.rng.random() < 0.1:
            return ""
        return self.rng.choice(STRINGS)

    def generate_int(self, schema: dict) -> int:
        """Generate a random integer value."""
        # 10% chance of zero
        if self.rng.random() < 0.1:
            return 0
        return self.rng.randint(1, 100)

    def generate_float(self, schema: dict) -> float:
        """Generate a random float value."""
        if self.rng.random() < 0.1:
            return 0.0
        return round(self.rng.uniform(0.1, 100.0), 2)

    def generate_bool(self, schema: dict) -> bool:
        """Generate a random boolean value."""
        return self.rng.choice([True, False])

    def generate_array(self, schema: dict) -> list:
        """Generate a random array."""
        items_schema = schema.get("items", {})
        if not items_schema:
            return []
        # Generate 1-3 items
        count = self.rng.randint(1, 3)
        return [self.generate_value(items_schema) for _ in range(count)]

    def generate_object(self, schema: dict) -> dict:
        """Generate a random object based on schema properties."""
        if self.depth >= self.max_depth or self.field_count >= self.max_fields:
            return {}

        self.depth += 1
        result = {}
        properties = schema.get("properties", {})
        required = set(schema.get("required", []))

        # Include required fields with 90% probability
        for prop_name in required:
            if self.field_count >= self.max_fields:
                break
            if prop_name in properties and self.rng.random() < 0.9:
                prop_schema = properties[prop_name]
                if not prop_schema.get("deprecated"):
                    value = self.generate_value(prop_schema)
                    if value is not None:
                        result[prop_name] = value
                        self.field_count += 1

        # Randomly select optional fields until budget exhausted
        optional = [k for k in properties if k not in required and not properties[k].get("deprecated")]
        self.rng.shuffle(optional)
        for prop_name in optional:
            if self.field_count >= self.max_fields:
                break
            value = self.generate_value(properties[prop_name])
            if value is not None:
                result[prop_name] = value
                self.field_count += 1

        self.depth -= 1
        return result

    def generate_value(self, schema: dict):
        """Generate a value based on schema type."""
        schema = self.resolver.get_effective_schema(schema)
        if not schema:
            return None

        # Handle oneOf - pick the first non-variable-reference option
        if "oneOf" in schema:
            for option in schema["oneOf"]:
                # Skip variable reference patterns
                if option.get("type") == "string" and "pattern" in option:
                    if "${" in option.get("pattern", ""):
                        continue
                return self.generate_value(option)
            return None

        schema_type = schema.get("type")

        if schema_type == "string":
            return self.generate_string(schema)
        elif schema_type == "integer":
            return self.generate_int(schema)
        elif schema_type == "number":
            return self.generate_float(schema)
        elif schema_type == "boolean":
            return self.generate_bool(schema)
        elif schema_type == "array":
            return self.generate_array(schema)
        elif schema_type == "object":
            return self.generate_object(schema)

        return None

    def generate_job(self) -> dict:
        """Generate a random job resource."""
        job_ref = "#/$defs/github.com/databricks/cli/bundle/config/resources.Job"
        job_schema = self.resolver.resolve_ref(job_ref)
        if not job_schema:
            return self._generate_minimal_job()

        job = self.generate_value(job_schema)
        if not job:
            job = {}

        # Ensure minimal required fields
        if "name" not in job:
            job["name"] = f"fuzz_job_{self.rng.randint(1000, 9999)}"

        return job

    def _generate_minimal_job(self) -> dict:
        """Generate a minimal valid job when schema resolution fails."""
        return {
            "name": f"fuzz_job_{self.rng.randint(1000, 9999)}",
            "tasks": [
                {
                    "task_key": "main_task",
                    "spark_python_task": {"python_file": self.rng.choice(PYTHON_FILES)},
                    "new_cluster": {
                        "spark_version": self.rng.choice(SPARK_VERSIONS),
                        "num_workers": self.rng.randint(1, 4),
                    },
                }
            ],
        }

    def generate_pipeline(self) -> dict:
        """Generate a random pipeline resource."""
        pipeline_ref = "#/$defs/github.com/databricks/cli/bundle/config/resources.Pipeline"
        pipeline_schema = self.resolver.resolve_ref(pipeline_ref)
        if not pipeline_schema:
            return self._generate_minimal_pipeline()

        pipeline = self.generate_value(pipeline_schema)
        if not pipeline:
            pipeline = {}

        if "name" not in pipeline:
            pipeline["name"] = f"fuzz_pipeline_{self.rng.randint(1000, 9999)}"

        return pipeline

    def _generate_minimal_pipeline(self) -> dict:
        """Generate a minimal valid pipeline."""
        return {
            "name": f"fuzz_pipeline_{self.rng.randint(1000, 9999)}",
            "libraries": [{"notebook": {"path": "./notebook"}}],
        }

    def generate_config(self, resource_types: list[str] | None = None) -> dict:
        """Generate a complete databricks.yml config."""
        if resource_types is None:
            resource_types = ["jobs"]

        config = {
            "bundle": {"name": f"fuzz_bundle_{self.rng.randint(10000, 99999)}"},
            "resources": {},
        }

        for rtype in resource_types:
            if rtype == "jobs":
                job_name = f"job_{self.rng.randint(100, 999)}"
                config["resources"]["jobs"] = {job_name: self.generate_job()}
            elif rtype == "pipelines":
                pipeline_name = f"pipeline_{self.rng.randint(100, 999)}"
                config["resources"]["pipelines"] = {pipeline_name: self.generate_pipeline()}

        return config


def get_cli_path() -> Path:
    """Get path to databricks CLI binary."""
    cli_env = os.environ.get("CLI")
    if cli_env:
        return Path(cli_env)

    # Build and use local binary
    print("CLI env var not set, running 'make build'...", file=sys.stderr)
    result = subprocess.run(["make", "build"], cwd=REPO_ROOT, capture_output=True)
    if result.returncode != 0:
        print(f"make build failed: {result.stderr.decode()}", file=sys.stderr)
        sys.exit(1)

    return REPO_ROOT / "cli"


class TestServer:
    """Manages a testserver process."""

    def __init__(self):
        self.proc = None
        self.url = None

    def start(self):
        """Start the testserver."""
        self.proc = subprocess.Popen(
            [str(TESTSERVER_PATH)],
            stdout=subprocess.PIPE,
            stderr=None,  # Let stderr go to console for debugging
            text=True,
        )
        # First line of stdout is the URL
        self.url = self.proc.stdout.readline().strip()

    def stop(self):
        """Stop the testserver."""
        if self.proc:
            self.proc.terminate()
            try:
                self.proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                self.proc.kill()
            self.proc = None
            self.url = None


def run_iteration(
    cli_path: Path,
    script_path: Path,
    generator: ConfigGenerator,
    resource_types: list[str] | None,
    seed: int,
) -> tuple[int, str, str]:
    """Run a single fuzzing iteration. Returns (return_code, output, config_yaml)."""
    config = generator.generate_config(resource_types)
    config_yaml = yaml.safe_dump(config, default_flow_style=False, sort_keys=False)

    with tempfile.TemporaryDirectory() as tmpdir:
        config_path = Path(tmpdir) / "databricks.yml"
        config_path.write_text(config_yaml)

        env = os.environ.copy()
        env["CLI"] = str(cli_path)

        result = subprocess.run(
            [sys.executable, str(script_path)],
            cwd=tmpdir,
            capture_output=True,
            text=True,
            env=env,
        )

        output = f"=== STDOUT ===\n{result.stdout}\n=== STDERR ===\n{result.stderr}"
        return result.returncode, output, config_yaml


def save_result(output_dir: Path, seed: int, error_code: int, config_yaml: str, output: str):
    """Save a bug result to disk."""
    result_dir = output_dir / f"run_{seed}_{error_code}"
    result_dir.mkdir(parents=True, exist_ok=True)

    (result_dir / "databricks.yml").write_text(config_yaml)
    (result_dir / "output.txt").write_text(output)

    print(f"Bug found! Saved to {result_dir}")
    print(f"Replay with: {sys.argv[0]} --seed {seed} {sys.argv[-1]}")


def main():
    parser = argparse.ArgumentParser(description="Bundle fuzzer")
    parser.add_argument("script", help="Test script to run")
    parser.add_argument("--seed", type=int, help="Random seed for reproducibility")
    parser.add_argument(
        "--resource",
        action="append",
        dest="resources",
        choices=["jobs", "pipelines"],
        help="Resource types to generate (default: jobs)",
    )
    parser.add_argument("--iterations", type=int, default=100, help="Number of iterations")
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=Path("fuzzer_results"),
        help="Output directory for bug results",
    )
    parser.add_argument(
        "--use-testserver",
        action="store_true",
        help="Use testserver (sets DATABRICKS_HOST/TOKEN env vars)",
    )
    args = parser.parse_args()

    script_path = Path(args.script).resolve()
    if not script_path.exists():
        print(f"Script not found: {script_path}", file=sys.stderr)
        sys.exit(1)

    cli_path = get_cli_path()
    if not cli_path.exists():
        print(f"CLI binary not found: {cli_path}", file=sys.stderr)
        sys.exit(1)

    if args.use_testserver and not TESTSERVER_PATH.exists():
        print(f"Testserver not found: {TESTSERVER_PATH}", file=sys.stderr)
        print("Build it with: go build -o testserver ./cmd/testserver", file=sys.stderr)
        sys.exit(1)

    # Load schema
    with open(SCHEMA_PATH) as f:
        schema = json.load(f)

    resolver = SchemaResolver(schema)
    resource_types = args.resources or ["jobs"]

    base_seed = args.seed if args.seed is not None else random.randint(0, 2**32 - 1)
    iterations = 1 if args.seed is not None else args.iterations

    testserver = None
    if args.use_testserver:
        testserver = TestServer()
        testserver.start()
        os.environ["DATABRICKS_HOST"] = testserver.url
        os.environ["DATABRICKS_TOKEN"] = "test-token"
        print(f"Started testserver at {testserver.url}")

    try:
        error_counts = Counter()

        def print_summary():
            if not error_counts:
                return
            parts = [f"code {c}: {error_counts[c]}" for c in sorted(error_counts)]
            print(f"Summary so far: {', '.join(parts)}")

        for i in range(iterations):
            seed = base_seed + i if args.seed is None else base_seed
            generator = ConfigGenerator(resolver, seed)

            print_summary()
            print(f"=== Iteration {i + 1}/{iterations} (seed={seed}) ===")

            return_code, output, config_yaml = run_iteration(cli_path, script_path, generator, resource_types, seed)

            print(config_yaml)
            print(output)

            error_counts[return_code] += 1

            if return_code == 0:
                print(f"Result: OK\n")
            elif 1 <= return_code <= 9:
                print(f"Result: Syntax error (code={return_code})\n")
            elif 10 <= return_code <= 19:
                print(f"Result: BUG DETECTED (code={return_code})")
                save_result(args.output_dir, seed, return_code, config_yaml, output)
                print()
            else:
                print(f"Result: Unknown return code: {return_code}\n")

        # Print summary
        print("=" * 60)
        print(f"Summary: {iterations} total runs")
        for code in sorted(error_counts.keys()):
            count = error_counts[code]
            print(f"  code {code}: {count}")
    finally:
        if testserver:
            testserver.stop()


if __name__ == "__main__":
    main()
