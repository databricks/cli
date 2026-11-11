#!/usr/bin/env python3
"""
Creates the feature tables and Python UDFs that the feature_specs acceptance tests look up, in an
existing schema. Creating a feature spec resolves each of them in Unity Catalog.

Usage: create_feature_spec_fixtures.py CATALOG.SCHEMA
"""

import json
import os
import subprocess
import sys
import time

schema_full_name = sys.argv[1]
cli = os.environ.get("CLI", "databricks")
warehouse_id = os.environ["TEST_DEFAULT_WAREHOUSE_ID"]

statements = [
    f"CREATE TABLE {schema_full_name}.user_features "
    "(user_id INT NOT NULL, age INT, country STRING, CONSTRAINT user_features_pk PRIMARY KEY (user_id))",
    f"CREATE TABLE {schema_full_name}.tenant_user_features "
    "(tenant_id INT NOT NULL, user_id INT NOT NULL, age INT, country STRING, "
    "CONSTRAINT tenant_user_features_pk PRIMARY KEY (tenant_id, user_id))",
    f"CREATE FUNCTION {schema_full_name}.score(x INT) RETURNS DOUBLE LANGUAGE PYTHON AS $$ return float(x) $$",
    f"CREATE FUNCTION {schema_full_name}.country_score(x INT, y STRING) RETURNS DOUBLE LANGUAGE PYTHON "
    "AS $$ return float(x) + len(y) $$",
]


def call_api(*args):
    result = subprocess.run([cli, "api", *args], capture_output=True, text=True)
    if result.returncode != 0:
        print(f"databricks api {' '.join(args)} failed: {result.stderr}", file=sys.stderr)
        sys.exit(1)
    return json.loads(result.stdout)


for statement in statements:
    payload = json.dumps({"warehouse_id": warehouse_id, "statement": statement, "wait_timeout": "50s"})
    response = call_api("post", "/api/2.0/sql/statements/", "--json", payload)
    # A warehouse that is still starting outlasts wait_timeout.
    deadline = time.monotonic() + 600
    while response["status"]["state"] in ("PENDING", "RUNNING") and time.monotonic() < deadline:
        time.sleep(5)
        response = call_api("get", f"/api/2.0/sql/statements/{response['statement_id']}")
    if response["status"]["state"] != "SUCCEEDED":
        print(f"Statement {statement!r} ended in {response['status']}", file=sys.stderr)
        sys.exit(1)
