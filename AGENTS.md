Below are some general AI assistant coding rules.

# General

When moving code from one place to another, please don't unnecessarily change
the code or omit parts.

# Style and comments

Please make sure code that you author is consistent with the codebase
and concise.

The code should be self-documenting based on the code and function names.

Functions should be documented with a doc comment as follows:

// SomeFunc does something.
func SomeFunc() {
	...
}

Note how the comment starts with the name of the function and is followed by a period.

Avoid redundant and verbose comments. Use terse comments and only add comments if it complements, not repeats the code.

Focus on making implementation as small and elegant as possible. Avoid unnecessary loops and allocations. If you see an opportunity of making things simpler by dropping or relaxing some requirements, ask user about the trade-off.

Use modern idiomatic Golang features (version 1.24+). Specifically:
 - Use for-range for integer iteration where possible. Instead of for i:=0; i < X; i++ {} you must write for i := range X{}.
 - Use builtin min() and max() where possible (works on any type and any number of values).
 - Do not capture the for-range variable, since go 1.22 a new copy of the variable is created for each loop iteration.

# Commands

Use "git rm" to remove and "git mv" to rename files instead of directly modifying files on FS.

Do not run “go test ./..." in the root folder as that will start long running integration tests. To test the whole project run "go build && make lint test" in root directory. However, prefer running tests for specific packages instead.

If asked to rebase, always prefix each git command with appropriate settings so that it never launches interactive editor.
GIT_EDITOR=true GIT_SEQUENCE_EDITOR=true VISUAL=true GIT_PAGER=cat git fetch origin main &&
GIT_EDITOR=true GIT_SEQUENCE_EDITOR=true VISUAL=true GIT_PAGER=cat git rebase origin/main

# Python

When writing Python scripts, we bias for conciseness. We think of Python in this code base as scripts.
 - use Python 3.11
 - Do not catch exceptions to make nicer messages, only catch if you can add critical information
 - use pathlib.Path in almost all cases over os.path unless it makes code longer
 - Do not add redundant comments.
 - Try to keep your code small and the number of abstractions low.
 - After done, format you code with "ruff format -n <path>"
 - Use "#!/usr/bin/env python3" shebang.

# Mutators

Mutators should have a structure as follows:

package mutator

type applySomeChange struct{}

// ApplySomeChange applies some change to the bundle.
func ApplySomeChange() *applySomeChange {
	return &applySomeChange{}
}

func (m *applySomeChange) Name() string {
	return "ApplySomeChange"
}

func (m *applySomeChange) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	...
}

In the code above, notice how the mutator name is used in
a public function, a struct, and the Name() method.
The mutator name should start with a verb such as Apply
and should be used consistently in the code.

# Tests

Each file like process_target_mode_test.go should have a corresponding test file
like process_target_mode_test.go. If you add new functionality to a file,
the test file should be extended to cover the new functionality.

Tests should look like the following:

package mutator_test

func TestApplySomeChangeReturnsDiagnostics(t *testing.T) {
	...
}

func TestApplySomeChangeFixesThings(t *testing.T) {
	ctx := context.Background()
	b, err := ...some operation...
	require.NoError(t, err)
	...
	assert.Equal(t, ...)
}

Notice that:
- Tests are often in the same package but suffixed wit _test.
- The test names are prefixed with Test and are named after the function or module they are testing.
- 'require' and 'require.NoError' are used to check for things that would cause the rest of the test case to fail.
- 'assert' is used to check for expected values where the rest of the test is not expected to fail.

When writing tests, please don't include an explanation in each
test case in your responses. I am just interested in the tests.

If you're editing acceptance tests configs test.toml files you need to make sure that you place stuff in the right place. Be aware of how TOML works and config definition in acceptance/internal/config.go. For example, if you incorrectly place Env.BLA="X" after [[Repls]] section, it won't parse correctly because it'll be associated with [[Repls]] block rather than TestConfig struct.

# databricks_template_schema.json

A databricks_template_schema.json file is used to configure bundle templates.

Below is a good reference template:

{
    "welcome_message": "\nWelcome to the dbt template for Databricks Asset Bundles!\n\nA workspace was selected based on your current profile. For information about how to change this, see https://docs.databricks.com/dev-tools/cli/profiles.html.\nworkspace_host: {{workspace_host}}",
    "properties": {
        "project_name": {
            "type": "string",
            "pattern": "^[A-Za-z_][A-Za-z0-9-_]+$",
            "pattern_match_failure_message": "Name must consist of letters, numbers, dashes, and underscores.",
            "default": "dbt_project",
            "description": "\nPlease provide a unique name for this project.\nproject_name",
            "order": 1
        },
        "http_path": {
            "type": "string",
            "pattern": "^/sql/.\\../warehouses/[a-z0-9]+$",
            "pattern_match_failure_message": "Path must be of the form /sql/1.0/warehouses/<warehouse id>",
            "description": "\nPlease provide the HTTP Path of the SQL warehouse you would like to use with dbt during development.\nYou can find this path by clicking on \"Connection details\" for your SQL warehouse.\nhttp_path [example: /sql/1.0/warehouses/abcdef1234567890]",
            "order": 2
        },
        "default_catalog": {
            "type": "string",
            "default": "{{default_catalog}}",
            "pattern": "^\\w*$",
            "pattern_match_failure_message": "Invalid catalog name.",
            "description": "\nPlease provide an initial catalog{{if eq (default_catalog) \"\"}} (leave blank when not using Unity Catalog){{end}}.\ndefault_catalog",
            "order": 3
        },
        "personal_schemas": {
            "type": "string",
            "description": "\nWould you like to use a personal schema for each user working on this project? (e.g., 'catalog.{{short_name}}')\npersonal_schemas",
            "enum": [
                "yes, use a schema based on the current user name during development",
                "no, use a shared schema during development"
            ],
            "order": 4
        },
        "shared_schema": {
            "skip_prompt_if": {
                "properties": {
                    "personal_schemas": {
                        "const": "yes, use a schema based on the current user name during development"
                    }
                }
            },
            "type": "string",
            "default": "default",
            "pattern": "^\\w+$",
            "pattern_match_failure_message": "Invalid schema name.",
            "description": "\nPlease provide an initial schema during development.\ndefault_schema",
            "order": 5
        }
    },
    "success_message": "\n📊 Your new project has been created in the '{{.project_name}}' directory!\nIf you already have dbt installed, just type 'cd {{.project_name}}; dbt init' to get started.\nRefer to the README.md file for full \"getting started\" guide and production setup instructions.\n"
}

Notice that:
- The welcome message has the template name.
- By convention, property messages  include the property name after a newline, e.g. default_catalog above has a description that says "\nPlease provide an initial catalog [...].\ndefault_catalog",
- Each property defines a variable that is used for the template.
- Each property has a unique 'order' value that increments by 1 with each property.
- Enums use 'type: "string' and have an 'enum' field with a list of possible values.
- Helpers such as {{default_catalog}} and {{short_name}} can be used within property descriptors.
- Properties can be referenced in messages and descriptions using {{.property_name}}. {{.project_name}} is an example.

# Logging and output to the terminal

Use the following for logging:

```
import "github.com/databricks/cli/libs/log"

log.Infof(ctx, "...")
log.Debugf(ctx, "...")
log.Warnf(ctx, "...")
log.Errorf(ctx, "...")
```

Note that the 'ctx' variable here is something that should be passed in as
an argument by the caller. We should not use context.Background() like we do in tests.

Use cmdio.LogString to print to stdout:

```
import "github.com/databricks/cli/libs/cmdio"

cmdio.LogString(ctx, "...")
```
