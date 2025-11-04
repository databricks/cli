#!/usr/bin/env python3
"""
Print resources state from default target.

Note, this intentionally has no logic on guessing what is the right stat file (e.g. via DATABRICKS_BUNDLE_ENGINE)
the goal is to record all states that are available.
"""

import os


def write(filename):
    data = open(filename).read()
    print(data, end="")
    if not data.endswith("\n"):
        print()


filename = ".databricks/bundle/default/terraform/terraform.tfstate"
if os.path.exists(filename):
    write(filename)

filename = ".databricks/bundle/default/resources.json"
if os.path.exists(filename):
    write(filename)
