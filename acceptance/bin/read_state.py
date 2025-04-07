#!/usr/bin/env python3
import sys
import os
import json


def print_resource_terraform(section, name, *attrs):
    resource_type = "databricks_" + section[:-1]
    filename = ".databricks/bundle/default/terraform/terraform.tfstate"
    data = json.load(open(filename))
    available = []
    found = 0
    for r in data["resources"]:
        r_type = r["type"]
        r_name = r["name"]
        if r_type != resource_type:
            available.append((r_type, r_name))
            continue
        if r_name != name:
            available.append((r_type, r_name))
            continue
        for inst in r["instances"]:
            attribute_values = inst.get("attributes")
            if attribute_values:
                values = [f"{x}={attribute_values.get(x)!r}" for x in attrs]
                print(section, name, " ".join(values))
                found += 1
    if not found:
        print(f"Resource {(resource_type, name)} not found. Available: {available}")


def print_resource_terranova(section, name, attrs):
    pass


if os.environ.get("TERRANOVA"):
    print_resource_terranova(*sys.argv[1:])
else:
    print_resource_terraform(*sys.argv[1:])
