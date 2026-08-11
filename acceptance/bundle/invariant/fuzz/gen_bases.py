#!/usr/bin/env python3
"""Regenerate fuzz/bases/*.json.tmpl from sibling invariant YAML configs.

Offline (PyYAML). From repo root:
  python3 acceptance/bundle/invariant/fuzz/gen_bases.py
Keeps $UNIQUE_NAME / $CURRENT_USER_NAME for runtime envsubst.
"""

import json
import sys
from pathlib import Path

import yaml

FUZZ = Path(__file__).resolve().parent
INVARIANT = FUZZ.parent
ROOT = INVARIANT.parents[2]
sys.path.insert(0, str(ROOT / "acceptance" / "bin"))

from mutate_fuzz_config import MUTATE_BASES  # noqa: E402

CONFIGS = INVARIANT / "configs"
OUT = FUZZ / "bases"


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
