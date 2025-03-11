sources = databricks databricks_tests

fmt:
	uv run ruff check --fix $(sources) || true
	uv run ruff format

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
