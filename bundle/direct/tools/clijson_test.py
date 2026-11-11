#!/usr/bin/env python3
# /// script
# dependencies = [
#   "pyyaml",
# ]
# ///

import hashlib
import json
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

from clijson import MAX_SHARD_BYTES, load_schemas

ROOT = Path(__file__).resolve().parents[3]
FIXTURE = ROOT / "internal/clijson/testdata/shards"
TOOLS = Path(__file__).resolve().parent


class ShardTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.directory = self.root / "checkout[1]"
        shutil.copytree(FIXTURE, self.directory)

    def change_index(self, change):
        path = self.directory / "index.json"
        index = json.loads(path.read_bytes())
        change(index)
        path.write_text(json.dumps(index))

    def replace_shard(self, relative, content, rehash=False):
        path = self.directory / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)
        if rehash:
            digest = hashlib.sha256(content.encode()).hexdigest()

            def update(index):
                for section in ("schemas", "services"):
                    for entry in index[section]:
                        expected = (
                            f"schemas/{entry['id']}.json"
                            if section == "schemas"
                            else f"commands/{entry['id'].replace('.', '/')}.json"
                        )
                        if relative == expected:
                            entry["sha256"] = digest

            self.change_index(update)

    def test_load_schemas(self):
        schemas = load_schemas(self.directory)
        self.assertEqual(set(schemas), {"alpha.Request", "alpha.Unused", "zeta.State"})
        self.assertEqual(schemas["zeta.State"]["enum"], ["READY", "STOPPED"])
        self.assertEqual(schemas["alpha.Request"]["fields"]["state"]["ref"], "zeta.State")
        self.assertEqual(schemas["alpha.Request"]["fields"]["state"]["behaviors"], ["IMMUTABLE"])

    def test_invalid_files(self):
        cases = [
            ("index.json", None, False),
            ("schemas/alpha.json", None, False),
            ("commands/zeta/Second.json", None, False),
            ("schemas/alpha.json", '{"alpha.Request":{}}', False),
            ("schemas/zeta.json", "{}", False),
            ("index.json", "{", False),
            ("index.json", "null", False),
            ("schemas/zeta.json", "{", True),
            ("commands/zeta/Second.json", "{", True),
            ("schemas/zeta.json", "null", True),
            ("schemas/zeta.json", "{}", True),
            ("schemas/zeta.json", '{"zeta.State":null}', True),
            ("schemas/zeta.json", '{"alpha.State":{}}', True),
            ("schemas/zeta.json", '{"zeta.A.B":{}}', True),
            ("commands/zeta/Second.json", '{"id":"alpha.First"}', True),
            ("schemas/stale.json", "{}", False),
            ("commands/extra/Stale.json", "{}", False),
            ("schemas/zeta.json", " " * MAX_SHARD_BYTES, False),
            ("index.json", " " * MAX_SHARD_BYTES, False),
        ]
        for relative, content, rehash in cases:
            with self.subTest(relative=relative, content=str(content)[:40], rehash=rehash):
                shutil.rmtree(self.directory)
                shutil.copytree(FIXTURE, self.directory)
                if content is None:
                    (self.directory / relative).unlink()
                else:
                    self.replace_shard(relative, content, rehash)
                with self.assertRaises((ValueError, OSError)):
                    load_schemas(self.directory)
        with self.assertRaises((ValueError, OSError)):
            load_schemas(self.root / "missing")

    def test_invalid_index(self):
        changes = [
            lambda index: index.update(version=2),
            lambda index: index.update(version=True),
            lambda index: index.pop("version"),
            lambda index: index.update(schemas=[]),
            lambda index: index.update(services=[]),
            lambda index: index.update(commands=None),
            lambda index: index["commands"].update(services=[{"id": "zeta.Second"}]),
            lambda index: index["schemas"].append(index["schemas"][0]),
            lambda index: index["services"].append(index["services"][0]),
            lambda index: index["schemas"][0].update(id="../escape"),
            lambda index: index["services"][0].update(id="zeta.A.B"),
            lambda index: index["schemas"][0].update(sha256=""),
        ]
        for position, change in enumerate(changes):
            with self.subTest(position=position):
                shutil.copyfile(FIXTURE / "index.json", self.directory / "index.json")
                self.change_index(change)
                with self.assertRaises(ValueError):
                    load_schemas(self.directory)

    def test_generators_preserve_outputs_on_invalid_input(self):
        fields = self.root / "fields"
        fields.mkdir()
        apitypes = self.root / "apitypes.yml"
        apitypes.write_text("widgets: alpha.Request\n")
        override = self.root / "override.yml"
        override.write_text("{}\n")
        output = self.root / "output"
        output.mkdir()
        resource = output / "widgets.generated.yml"
        resource.write_text("previous resource\n")
        for relative, content in [
            ("index.json", None),
            ("schemas/alpha.json", '{"alpha.Request":{}}'),
            ("commands/zeta/Second.json", "{}"),
        ]:
            with self.subTest(relative=relative):
                shutil.rmtree(self.directory)
                shutil.copytree(FIXTURE, self.directory)
                if content is None:
                    (self.directory / relative).unlink()
                else:
                    self.replace_shard(relative, content)
                commands = [
                    [TOOLS / "generate_apitypes.py", self.directory, fields, "--output", apitypes],
                    [TOOLS / "generate_resources.py", self.directory, apitypes, override, fields, output],
                ]
                for command in commands:
                    result = subprocess.run([sys.executable, *map(str, command)], capture_output=True, text=True)
                    self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
                self.assertEqual(apitypes.read_text(), "widgets: alpha.Request\n")
                self.assertEqual(
                    {path.name: path.read_text() for path in output.iterdir()},
                    {
                        "widgets.generated.yml": "previous resource\n",
                    },
                )


if __name__ == "__main__":
    unittest.main()
