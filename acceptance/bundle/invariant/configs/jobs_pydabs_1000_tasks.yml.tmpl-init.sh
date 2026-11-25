#!/bin/bash

uv venv --quiet
uv pip install --quiet "$DATABRICKS_BUNDLES_WHEEL"
