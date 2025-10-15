# Update cli-pipelines template to align with Lakeflow conventions

## Changes

This PR modernizes the `cli-pipelines` template to align with the Lakeflow conventions introduced in PR #3671 (commit c30c45611), while preserving its unique characteristic of keeping `transformations/` in the project root:

**Key updates:**
* Add `pyproject.toml.tmpl` with modern Python project configuration (PEP 621, hatchling, dependency-groups)
* Add editable install via `environment.dependencies` in pipeline configuration
* Update `databricks.yml.tmpl` to match default template structure (comments, variable descriptions)
* Fix VS Code settings `extraPaths` to point to actual directories (`transformations`, `utilities`)
* Add `library/versions.tmpl` for reusable template version definitions
* Update pipeline.yml comments to match default template's environment.dependencies documentation

**Testing infrastructure:**
* Add comprehensive acceptance tests for Python and SQL variants
* Add comparison test (`compare-with-lakeflow/`) that validates structural alignment with lakeflow-pipelines template
  - Generates both templates with equivalent settings
  - Compares key files (pyproject.toml, databricks.yml, pipeline.yml)
  - Documents expected differences in a single diff file
  - Serves as regression test for future maintenance

**Differences preserved:**
* `transformations/` and `utilities/` remain in project root (not moved to `src/` like lakeflow-pipelines)
* Pipeline definitions at project root (`*.pipeline.yml`) rather than in `resources/` subdirectory
* Explicit package declarations in pyproject.toml (`packages = ["transformations", "utilities"]`)
* README.md focuses on pipelines CLI (`pipelines deploy`, `pipelines run`)

## Why

* The cli-pipelines template was outdated and didn't follow modern Lakeflow conventions
* Aligning with the default template makes maintenance easier and provides a consistent user experience
* The template now supports edit mode with editable installs for better development workflow
* Users get modern tooling (pyproject.toml, dependency-groups) while keeping the simpler root-level layout

## Tests

* All acceptance tests pass (Python variant, SQL variant, comparison test)
* Template validates successfully with `databricks bundle validate`
* Template deploys and runs successfully on e2-dogfood workspace
* Comparison test documents exactly what differs from lakeflow-pipelines template

---

## Guidance for AI Agents: Maintaining Alignment with Default Template

This section provides guidance for AI agents on how to keep the cli-pipelines template aligned with the upstream default template as it evolves.

### Understanding the Relationship

The cli-pipelines template is **structurally based on** the default template (from commit c30c45611, PR #3671), with intentional differences:

**Same structure/conventions:**
- pyproject.toml format (PEP 621, hatchling, dependency-groups structure)
- databricks.yml structure (variable declarations, target comments, deployment-modes link)
- Pipeline environment.dependencies format and comments
- Version specifications (conservative DB Connect versions)
- VS Code settings format

**Intentional differences:**
- **Layout**: transformations/ in root (cli-pipelines) vs src/<name>/ (default)
- **Include patterns**: `*.pipeline.yml` (cli-pipelines) vs `resources/*.yml` (default)
- **Package declarations**: Explicit in pyproject.toml (cli-pipelines) vs implicit (default with src/ layout)
- **Comments**: "pipelines" (cli-pipelines) vs "resources" (default) where appropriate

### The Comparison Test is Your Tool

The acceptance test at `acceptance/bundle/templates/cli-pipelines/compare-with-lakeflow/` is the **source of truth** for expected differences:

1. It generates both cli-pipelines and lakeflow-pipelines with identical inputs
2. It compares key files and outputs a **single diff file** showing all differences
3. The diff file (`out.compare.diff`) documents what SHOULD be different

**When the default template changes:**
1. Run the comparison test: `go test ./acceptance -run TestAccept/bundle/templates/cli-pipelines/compare-with-lakeflow -update`
2. Examine the updated diff file
3. New differences = potential divergence that needs review
4. Ask yourself: "Is this a structural difference (expected) or a conventions difference (should align)?"

### Maintenance Checklist

When updating cli-pipelines to match a new version of the default template:

**Step 1: Identify changes in default template**
```bash
# Compare current default template to previous version
git diff <old-commit> <new-commit> libs/template/templates/default/
```

**Step 2: Categorize each change**

For each change in the default template, determine if it should propagate:

**✅ Should propagate (conventions/comments/structure):**
- Comments and documentation strings
- pyproject.toml sections (dependencies, dependency-groups, build-system, tool.black)
- Environment.dependencies comments
- Variable descriptions in databricks.yml
- Development workflow patterns (edit mode, editable installs)
- Version specifications (conservative_db_connect_version_spec, etc.)

**❌ Should NOT propagate (structural differences):**
- File paths (src/ vs root)
- Include patterns (resources/*.yml vs *.pipeline.yml)
- Package declarations (explicit packages vs implicit)
- Names with "etl" suffix (lakeflow uses compare_test_etl, cli-pipelines uses compare_test_pipeline)
- Anything specific to the src/ layout

**❓ Review carefully (context-dependent):**
- New sections in pyproject.toml (evaluate if relevant for pipeline-only template)
- New presets in databricks.yml (check if applicable - e.g., artifacts_dynamic_version only for classic compute)
- Changes to schema defaults (cli-pipelines has different prod_schema logic, this is intentional)

**Step 3: Apply changes**
- Make changes to cli-pipelines template files
- Update acceptance tests: `go test ./acceptance -run TestAccept/bundle/templates/cli-pipelines -update`
- Verify comparison test shows only expected differences

**Step 4: Validate**
```bash
# Run all cli-pipelines tests
go test ./acceptance -run TestAccept/bundle/templates/cli-pipelines

# Check the comparison diff
cat acceptance/bundle/templates/cli-pipelines/compare-with-lakeflow/out.compare.diff

# Should show ONLY structural differences, not new convention drift
```

### Specific Files to Monitor

**Always keep synchronized:**
- `library/versions.tmpl` - Version specifications should match exactly
- `pyproject.toml.tmpl` - Sections, comments, dependencies structure (but not package paths)
- Environment.dependencies comment in pipeline.yml

**Keep structurally similar:**
- `databricks.yml.tmpl` - Same sections/comments, different include patterns
- `.vscode/settings.json.tmpl` - Same structure, different extraPaths values

**Divergence is expected:**
- `README.md.tmpl` - cli-pipelines documents pipelines CLI, default documents databricks CLI
- Package/resource naming conventions

### Example: Applying a Hypothetical Update

**Scenario:** Default template adds a new `[tool.ruff]` section to pyproject.toml

**Analysis:**
- This is a **conventions change** (new tooling configuration)
- Not path-specific or structure-specific
- **Decision:** Should propagate ✅

**Steps:**
1. Add same `[tool.ruff]` section to cli-pipelines pyproject.toml.tmpl
2. Regenerate tests: `go test ./acceptance -run TestAccept/bundle/templates/cli-pipelines -update`
3. Check comparison diff - should show no NEW differences (tool.ruff same in both)
4. Commit with message explaining the alignment

**Scenario:** Default template changes `root_path: "../src/my_project"` to `root_path: "src/my_project"`

**Analysis:**
- This is a **structural change** specific to src/ layout
- cli-pipelines uses root layout, so it has `root_path: "."`
- **Decision:** Should NOT propagate ❌

**Steps:**
1. No change needed to cli-pipelines
2. The comparison test will show this as an expected difference
3. Verify the test still passes

### Red Flags

Watch out for these signs that cli-pipelines is diverging:

🚩 **New differences in comparison test** that aren't structural:
- Different pyproject.toml sections (not just paths)
- Different comment styles or missing documentation
- Different version specifications

🚩 **Acceptance tests fail** after default template update

🚩 **Comparison diff grows significantly** without clear reason

When in doubt: **Align on conventions, preserve on structure**. The comparison test is your guide.

### Commit Message Template

When updating cli-pipelines to match default template changes:

```
Sync cli-pipelines template with default template changes

Align with default template commit <commit-hash>:
- <specific change 1 that was applied>
- <specific change 2 that was applied>

Preserved structural differences:
- transformations/ remains in root
- Include patterns remain *.pipeline.yml
- [other preserved differences if relevant]

Verified with comparison test showing only expected structural differences.
```
