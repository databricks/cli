#!/usr/bin/env python3
"""
Contract checks for mutate_fuzz_config. Failures go to stderr; stdout samples mutated
configs so an algorithm change shows up as an acceptance output diff.

- every MUTATE_BASES entry stays in the loader dialect and parses to one non-empty instance
- every base type has INJECT entries or a NO_INJECT reason, never both or neither
- every INJECT field is a settable input in the committed reference schema
- mutate(seed) is deterministic for every base
- sample volume seeds stay pairwise distinct (so output.txt catches algorithm drift)
- INJECT eventually lands on a sparse base (registered_model)
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from envsubst import substitute_variables
from mutate_fuzz_config import INJECT, MUTATE_BASES, NO_INJECT, dump_config, load_yaml, mutate

BIN = os.path.dirname(os.path.abspath(__file__))
CONFIGS = os.path.join(BIN, "..", "bundle", "invariant", "configs")
FIELDS = os.path.join(BIN, "..", "bundle", "refschema", "out.fields.txt")


def render(name):
    with open(os.path.join(CONFIGS, name + ".yml.tmpl")) as f:
        return substitute_variables(f.read())


def load(name):
    return load_yaml(render(name))


def instance(config):
    (instances,) = config["resources"].values()
    (value,) = instances.values()
    return value


def field_flags():
    """Map field path -> flags. A path can repeat (one Go type each), so union them."""
    flags = {}
    with open(FIELDS) as f:
        for line in f:
            path, _, rest = line.rstrip("\n").partition("\t")
            if path:
                flags.setdefault(path, set()).update(rest.split("\t")[1:])
    return flags


def dialect_ok(name, text):
    """Bases must stay in the subset load_yaml understands (block style, full-line # only)."""
    ok = True
    for lineno, raw in enumerate(text.splitlines(), 1):
        stripped = raw.lstrip(" ")
        if not stripped or stripped.startswith("#"):
            continue
        if "#" in stripped:
            sys.stderr.write(f"{name}:{lineno}: trailing comment is not supported by load_yaml\n")
            ok = False
        if ": " in stripped:
            rest = stripped.partition(": ")[2]
        elif stripped.startswith("- "):
            rest = stripped[2:]
        else:
            rest = ""
        # "" in "[{" is True; require a real opener.
        if rest[:1] in ("{", "["):
            sys.stderr.write(f"{name}:{lineno}: flow-style value is not supported by load_yaml\n")
            ok = False
    return ok


def main():
    # Pin UNIQUE_NAME so printed configs are stable across harness runs.
    os.environ["UNIQUE_NAME"] = "check"
    os.environ.setdefault("CURRENT_USER_NAME", "check-user")
    failed = False

    for name in MUTATE_BASES:
        text = render(name)
        if not dialect_ok(name, text):
            failed = True
        parsed = load_yaml(text)
        if not isinstance(parsed, dict) or "resources" not in parsed:
            sys.stderr.write(f"{name}: base did not parse to a config with resources\n")
            failed = True
            continue
        resources = parsed["resources"]
        if len(resources) != 1:
            sys.stderr.write(f"{name}: expected one resource type, got {sorted(resources)}\n")
            failed = True
            continue
        instances = next(iter(resources.values()))
        if not isinstance(instances, dict) or len(instances) != 1:
            sys.stderr.write(f"{name}: expected one resource instance\n")
            failed = True
            continue
        # A mis-parse can still yield a dict; empty instance means the loader dropped fields.
        if not next(iter(instances.values())):
            sys.stderr.write(f"{name}: base parsed to an empty resource instance\n")
            failed = True
        rtype = next(iter(resources))
        if bool(INJECT.get(rtype)) == (rtype in NO_INJECT):
            sys.stderr.write(
                f"{name}: resources.{rtype} needs INJECT entries or a NO_INJECT reason, not both or neither\n"
            )
            failed = True

    # Unknown fields are only a warning, so a typo would deploy as a silent no-op inject.
    flags = field_flags()
    for rtype, fields in INJECT.items():
        for field, _ in fields:
            path = f"resources.{rtype}.*.{field}"
            if not flags.get(path, set()) & {"INPUT", "ALL"}:
                sys.stderr.write(f"INJECT[{rtype}]: {path} is not a settable input field\n")
                failed = True

    for name in MUTATE_BASES:
        for seed in range(5):
            a = dump_config(mutate(load(name), seed))
            b = dump_config(mutate(load(name), seed))
            if a != b:
                sys.stderr.write(f"{name} seed {seed}: mutation is not deterministic\n")
                failed = True

    # Distinct dumps: colliding samples hide algorithm changes in output.txt.
    samples = [0, 1, 5]
    dumps = []
    for seed in samples:
        out = dump_config(mutate(load("volume"), seed))
        dumps.append(out)
        sys.stdout.write(f"=== volume seed={seed} ===\n")
        sys.stdout.write(out)
    if len(set(dumps)) != len(dumps):
        sys.stderr.write(f"sample seeds {samples} are not pairwise distinct\n")
        failed = True

    # Sparse base: any new field must come from INJECT.
    base_fields = set(instance(load("registered_model")))
    inject_names = {name for name, _ in INJECT["registered_models"]}
    injected = False
    for seed in range(30):
        fields = set(instance(mutate(load("registered_model"), seed)))
        added = fields - base_fields
        if added:
            injected = True
        unexpected = added - inject_names
        if unexpected:
            sys.stderr.write(f"seed {seed}: injected non-catalog field(s): {sorted(unexpected)}\n")
            failed = True
    if not injected:
        sys.stderr.write("mutation never injected a curated optional field\n")
        failed = True

    if failed:
        sys.exit(1)


if __name__ == "__main__":
    main()
