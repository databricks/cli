#!/usr/bin/env python3
"""
Creates the feature tables and Python UDFs that the feature_specs acceptance tests look up, in an
existing schema. Creating a feature spec resolves each of them in Unity Catalog.

Usage: create_feature_spec_fixtures.py CATALOG.SCHEMA
"""

import sys

from util import run_statements

schema_full_name = sys.argv[1]

run_statements(
    [
        f"CREATE TABLE {schema_full_name}.user_features "
        "(user_id INT NOT NULL, age INT, country STRING, CONSTRAINT user_features_pk PRIMARY KEY (user_id))",
        f"CREATE TABLE {schema_full_name}.tenant_user_features "
        "(tenant_id INT NOT NULL, user_id INT NOT NULL, age INT, country STRING, "
        "CONSTRAINT tenant_user_features_pk PRIMARY KEY (tenant_id, user_id))",
        f"CREATE FUNCTION {schema_full_name}.score(x INT) RETURNS DOUBLE LANGUAGE PYTHON AS $$ return float(x) $$",
        f"CREATE FUNCTION {schema_full_name}.country_score(x INT, y STRING) RETURNS DOUBLE LANGUAGE PYTHON "
        "AS $$ return float(x) + len(y) $$",
    ]
)
