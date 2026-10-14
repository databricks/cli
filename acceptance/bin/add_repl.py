#!/usr/bin/env python3
"""
Add entry to ACC_REPLS without clobbering existing ones.

If entry already exists, it'll add suffix in _<number> format.
"""

import argparse
import json
import os
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from repls import USER_ORDER, read_entries

ACC_REPLS = Path(os.environ["ACC_REPLS"])


def get_repls():
    # Only the literal entries are added here, so only those can collide.
    return {item["New"] for item in read_entries() if item.get("Literal") is not None}


def add_repl(value, repl):
    existing = get_repls()
    for extra in range(1, 100):
        if extra >= 2:
            r = f"{repl}_{extra}"
        else:
            r = repl
        if f"[{r}]" in existing:
            continue
        with ACC_REPLS.open("a") as fobj:
            json.dump({"Literal": value, "New": f"[{r}]", "Order": USER_ORDER}, fobj)
            fobj.write("\n")
        break


def id_repl_name(key):
    """Replacement name for the id of the resource with the given key: "foo" -> "FOO_ID".

    The name is upper-cased, so a $UNIQUE_NAME inside the key would escape the harness's
    (case-sensitive) [UNIQUE_NAME] replacement and make the name differ on every run.
    Substitute it first: "app_${UNIQUE_NAME}_job" -> "APP_UNIQUE_NAME_JOB_ID".
    """
    return key.replace(os.environ["UNIQUE_NAME"], "UNIQUE_NAME").upper() + "_ID"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("value")
    parser.add_argument("replacement")
    args = parser.parse_args()
    add_repl(args.value, args.replacement)


if __name__ == "__main__":
    main()
