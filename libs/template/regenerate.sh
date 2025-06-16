#!/bin/bash
set -ex
./generate_uv_lock.py ./templates/default-python/template/{{.project_name}}/pyproject.toml.tmpl
