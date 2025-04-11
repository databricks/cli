sources = databricks databricks_tests

fmt:
	uv run ruff check --fix $(sources) || true
	uv run ruff format

docs:
	# Python 3.12+ is needed for get_overloads
	uv run --python 3.12 sphinx-build docs docs/_output --show-traceback --nitpicky --fresh-env --keep-going

lint:
	# check if lock matches the project metadata
	uv lock --check

	uv run ruff check $(sources)
	uv run pyright
	uv run ruff format --diff

codegen:
	find databricks -name _models | xargs rm -rf

	cd codegen; uv run -m pytest codegen_tests
	cd codegen; uv run -m codegen.main --output ..

	uv run ruff check --fix $(sources) || true
	uv run ruff format

test:
	uv run python -m pytest databricks_tests --cov=databricks.bundles --cov-report html -vv

build:
	rm -rf build dist
	uv build .

.PHONY: fmt docs lint codegen test build
