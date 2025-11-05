## Changes

Adds a new `default-minimal` template for advanced users who want a clean slate without sample code.

**Additional changes:**
- Enhanced template renderer to preserve empty directories (needed for minimal template structure)
- Fixed template schema consistency with default-python
- Fixed pyproject.toml for templates without Python package directories

## Why

Advanced users don't want to delete sample code when starting new projects. This gives them just the bundle structure.

## Tests

All 27 template acceptance tests passing.

**Example:**
```bash
databricks bundle init default-minimal
cd my_minimal_project
find .
# Output shows clean structure with empty src/ and resources/ directories
```
