#!/usr/bin/env python3
# /// script
# dependencies = [
#   "pyyaml",
# ]
# ///
"""Snapshot the MUTATE_BASES invariant configs as fuzz/bases/*.json.tmpl.

Run via: ./task generate-fuzz-bases
Keeps $UNIQUE_NAME / $CURRENT_USER_NAME for envsubst at test time.
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
        config = yaml.safe_load((CONFIGS / f"{name}.yml.tmpl").read_text())
        (OUT / f"{name}.json.tmpl").write_text(json.dumps(config, indent=2, ensure_ascii=False) + "\n")


if __name__ == "__main__":
    main()
