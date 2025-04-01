sources = databricks databricks_tests

fmt:
	uv run ruff check --fix $(sources) || true
	uv run ruff format

docs:
	# Python 3.12+ is needed for get_overloads
	uv run --python 3.12 sphinx-build docs docs/_output --fail-on-warning --show-traceback --nitpicky --fresh-env --keep-going

lint:
	# check if lock matches the project metadata
	uv lock --check

	uv run ruff check $(sources)
	uv run pyright
	uv run ruff format --diff

test:
	uv run python -m pytest databricks_tests --cov=databricks.bundles --cov-report html -vv

build:
	rm -rf build dist
	uv build .

.PHONY: fmt docs lint test build
