#!/usr/bin/env python3
"""
Print selected attributes from terraform state.

Usage: <group> <name> [attr...]
"""

import sys
import os
import json


def print_resource_terraform(group, name):
    resource_type = "databricks_" + group[:-1]
    filename = ".databricks/bundle/default/terraform/terraform.tfstate"
    raw = open(filename).read()
    data = json.loads(raw)
    found = 0
    for r in data["resources"]:
        r_type = r["type"]
        r_name = r["name"]
        if r_type != resource_type:
            continue
        if r_name != name:
            continue
        for inst in r["instances"]:
            attribute_values = inst.get("attributes") or {}
            print(attribute_values.get("id"))
            return


def print_resource_terranova(group, name):
    filename = ".databricks/bundle/default/resources.json"
    raw = open(filename).read()
    data = json.loads(raw)
    resources = data["resources"].get(group, {})
    result = resources.get(name)
    if result is None:
        print(f"Resource {group=} {name=} not found. Available: {raw}")
        return
    print(result.get("__id__"))


if os.environ.get("DATABRICKS_CLI_DEPLOYMENT", "").startswith("direct"):
    print_resource_terranova(*sys.argv[1:])
else:
    print_resource_terraform(*sys.argv[1:])
