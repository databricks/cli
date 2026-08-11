#!/usr/bin/env python3
"""Regenerate fuzz/bases/*.json.tmpl from invariant YAML configs.

Offline (PyYAML). From repo root: python3 acceptance/bundle/fuzz/gen_bases.py
Keeps $UNIQUE_NAME / $CURRENT_USER_NAME for runtime envsubst.
"""

import json
import sys
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / "acceptance" / "bin"))

from mutate_fuzz_config import MUTATE_BASES  # noqa: E402

CONFIGS = ROOT / "acceptance" / "bundle" / "invariant" / "configs"
OUT = Path(__file__).resolve().parent / "bases"


def main():
    OUT.mkdir(exist_ok=True)
    for name in MUTATE_BASES:
        text = (CONFIGS / f"{name}.yml.tmpl").read_text()
        config = yaml.safe_load(text)
        path = OUT / f"{name}.json.tmpl"
        path.write_text(json.dumps(config, indent=2, ensure_ascii=False) + "\n")
        print(path.relative_to(ROOT))


if __name__ == "__main__":
    main()
