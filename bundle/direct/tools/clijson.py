#!/usr/bin/env python3

import hashlib
import json
import re
from pathlib import Path

FORMAT_VERSION = 1
MAX_SHARD_BYTES = 1_000_000
IDENTIFIER = re.compile(r"[A-Za-z_][A-Za-z0-9_]*")


def read_file(path):
    if path.is_symlink() or not path.is_file():
        raise ValueError(f"Shard {path} is not a regular file")
    with path.open("rb") as stream:
        content = stream.read(MAX_SHARD_BYTES)
    if len(content) >= MAX_SHARD_BYTES:
        raise ValueError(f"Shard {path} must be smaller than {MAX_SHARD_BYTES} bytes")
    return content


def load_schemas(directory):
    directory = Path(directory)
    index = json.loads(read_file(directory / "index.json"))
    if not isinstance(index, dict):
        raise ValueError("Shard index must be an object")
    if type(index.get("version")) is not int or index["version"] != FORMAT_VERSION:
        raise ValueError(f"Unsupported shard format version {index.get('version')!r}")
    if (
        not isinstance(index.get("schemas"), list)
        or not index["schemas"]
        or not isinstance(index.get("services"), list)
        or not index["services"]
        or not isinstance(index.get("commands"), dict)
        or index["commands"].get("services") not in (None, [])
    ):
        raise ValueError("Shard index must list schemas and services, with command metadata only")

    expected = set()
    schemas = {}
    for section in ("schemas", "services"):
        for entry in index[section]:
            if not isinstance(entry, dict) or not isinstance(entry.get("id"), str):
                raise ValueError(f"Invalid shard entry {entry!r}")
            identifier = entry["id"]
            parts = identifier.split(".") if section == "services" else [identifier]
            if len(parts) != (2 if section == "services" else 1) or any(
                IDENTIFIER.fullmatch(part) is None for part in parts
            ):
                raise ValueError(f"Invalid shard ID {identifier!r}")
            relative = (
                Path("commands", parts[0], parts[1] + ".json")
                if section == "services"
                else Path("schemas", identifier + ".json")
            )
            if relative in expected:
                raise ValueError(f"Duplicate shard {relative}")
            expected.add(relative)
            content = read_file(directory / relative)
            if hashlib.sha256(content).hexdigest() != entry["sha256"]:
                raise ValueError(f"Checksum mismatch for shard {relative}")
            shard = json.loads(content)
            if section == "services":
                if not isinstance(shard, dict) or shard.get("id") != identifier:
                    raise ValueError(f"Service ID does not match indexed ID {identifier!r}")
                continue
            if not isinstance(shard, dict) or not shard:
                raise ValueError(f"Empty schema package {identifier}")
            for name, schema in shard.items():
                package, separator, type_name = name.partition(".")
                if package != identifier or not separator or IDENTIFIER.fullmatch(type_name) is None or schema is None:
                    raise ValueError(f"Invalid schema {name!r} in package {identifier}")
                if name in schemas:
                    raise ValueError(f"Duplicate schema {name!r}")
                schemas[name] = schema

    def check_inventory(root):
        if root.is_symlink():
            raise ValueError(f"Shard directory {root} is a symlink")
        for path in root.iterdir():
            if path.is_dir() and not path.is_symlink():
                check_inventory(path)
            elif path.relative_to(directory) not in expected:
                raise ValueError(f"Unexpected shard {path.relative_to(directory)}")

    for sub in ("schemas", "commands"):
        check_inventory(directory / sub)
    return schemas
