#!/usr/bin/env python3
"""
Print selected attributes from terraform state.

Usage: <section> <name> [attr...]
"""

import sys
import os
import json


def print_resource_terraform(section, name):
    resource_type = "databricks_" + section[:-1]
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


print_resource_terraform(*sys.argv[1:])
