// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package lakewatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/lakewatch"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

// Spec declares the command for this service.
var Spec = root.CommandSpec{
	Use:     "lakewatch",
	Short:   `*Beta* Manage Lakewatch: security-analytics data sources, detection rules, and per-catalog configuration.`,
	GroupID: "lakewatch",
	Build:   buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `This command is in Beta and may change without notice.

Manage Lakewatch: security-analytics data sources, detection rules, and
  per-catalog configuration.`
	cmd.RunE = root.ReportUnknownSubcommand

	cmd.Annotations = map[string]string{
		"launch_stage":         "PUBLIC_BETA",
		"launch_stage_display": "Beta",
	}

	// Add methods
	cmd.AddCommand(newCreateDatasource())
	cmd.AddCommand(newCreateParser())
	cmd.AddCommand(newCreateParserVersion())
	cmd.AddCommand(newCreateRule())
	cmd.AddCommand(newDeleteDatasource())
	cmd.AddCommand(newDeleteParser())
	cmd.AddCommand(newDeleteParserVersion())
	cmd.AddCommand(newDeleteRule())
	cmd.AddCommand(newGetAlertConfig())
	cmd.AddCommand(newGetBuiltinParser())
	cmd.AddCommand(newGetBuiltinParserVersion())
	cmd.AddCommand(newGetCommonConfig())
	cmd.AddCommand(newGetDatasource())
	cmd.AddCommand(newGetDatasourceConfig())
	cmd.AddCommand(newGetParser())
	cmd.AddCommand(newGetParserVersion())
	cmd.AddCommand(newGetRule())
	cmd.AddCommand(newGetRuleConfig())
	cmd.AddCommand(newListBuiltinParserVersions())
	cmd.AddCommand(newListBuiltinParsers())
	cmd.AddCommand(newListDatasources())
	cmd.AddCommand(newListParserVersions())
	cmd.AddCommand(newListParsers())
	cmd.AddCommand(newListRules())
	cmd.AddCommand(newUpdateAlertConfig())
	cmd.AddCommand(newUpdateCommonConfig())
	cmd.AddCommand(newUpdateDatasource())
	cmd.AddCommand(newUpdateDatasourceConfig())
	cmd.AddCommand(newUpdateRule())
	cmd.AddCommand(newUpdateRuleConfig())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}
}

// start create-datasource command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createDatasourceOverrides []func(
	*cobra.Command,
	*lakewatch.CreateDatasourceRequest,
)

func newCreateDatasource() *cobra.Command {
	cmd := &cobra.Command{}

	var createDatasourceReq lakewatch.CreateDatasourceRequest
	createDatasourceReq.Datasource = lakewatch.Datasource{}
	var createDatasourceJson flags.JsonFlag

	cmd.Flags().Var(&createDatasourceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: map via StringToStringVar: annotations
	// TODO: complex arg: autoloader
	cmd.Flags().Var(&createDatasourceReq.Datasource.ComputeMode, "compute-mode", `Compute mode for the datasource's ingestion job. Supported values: [DS_COMPUTE_PERFORMANCE_OPTIMIZED, DS_COMPUTE_STANDARD]`)
	// TODO: complex arg: created_by
	cmd.Flags().StringVar(&createDatasourceReq.Datasource.Description, "description", createDatasourceReq.Datasource.Description, `Free-form description of the datasource.`)
	// TODO: complex arg: existing_table
	cmd.Flags().StringVar(&createDatasourceReq.Datasource.Name, "name", createDatasourceReq.Datasource.Name, `Resource name of the datasource, in the format datasources/{datasource_id}.`)
	// TODO: complex arg: owner
	// TODO: array: parsers
	cmd.Flags().StringVar(&createDatasourceReq.Datasource.RunAsPrincipal, "run-as-principal", createDatasourceReq.Datasource.RunAsPrincipal, `The user or service principal that the datasource runs as, given as a user email or service principal application ID.`)
	// TODO: complex arg: status
	// TODO: complex arg: unity_catalog_connection
	// TODO: complex arg: updated_by
	cmd.Flags().StringVar(&createDatasourceReq.Datasource.WarehouseId, "warehouse-id", createDatasourceReq.Datasource.WarehouseId, `ID of the SQL warehouse that runs the checks Lakewatch performs to validate a create or update request, such as source and destination access checks.`)

	cmd.Use = "create-datasource CATALOG_NAME SCHEMA_NAME DATASOURCE_NAME SCHEDULE"
	cmd.Short = `*Beta* Create a datasource.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a datasource.

  Creates a new datasource.

  Arguments:
    CATALOG_NAME: Name of the Unity Catalog catalog that contains the datasource. Required
      on create and cannot be changed afterward. Reads return the value as
      supplied; the catalog's current name is reported in
      effective_catalog_name.
    SCHEMA_NAME: Name of the Unity Catalog schema that contains the datasource. Required on
      create and cannot be changed afterward. Reads return the value as
      supplied; the schema's current name is reported in
      effective_schema_name.
    DATASOURCE_NAME: Name of the datasource in Unity Catalog, unique within its schema.
      Required on create and can be updated within the same catalog and schema.
      Get synchronizes this field with the current Unity Catalog name, also
      reported in effective_datasource_name.
    SCHEDULE: Schedule on which the datasource's ingestion job runs. A datasource cannot
      be paused, so a request that sets schedule.enabled to false is
      rejected. Continuous schedules are not supported for Unity Catalog
      connection datasources. For an existing-table datasource, runs must be no
      further apart than the shorter of the source table's
      delta.deletedFileRetentionDuration and delta.logRetentionDuration.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'catalog_name', 'schema_name', 'datasource_name', 'schedule' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createDatasourceJson.Unmarshal(&createDatasourceReq.Datasource)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createDatasourceReq.Datasource.CatalogName = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createDatasourceReq.Datasource.SchemaName = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createDatasourceReq.Datasource.DatasourceName = args[2]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &createDatasourceReq.Datasource.Schedule)
			if err != nil {
				return fmt.Errorf("invalid SCHEDULE: %s", args[3])
			}

		}

		response, err := w.Lakewatch.CreateDatasource(ctx, createDatasourceReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createDatasourceOverrides {
		fn(cmd, &createDatasourceReq)
	}

	return cmd
}

// start create-parser command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createParserOverrides []func(
	*cobra.Command,
	*lakewatch.CreateParserRequest,
)

func newCreateParser() *cobra.Command {
	cmd := &cobra.Command{}

	var createParserReq lakewatch.CreateParserRequest
	createParserReq.Parser = lakewatch.Parser{}
	var createParserJson flags.JsonFlag

	cmd.Flags().Var(&createParserJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: created_by
	cmd.Flags().StringVar(&createParserReq.Parser.Description, "description", createParserReq.Parser.Description, `Description of version 1.`)
	cmd.Flags().StringVar(&createParserReq.Parser.Name, "name", createParserReq.Parser.Name, `Resource name.`)
	// TODO: complex arg: owner

	cmd.Use = "create-parser CATALOG_NAME SCHEMA_NAME PARSER_NAME SPEC"
	cmd.Short = `*Beta* Create a parser.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a parser.

  Creates a parser and its first version in one call.

  Arguments:
    CATALOG_NAME: Name of the parent UC catalog, set on create. Reads return the value as
      supplied; the current UC catalog name is reported in
      effective_catalog_name.
    SCHEMA_NAME: Name of the parent UC schema, set on create. Reads return the value as
      supplied; the current UC schema name is reported in
      effective_schema_name.
    PARSER_NAME: Client-supplied parser name, stored as supplied and fixed at create time.
      Must be a valid UC identifier. A custom parser is not currently a UC
      securable and this field has no rename path.
    SPEC: Spec of version 1, as a YAML document. Accepted only by CreateParser;
      read a version's spec with GetParserVersion.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'catalog_name', 'schema_name', 'parser_name', 'spec' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createParserJson.Unmarshal(&createParserReq.Parser)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createParserReq.Parser.CatalogName = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createParserReq.Parser.SchemaName = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createParserReq.Parser.ParserName = args[2]
		}
		if !cmd.Flags().Changed("json") {
			createParserReq.Parser.Spec = args[3]
		}

		response, err := w.Lakewatch.CreateParser(ctx, createParserReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createParserOverrides {
		fn(cmd, &createParserReq)
	}

	return cmd
}

// start create-parser-version command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createParserVersionOverrides []func(
	*cobra.Command,
	*lakewatch.CreateParserVersionRequest,
)

func newCreateParserVersion() *cobra.Command {
	cmd := &cobra.Command{}

	var createParserVersionReq lakewatch.CreateParserVersionRequest
	createParserVersionReq.ParserVersion = lakewatch.ParserVersion{}
	var createParserVersionJson flags.JsonFlag

	cmd.Flags().Var(&createParserVersionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: created_by
	cmd.Flags().StringVar(&createParserVersionReq.ParserVersion.Description, "description", createParserVersionReq.ParserVersion.Description, `Free-form description of this version, set when the version is created.`)
	cmd.Flags().StringVar(&createParserVersionReq.ParserVersion.Name, "name", createParserVersionReq.ParserVersion.Name, `Resource name.`)

	cmd.Use = "create-parser-version PARENT SPEC"
	cmd.Short = `*Beta* Create a parser version.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a parser version.

  Appends a new immutable version, advancing the parser's latest-version
  pointer. This is how a parser is edited; versions are never changed in place.

  Arguments:
    PARENT: Resource name of the parser to append a version to. Format:
      parsers/{parser_id}.
    SPEC: The field definitions for this log source, as a YAML document. Validated
      on write, so an invalid spec is never stored.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'spec' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createParserVersionJson.Unmarshal(&createParserVersionReq.ParserVersion)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		createParserVersionReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			createParserVersionReq.ParserVersion.Spec = args[1]
		}

		response, err := w.Lakewatch.CreateParserVersion(ctx, createParserVersionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createParserVersionOverrides {
		fn(cmd, &createParserVersionReq)
	}

	return cmd
}

// start create-rule command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createRuleOverrides []func(
	*cobra.Command,
	*lakewatch.CreateRuleRequest,
)

func newCreateRule() *cobra.Command {
	cmd := &cobra.Command{}

	var createRuleReq lakewatch.CreateRuleRequest
	createRuleReq.Rule = lakewatch.Rule{}
	var createRuleJson flags.JsonFlag

	cmd.Flags().Var(&createRuleJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: map via StringToStringVar: annotations
	// TODO: complex arg: batch
	cmd.Flags().StringVar(&createRuleReq.Rule.Category, "category", createRuleReq.Rule.Category, `Free-form rule category.`)
	cmd.Flags().StringVar(&createRuleReq.Rule.ContentResourceId, "content-resource-id", createRuleReq.Rule.ContentResourceId, `Immutable UUIDv4 identifying the authored content resource installed as this rule.`)
	// TODO: complex arg: created_by
	cmd.Flags().Var(&createRuleReq.Rule.Deduplication, "deduplication", `Signal deduplication mode. Supported values: [DEDUPLICATION_DEFAULT, DEDUPLICATION_DISABLED, DEDUPLICATION_ENABLED]`)
	cmd.Flags().StringVar(&createRuleReq.Rule.DeduplicationTimeWindow, "deduplication-time-window", createRuleReq.Rule.DeduplicationTimeWindow, `Time window for deduplication, such as 1h, 24h, 7d, or automatic.`)
	cmd.Flags().StringVar(&createRuleReq.Rule.Description, "description", createRuleReq.Rule.Description, `Free-form description.`)
	// TODO: array: mitre
	cmd.Flags().StringVar(&createRuleReq.Rule.Name, "name", createRuleReq.Rule.Name, `Resource name, in the form rules/{rule_id}.`)
	cmd.Flags().StringVar(&createRuleReq.Rule.Objective, "objective", createRuleReq.Rule.Objective, `Long-form description of what this rule detects.`)
	// TODO: complex arg: original_content_pack
	// TODO: complex arg: owner
	// TODO: array: references
	cmd.Flags().StringVar(&createRuleReq.Rule.RunAsPrincipal, "run-as-principal", createRuleReq.Rule.RunAsPrincipal, `The principal whose identity this rule's job runs as, given as a user email or service-principal application ID.`)
	cmd.Flags().StringVar(&createRuleReq.Rule.Runbook, "runbook", createRuleReq.Rule.Runbook, `Long-form operational guidance carried from the authored rule metadata.`)
	// TODO: complex arg: schedule
	// TODO: complex arg: status
	// TODO: complex arg: streaming
	var suppressUntilTimeParam string
	cmd.Flags().StringVar(&suppressUntilTimeParam, "suppress-until-time", suppressUntilTimeParam, `If set, this rule's signals do not open new alerts until the given time.`)
	// TODO: complex arg: timerange
	// TODO: complex arg: updated_by

	cmd.Use = "create-rule CATALOG_NAME SCHEMA_NAME RULE_NAME SEVERITY FIDELITY OUTPUT"
	cmd.Short = `*Beta* Create a rule.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a rule.

  Creates a new rule.

  Arguments:
    CATALOG_NAME: Name of the Unity Catalog catalog that contains the rule. Required. Cannot
      be changed after creation. Reads return the value as supplied; the
      catalog's current name is reported in effective_catalog_name.
    SCHEMA_NAME: Name of the Unity Catalog schema that contains the rule. Required. Cannot
      be changed after creation. Reads return the value as supplied; the
      schema's current name is reported in effective_schema_name.
    RULE_NAME: Name of the rule within its Unity Catalog schema. Required on create; can
      be updated within the same catalog and schema. Get synchronizes this field
      with the current Unity Catalog name, also reported in
      effective_rule_name.
    SEVERITY: Free-form severity label, such as Informational, Medium, High, or
      Critical.
    FIDELITY: Signal quality for this rule.
      Supported values: [FIDELITY_DEVELOPMENT, FIDELITY_HIGH, FIDELITY_INVESTIGATIVE, FIDELITY_MEDIUM]
    OUTPUT: What to record in each signal.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'catalog_name', 'schema_name', 'rule_name', 'severity', 'fidelity', 'output' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(6)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createRuleJson.Unmarshal(&createRuleReq.Rule)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createRuleReq.Rule.CatalogName = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createRuleReq.Rule.SchemaName = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createRuleReq.Rule.RuleName = args[2]
		}
		if !cmd.Flags().Changed("json") {
			createRuleReq.Rule.Severity = args[3]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[4], &createRuleReq.Rule.Fidelity)
			if err != nil {
				return fmt.Errorf("invalid FIDELITY: %s", args[4])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[5], &createRuleReq.Rule.Output)
			if err != nil {
				return fmt.Errorf("invalid OUTPUT: %s", args[5])
			}

		}

		if suppressUntilTimeParam != "" {
			suppressUntilTimeBytes := []byte(fmt.Sprintf("\"%s\"", suppressUntilTimeParam))
			var suppressUntilTimeField sdktime.Time
			err = json.Unmarshal(suppressUntilTimeBytes, &suppressUntilTimeField)
			if err != nil {
				return fmt.Errorf("invalid SUPPRESS_UNTIL_TIME: %s", suppressUntilTimeParam)
			}
			createRuleReq.Rule.SuppressUntilTime = &suppressUntilTimeField
		}

		response, err := w.Lakewatch.CreateRule(ctx, createRuleReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createRuleOverrides {
		fn(cmd, &createRuleReq)
	}

	return cmd
}

// start delete-datasource command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDatasourceOverrides []func(
	*cobra.Command,
	*lakewatch.DeleteDatasourceRequest,
)

func newDeleteDatasource() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteDatasourceReq lakewatch.DeleteDatasourceRequest

	cmd.Use = "delete-datasource NAME"
	cmd.Short = `*Beta* Delete a datasource.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a datasource.

  Deletes a datasource by its resource name.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteDatasourceReq.Name = args[0]

		err = w.Lakewatch.DeleteDatasource(ctx, deleteDatasourceReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDatasourceOverrides {
		fn(cmd, &deleteDatasourceReq)
	}

	return cmd
}

// start delete-parser command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteParserOverrides []func(
	*cobra.Command,
	*lakewatch.DeleteParserRequest,
)

func newDeleteParser() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteParserReq lakewatch.DeleteParserRequest

	cmd.Use = "delete-parser NAME"
	cmd.Short = `*Beta* Delete a parser.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a parser.

  Deletes a parser and all of its versions. Refused while any data source
  references it.

  Arguments:
    NAME: Resource name of the parser. Format: parsers/{parser_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteParserReq.Name = args[0]

		err = w.Lakewatch.DeleteParser(ctx, deleteParserReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteParserOverrides {
		fn(cmd, &deleteParserReq)
	}

	return cmd
}

// start delete-parser-version command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteParserVersionOverrides []func(
	*cobra.Command,
	*lakewatch.DeleteParserVersionRequest,
)

func newDeleteParserVersion() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteParserVersionReq lakewatch.DeleteParserVersionRequest

	cmd.Use = "delete-parser-version NAME"
	cmd.Short = `*Beta* Delete a parser version.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a parser version.

  Deletes one version of a parser. Returns NOT_FOUND when the parser or the
  version does not exist, and a conflict (HTTP 409) only while a data source
  references the version. Deleting the last version leaves the parser in place
  with no versions.

  Arguments:
    NAME: Resource name of the version. Format:
      parsers/{parser_id}/versions/{version}, where {version} is a version
      number written without sign or leading zeros, such as 3. Unlike
      GetParserVersion, the alias latest and forms such as 007 or +3 are
      rejected, so a delete always names one fixed version.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteParserVersionReq.Name = args[0]

		err = w.Lakewatch.DeleteParserVersion(ctx, deleteParserVersionReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteParserVersionOverrides {
		fn(cmd, &deleteParserVersionReq)
	}

	return cmd
}

// start delete-rule command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteRuleOverrides []func(
	*cobra.Command,
	*lakewatch.DeleteRuleRequest,
)

func newDeleteRule() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteRuleReq lakewatch.DeleteRuleRequest

	cmd.Use = "delete-rule NAME"
	cmd.Short = `*Beta* Delete a rule.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a rule.

  Deletes a rule by its resource name.

  Arguments:
    NAME: Resource name of the rule to delete. Format: rules/{rule_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteRuleReq.Name = args[0]

		err = w.Lakewatch.DeleteRule(ctx, deleteRuleReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteRuleOverrides {
		fn(cmd, &deleteRuleReq)
	}

	return cmd
}

// start get-alert-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getAlertConfigOverrides []func(
	*cobra.Command,
	*lakewatch.GetAlertConfigRequest,
)

func newGetAlertConfig() *cobra.Command {
	cmd := &cobra.Command{}

	var getAlertConfigReq lakewatch.GetAlertConfigRequest

	cmd.Use = "get-alert-config NAME"
	cmd.Short = `*Beta* Get the alert config for a catalog.`
	cmd.Long = `This command is in Beta and may change without notice.

Get the alert config for a catalog.

  Returns the alert delivery configuration for a catalog.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getAlertConfigReq.Name = args[0]

		response, err := w.Lakewatch.GetAlertConfig(ctx, getAlertConfigReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getAlertConfigOverrides {
		fn(cmd, &getAlertConfigReq)
	}

	return cmd
}

// start get-builtin-parser command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getBuiltinParserOverrides []func(
	*cobra.Command,
	*lakewatch.GetBuiltinParserRequest,
)

func newGetBuiltinParser() *cobra.Command {
	cmd := &cobra.Command{}

	var getBuiltinParserReq lakewatch.GetBuiltinParserRequest

	cmd.Use = "get-builtin-parser NAME"
	cmd.Short = `*Beta* Get a built-in parser.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a built-in parser.

  Returns a built-in parser's identity and latest-version pointer. Does not
  return the spec.

  Arguments:
    NAME: Resource name. Format: builtin-parsers/{parser_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getBuiltinParserReq.Name = args[0]

		response, err := w.Lakewatch.GetBuiltinParser(ctx, getBuiltinParserReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getBuiltinParserOverrides {
		fn(cmd, &getBuiltinParserReq)
	}

	return cmd
}

// start get-builtin-parser-version command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getBuiltinParserVersionOverrides []func(
	*cobra.Command,
	*lakewatch.GetBuiltinParserVersionRequest,
)

func newGetBuiltinParserVersion() *cobra.Command {
	cmd := &cobra.Command{}

	var getBuiltinParserVersionReq lakewatch.GetBuiltinParserVersionRequest

	cmd.Use = "get-builtin-parser-version NAME"
	cmd.Short = `*Beta* Get a built-in parser version.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a built-in parser version.

  Returns one version of a built-in parser, including its spec.

  Arguments:
    NAME: Resource name of the version. Format:
      builtin-parsers/{parser_id}/versions/{version}, where {version} is
      either a version number or the alias latest.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getBuiltinParserVersionReq.Name = args[0]

		response, err := w.Lakewatch.GetBuiltinParserVersion(ctx, getBuiltinParserVersionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getBuiltinParserVersionOverrides {
		fn(cmd, &getBuiltinParserVersionReq)
	}

	return cmd
}

// start get-common-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getCommonConfigOverrides []func(
	*cobra.Command,
	*lakewatch.GetCommonConfigRequest,
)

func newGetCommonConfig() *cobra.Command {
	cmd := &cobra.Command{}

	var getCommonConfigReq lakewatch.GetCommonConfigRequest

	cmd.Use = "get-common-config NAME"
	cmd.Short = `*Beta* Get the common config for a catalog.`
	cmd.Long = `This command is in Beta and may change without notice.

Get the common config for a catalog.

  Returns the common Lakewatch configuration for a catalog.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getCommonConfigReq.Name = args[0]

		response, err := w.Lakewatch.GetCommonConfig(ctx, getCommonConfigReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getCommonConfigOverrides {
		fn(cmd, &getCommonConfigReq)
	}

	return cmd
}

// start get-datasource command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDatasourceOverrides []func(
	*cobra.Command,
	*lakewatch.GetDatasourceRequest,
)

func newGetDatasource() *cobra.Command {
	cmd := &cobra.Command{}

	var getDatasourceReq lakewatch.GetDatasourceRequest

	cmd.Use = "get-datasource NAME"
	cmd.Short = `*Beta* Get a datasource.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a datasource.

  Returns a single datasource by its resource name.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getDatasourceReq.Name = args[0]

		response, err := w.Lakewatch.GetDatasource(ctx, getDatasourceReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDatasourceOverrides {
		fn(cmd, &getDatasourceReq)
	}

	return cmd
}

// start get-datasource-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDatasourceConfigOverrides []func(
	*cobra.Command,
	*lakewatch.GetDatasourceConfigRequest,
)

func newGetDatasourceConfig() *cobra.Command {
	cmd := &cobra.Command{}

	var getDatasourceConfigReq lakewatch.GetDatasourceConfigRequest

	cmd.Use = "get-datasource-config NAME"
	cmd.Short = `*Beta* Get the datasource config for a catalog.`
	cmd.Long = `This command is in Beta and may change without notice.

Get the datasource config for a catalog.

  Returns the datasource ingestion configuration for a catalog.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getDatasourceConfigReq.Name = args[0]

		response, err := w.Lakewatch.GetDatasourceConfig(ctx, getDatasourceConfigReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDatasourceConfigOverrides {
		fn(cmd, &getDatasourceConfigReq)
	}

	return cmd
}

// start get-parser command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getParserOverrides []func(
	*cobra.Command,
	*lakewatch.GetParserRequest,
)

func newGetParser() *cobra.Command {
	cmd := &cobra.Command{}

	var getParserReq lakewatch.GetParserRequest

	cmd.Use = "get-parser NAME"
	cmd.Short = `*Beta* Get a parser.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a parser.

  Returns a parser's identity and latest-version pointer. Does not return the
  spec; read a spec with GetParserVersion.

  Arguments:
    NAME: Resource name of the parser. Format: parsers/{parser_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getParserReq.Name = args[0]

		response, err := w.Lakewatch.GetParser(ctx, getParserReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getParserOverrides {
		fn(cmd, &getParserReq)
	}

	return cmd
}

// start get-parser-version command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getParserVersionOverrides []func(
	*cobra.Command,
	*lakewatch.GetParserVersionRequest,
)

func newGetParserVersion() *cobra.Command {
	cmd := &cobra.Command{}

	var getParserVersionReq lakewatch.GetParserVersionRequest

	cmd.Use = "get-parser-version NAME"
	cmd.Short = `*Beta* Get a parser version.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a parser version.

  Returns one version of a parser, including its spec.

  Arguments:
    NAME: Resource name of the version. Format:
      parsers/{parser_id}/versions/{version}, where {version} is either a
      version number or the alias latest. latest is NOT_FOUND when every
      version of the parser has been deleted.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getParserVersionReq.Name = args[0]

		response, err := w.Lakewatch.GetParserVersion(ctx, getParserVersionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getParserVersionOverrides {
		fn(cmd, &getParserVersionReq)
	}

	return cmd
}

// start get-rule command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getRuleOverrides []func(
	*cobra.Command,
	*lakewatch.GetRuleRequest,
)

func newGetRule() *cobra.Command {
	cmd := &cobra.Command{}

	var getRuleReq lakewatch.GetRuleRequest

	cmd.Use = "get-rule NAME"
	cmd.Short = `*Beta* Get a rule.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a rule.

  Returns a single rule by its resource name.

  Arguments:
    NAME: Full resource name of the detection rule, in the form rules/{rule_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getRuleReq.Name = args[0]

		response, err := w.Lakewatch.GetRule(ctx, getRuleReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getRuleOverrides {
		fn(cmd, &getRuleReq)
	}

	return cmd
}

// start get-rule-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getRuleConfigOverrides []func(
	*cobra.Command,
	*lakewatch.GetRuleConfigRequest,
)

func newGetRuleConfig() *cobra.Command {
	cmd := &cobra.Command{}

	var getRuleConfigReq lakewatch.GetRuleConfigRequest

	cmd.Use = "get-rule-config NAME"
	cmd.Short = `*Beta* Get the rule config for a catalog.`
	cmd.Long = `This command is in Beta and may change without notice.

Get the rule config for a catalog.

  Returns the detection-rule configuration for a catalog.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getRuleConfigReq.Name = args[0]

		response, err := w.Lakewatch.GetRuleConfig(ctx, getRuleConfigReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getRuleConfigOverrides {
		fn(cmd, &getRuleConfigReq)
	}

	return cmd
}

// start list-builtin-parser-versions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listBuiltinParserVersionsOverrides []func(
	*cobra.Command,
	*lakewatch.ListBuiltinParserVersionsRequest,
)

func newListBuiltinParserVersions() *cobra.Command {
	cmd := &cobra.Command{}

	var listBuiltinParserVersionsReq lakewatch.ListBuiltinParserVersionsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listBuiltinParserVersionsLimit int

	cmd.Flags().IntVar(&listBuiltinParserVersionsReq.PageSize, "page-size", listBuiltinParserVersionsReq.PageSize, `Maximum number of results to return in one page.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listBuiltinParserVersionsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listBuiltinParserVersionsReq.PageToken, "page-token", listBuiltinParserVersionsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-builtin-parser-versions PARENT"
	cmd.Short = `*Beta* List built-in parser versions.`
	cmd.Long = `This command is in Beta and may change without notice.

List built-in parser versions.

  Lists a built-in parser's published versions, oldest first (ascending
  1..resource_revision), without their specs.

  Arguments:
    PARENT: Resource name of the built-in parser whose versions are listed. Format:
      builtin-parsers/{parser_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listBuiltinParserVersionsReq.Parent = args[0]

		response := w.Lakewatch.ListBuiltinParserVersions(ctx, listBuiltinParserVersionsReq)
		if listBuiltinParserVersionsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listBuiltinParserVersionsLimit)
		}
		if listBuiltinParserVersionsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listBuiltinParserVersionsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listBuiltinParserVersionsOverrides {
		fn(cmd, &listBuiltinParserVersionsReq)
	}

	return cmd
}

// start list-builtin-parsers command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listBuiltinParsersOverrides []func(
	*cobra.Command,
	*lakewatch.ListBuiltinParsersRequest,
)

func newListBuiltinParsers() *cobra.Command {
	cmd := &cobra.Command{}

	var listBuiltinParsersReq lakewatch.ListBuiltinParsersRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listBuiltinParsersLimit int

	cmd.Flags().StringVar(&listBuiltinParsersReq.Filter, "filter", listBuiltinParsersReq.Filter, `AIP-160 filter expression.`)
	cmd.Flags().IntVar(&listBuiltinParsersReq.PageSize, "page-size", listBuiltinParsersReq.PageSize, `Maximum number of results to return in one page.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listBuiltinParsersLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listBuiltinParsersReq.PageToken, "page-token", listBuiltinParsersReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-builtin-parsers"
	cmd.Short = `*Beta* List built-in parsers.`
	cmd.Long = `This command is in Beta and may change without notice.

List built-in parsers.

  Lists the built-in parsers Databricks publishes, without their specs.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Lakewatch.ListBuiltinParsers(ctx, listBuiltinParsersReq)
		if listBuiltinParsersLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listBuiltinParsersLimit)
		}
		if listBuiltinParsersLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listBuiltinParsersLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listBuiltinParsersOverrides {
		fn(cmd, &listBuiltinParsersReq)
	}

	return cmd
}

// start list-datasources command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listDatasourcesOverrides []func(
	*cobra.Command,
	*lakewatch.ListDatasourcesRequest,
)

func newListDatasources() *cobra.Command {
	cmd := &cobra.Command{}

	var listDatasourcesReq lakewatch.ListDatasourcesRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listDatasourcesLimit int

	cmd.Flags().StringVar(&listDatasourcesReq.Filter, "filter", listDatasourcesReq.Filter, `Filter expression that restricts the results to datasources with a given name, for example datasource_name = "my_source".`)
	cmd.Flags().IntVar(&listDatasourcesReq.PageSize, "page-size", listDatasourcesReq.PageSize, `Maximum number of results to return in one page.`)
	cmd.Flags().StringVar(&listDatasourcesReq.SchemaName, "schema-name", listDatasourcesReq.SchemaName, `Name of a Unity Catalog schema in catalog_name.`)
	cmd.Flags().Var(&listDatasourcesReq.View, "view", `Which fields to return for each datasource. Supported values: [DATASOURCE_VIEW_BASIC, DATASOURCE_VIEW_FULL]`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listDatasourcesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listDatasourcesReq.PageToken, "page-token", listDatasourcesReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-datasources CATALOG_NAME"
	cmd.Short = `*Beta* List datasources.`
	cmd.Long = `This command is in Beta and may change without notice.

List datasources.

  Lists the datasources in a Unity Catalog catalog, optionally narrowed to one
  schema or filtered by name. Results are paginated. Datasources that are still
  being created or deleted are not listed.

  Arguments:
    CATALOG_NAME: Name of the Unity Catalog catalog to list datasources from. Required; each
      request lists the datasources of a single catalog.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listDatasourcesReq.CatalogName = args[0]

		response := w.Lakewatch.ListDatasources(ctx, listDatasourcesReq)
		if listDatasourcesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listDatasourcesLimit)
		}
		if listDatasourcesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listDatasourcesLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listDatasourcesOverrides {
		fn(cmd, &listDatasourcesReq)
	}

	return cmd
}

// start list-parser-versions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listParserVersionsOverrides []func(
	*cobra.Command,
	*lakewatch.ListParserVersionsRequest,
)

func newListParserVersions() *cobra.Command {
	cmd := &cobra.Command{}

	var listParserVersionsReq lakewatch.ListParserVersionsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listParserVersionsLimit int

	cmd.Flags().IntVar(&listParserVersionsReq.PageSize, "page-size", listParserVersionsReq.PageSize, `Maximum number of results to return in one page.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listParserVersionsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listParserVersionsReq.PageToken, "page-token", listParserVersionsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-parser-versions PARENT"
	cmd.Short = `*Beta* List parser versions.`
	cmd.Long = `This command is in Beta and may change without notice.

List parser versions.

  Lists a parser's versions, oldest first, without their specs.

  Arguments:
    PARENT: Resource name of the parser whose versions are listed. Format:
      parsers/{parser_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listParserVersionsReq.Parent = args[0]

		response := w.Lakewatch.ListParserVersions(ctx, listParserVersionsReq)
		if listParserVersionsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listParserVersionsLimit)
		}
		if listParserVersionsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listParserVersionsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listParserVersionsOverrides {
		fn(cmd, &listParserVersionsReq)
	}

	return cmd
}

// start list-parsers command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listParsersOverrides []func(
	*cobra.Command,
	*lakewatch.ListParsersRequest,
)

func newListParsers() *cobra.Command {
	cmd := &cobra.Command{}

	var listParsersReq lakewatch.ListParsersRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listParsersLimit int

	cmd.Flags().StringVar(&listParsersReq.Filter, "filter", listParsersReq.Filter, `AIP-160 filter expression.`)
	cmd.Flags().IntVar(&listParsersReq.PageSize, "page-size", listParsersReq.PageSize, `Maximum number of results to return in one page.`)
	cmd.Flags().StringVar(&listParsersReq.SchemaName, "schema-name", listParsersReq.SchemaName, `Name of the parent UC schema.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listParsersLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listParsersReq.PageToken, "page-token", listParsersReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-parsers CATALOG_NAME"
	cmd.Short = `*Beta* List parsers.`
	cmd.Long = `This command is in Beta and may change without notice.

List parsers.

  Lists parsers in a catalog, without their specs.

  Arguments:
    CATALOG_NAME: Name of the parent UC catalog to list parsers from. Required: List is
      scoped to a single catalog, mirroring the Unity Catalog tables/functions
      APIs, which take catalog_name as a request field (bound to a query
      parameter) rather than a catalog path segment.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listParsersReq.CatalogName = args[0]

		response := w.Lakewatch.ListParsers(ctx, listParsersReq)
		if listParsersLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listParsersLimit)
		}
		if listParsersLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listParsersLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listParsersOverrides {
		fn(cmd, &listParsersReq)
	}

	return cmd
}

// start list-rules command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listRulesOverrides []func(
	*cobra.Command,
	*lakewatch.ListRulesRequest,
)

func newListRules() *cobra.Command {
	cmd := &cobra.Command{}

	var listRulesReq lakewatch.ListRulesRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listRulesLimit int

	cmd.Flags().StringVar(&listRulesReq.Filter, "filter", listRulesReq.Filter, `Filter expression that restricts the results, such as rule_name = "my_rule" to return only the rule with that name.`)
	cmd.Flags().IntVar(&listRulesReq.PageSize, "page-size", listRulesReq.PageSize, `Maximum number of rules to return.`)
	cmd.Flags().StringVar(&listRulesReq.SchemaName, "schema-name", listRulesReq.SchemaName, `Name of a Unity Catalog schema in catalog_name.`)
	cmd.Flags().Var(&listRulesReq.View, "view", `How much of each rule to return. Supported values: [RULE_VIEW_BASIC, RULE_VIEW_FULL]`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listRulesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listRulesReq.PageToken, "page-token", listRulesReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-rules CATALOG_NAME"
	cmd.Short = `*Beta* List rules.`
	cmd.Long = `This command is in Beta and may change without notice.

List rules.

  Lists rules, with optional filtering and pagination.

  Arguments:
    CATALOG_NAME: Name of the Unity Catalog catalog to list rules from. Required.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listRulesReq.CatalogName = args[0]

		response := w.Lakewatch.ListRules(ctx, listRulesReq)
		if listRulesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listRulesLimit)
		}
		if listRulesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listRulesLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listRulesOverrides {
		fn(cmd, &listRulesReq)
	}

	return cmd
}

// start update-alert-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateAlertConfigOverrides []func(
	*cobra.Command,
	*lakewatch.UpdateAlertConfigRequest,
)

func newUpdateAlertConfig() *cobra.Command {
	cmd := &cobra.Command{}

	var updateAlertConfigReq lakewatch.UpdateAlertConfigRequest
	updateAlertConfigReq.AlertConfig = lakewatch.AlertConfig{}
	var updateAlertConfigJson flags.JsonFlag

	cmd.Flags().Var(&updateAlertConfigJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateAlertConfigReq.Etag, "etag", updateAlertConfigReq.Etag, `Optimistic-concurrency precondition: if set and it does not match the resource's current etag, the update is rejected as a conflict (HTTP 409).`)
	cmd.Flags().StringVar(&updateAlertConfigReq.AlertConfig.DefaultSecurityAlertCollationWindow, "default-security-alert-collation-window", updateAlertConfigReq.AlertConfig.DefaultSecurityAlertCollationWindow, `Window for collating Signals into security alerts.`)
	cmd.Flags().StringVar(&updateAlertConfigReq.AlertConfig.DefaultSystemAlertCollationWindow, "default-system-alert-collation-window", updateAlertConfigReq.AlertConfig.DefaultSystemAlertCollationWindow, `Window for collating SystemEvents into system alerts.`)
	cmd.Flags().StringVar(&updateAlertConfigReq.AlertConfig.Name, "name", updateAlertConfigReq.AlertConfig.Name, ``)
	// TODO: complex arg: security_alert_export_config
	// TODO: complex arg: system_alert_export_config
	// TODO: complex arg: updated_by

	cmd.Use = "update-alert-config NAME UPDATE_MASK"
	cmd.Short = `*Beta* Update the alert config for a catalog.`
	cmd.Long = `This command is in Beta and may change without notice.

Update the alert config for a catalog.

  Updates the alert delivery configuration for a catalog.

  Arguments:
    NAME:
    UPDATE_MASK: The field mask must be a single string, with multiple fields separated by
      commas (no spaces). The field path is relative to the resource object,
      using a dot (.) to navigate sub-fields (e.g., author.given_name).
      Specification of elements in sequence or map fields is not allowed, as
      only the entire collection field can be specified. Field names must
      exactly match the resource field names.

      A field mask of * indicates full replacement. It’s recommended to
      always explicitly list the fields being updated and avoid using *
      wildcards, as it can lead to unintended results if the API changes in the
      future.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateAlertConfigJson.Unmarshal(&updateAlertConfigReq.AlertConfig)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateAlertConfigReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateAlertConfigReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		response, err := w.Lakewatch.UpdateAlertConfig(ctx, updateAlertConfigReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateAlertConfigOverrides {
		fn(cmd, &updateAlertConfigReq)
	}

	return cmd
}

// start update-common-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateCommonConfigOverrides []func(
	*cobra.Command,
	*lakewatch.UpdateCommonConfigRequest,
)

func newUpdateCommonConfig() *cobra.Command {
	cmd := &cobra.Command{}

	var updateCommonConfigReq lakewatch.UpdateCommonConfigRequest
	updateCommonConfigReq.CommonConfig = lakewatch.CommonConfig{}
	var updateCommonConfigJson flags.JsonFlag

	cmd.Flags().Var(&updateCommonConfigJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateCommonConfigReq.Etag, "etag", updateCommonConfigReq.Etag, `Optimistic-concurrency precondition: if set and it does not match the resource's current etag, the update is rejected as a conflict (HTTP 409).`)
	cmd.Flags().StringVar(&updateCommonConfigReq.CommonConfig.Name, "name", updateCommonConfigReq.CommonConfig.Name, ``)
	cmd.Flags().BoolVar(&updateCommonConfigReq.CommonConfig.PerResourcePermissionsEnabled, "per-resource-permissions-enabled", updateCommonConfigReq.CommonConfig.PerResourcePermissionsEnabled, `Whether users may grant permissions on individual resources rather than only at the Folder level.`)
	// TODO: complex arg: updated_by

	cmd.Use = "update-common-config NAME UPDATE_MASK"
	cmd.Short = `*Beta* Update the common config for a catalog.`
	cmd.Long = `This command is in Beta and may change without notice.

Update the common config for a catalog.

  Updates the common Lakewatch configuration for a catalog.

  Arguments:
    NAME:
    UPDATE_MASK: The field mask must be a single string, with multiple fields separated by
      commas (no spaces). The field path is relative to the resource object,
      using a dot (.) to navigate sub-fields (e.g., author.given_name).
      Specification of elements in sequence or map fields is not allowed, as
      only the entire collection field can be specified. Field names must
      exactly match the resource field names.

      A field mask of * indicates full replacement. It’s recommended to
      always explicitly list the fields being updated and avoid using *
      wildcards, as it can lead to unintended results if the API changes in the
      future.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateCommonConfigJson.Unmarshal(&updateCommonConfigReq.CommonConfig)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateCommonConfigReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateCommonConfigReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		response, err := w.Lakewatch.UpdateCommonConfig(ctx, updateCommonConfigReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateCommonConfigOverrides {
		fn(cmd, &updateCommonConfigReq)
	}

	return cmd
}

// start update-datasource command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateDatasourceOverrides []func(
	*cobra.Command,
	*lakewatch.UpdateDatasourceRequest,
)

func newUpdateDatasource() *cobra.Command {
	cmd := &cobra.Command{}

	var updateDatasourceReq lakewatch.UpdateDatasourceRequest
	updateDatasourceReq.Datasource = lakewatch.Datasource{}
	var updateDatasourceJson flags.JsonFlag

	cmd.Flags().Var(&updateDatasourceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateDatasourceReq.Etag, "etag", updateDatasourceReq.Etag, `Optimistic-concurrency precondition: if set to a non-empty value that does not match the resource's current etag, the update is rejected as a conflict (HTTP 409).`)
	// TODO: map via StringToStringVar: annotations
	// TODO: complex arg: autoloader
	cmd.Flags().Var(&updateDatasourceReq.Datasource.ComputeMode, "compute-mode", `Compute mode for the datasource's ingestion job. Supported values: [DS_COMPUTE_PERFORMANCE_OPTIMIZED, DS_COMPUTE_STANDARD]`)
	// TODO: complex arg: created_by
	cmd.Flags().StringVar(&updateDatasourceReq.Datasource.Description, "description", updateDatasourceReq.Datasource.Description, `Free-form description of the datasource.`)
	// TODO: complex arg: existing_table
	cmd.Flags().StringVar(&updateDatasourceReq.Datasource.Name, "name", updateDatasourceReq.Datasource.Name, `Resource name of the datasource, in the format datasources/{datasource_id}.`)
	// TODO: complex arg: owner
	// TODO: array: parsers
	cmd.Flags().StringVar(&updateDatasourceReq.Datasource.RunAsPrincipal, "run-as-principal", updateDatasourceReq.Datasource.RunAsPrincipal, `The user or service principal that the datasource runs as, given as a user email or service principal application ID.`)
	// TODO: complex arg: status
	// TODO: complex arg: unity_catalog_connection
	// TODO: complex arg: updated_by
	cmd.Flags().StringVar(&updateDatasourceReq.Datasource.WarehouseId, "warehouse-id", updateDatasourceReq.Datasource.WarehouseId, `ID of the SQL warehouse that runs the checks Lakewatch performs to validate a create or update request, such as source and destination access checks.`)

	cmd.Use = "update-datasource NAME UPDATE_MASK CATALOG_NAME SCHEMA_NAME DATASOURCE_NAME SCHEDULE"
	cmd.Short = `*Beta* Update a datasource.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a datasource.

  Updates an existing datasource.

  Arguments:
    NAME: Resource name of the datasource, in the format
      datasources/{datasource_id}. Unlike the datasource's Unity Catalog
      location (catalog_name, schema_name, and datasource_name), the name
      does not change when the datasource or its catalog or schema is renamed in
      Unity Catalog.
    UPDATE_MASK: The field mask must be a single string, with multiple fields separated by
      commas (no spaces). The field path is relative to the resource object,
      using a dot (.) to navigate sub-fields (e.g., author.given_name).
      Specification of elements in sequence or map fields is not allowed, as
      only the entire collection field can be specified. Field names must
      exactly match the resource field names.

      A field mask of * indicates full replacement. It’s recommended to
      always explicitly list the fields being updated and avoid using *
      wildcards, as it can lead to unintended results if the API changes in the
      future.
    CATALOG_NAME: Name of the Unity Catalog catalog that contains the datasource. Required
      on create and cannot be changed afterward. Reads return the value as
      supplied; the catalog's current name is reported in
      effective_catalog_name.
    SCHEMA_NAME: Name of the Unity Catalog schema that contains the datasource. Required on
      create and cannot be changed afterward. Reads return the value as
      supplied; the schema's current name is reported in
      effective_schema_name.
    DATASOURCE_NAME: Name of the datasource in Unity Catalog, unique within its schema.
      Required on create and can be updated within the same catalog and schema.
      Get synchronizes this field with the current Unity Catalog name, also
      reported in effective_datasource_name.
    SCHEDULE: Schedule on which the datasource's ingestion job runs. A datasource cannot
      be paused, so a request that sets schedule.enabled to false is
      rejected. Continuous schedules are not supported for Unity Catalog
      connection datasources. For an existing-table datasource, runs must be no
      further apart than the shorter of the source table's
      delta.deletedFileRetentionDuration and delta.logRetentionDuration.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'catalog_name', 'schema_name', 'datasource_name', 'schedule' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(6)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateDatasourceJson.Unmarshal(&updateDatasourceReq.Datasource)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateDatasourceReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateDatasourceReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			updateDatasourceReq.Datasource.CatalogName = args[2]
		}
		if !cmd.Flags().Changed("json") {
			updateDatasourceReq.Datasource.SchemaName = args[3]
		}
		if !cmd.Flags().Changed("json") {
			updateDatasourceReq.Datasource.DatasourceName = args[4]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[5], &updateDatasourceReq.Datasource.Schedule)
			if err != nil {
				return fmt.Errorf("invalid SCHEDULE: %s", args[5])
			}

		}

		response, err := w.Lakewatch.UpdateDatasource(ctx, updateDatasourceReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateDatasourceOverrides {
		fn(cmd, &updateDatasourceReq)
	}

	return cmd
}

// start update-datasource-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateDatasourceConfigOverrides []func(
	*cobra.Command,
	*lakewatch.UpdateDatasourceConfigRequest,
)

func newUpdateDatasourceConfig() *cobra.Command {
	cmd := &cobra.Command{}

	var updateDatasourceConfigReq lakewatch.UpdateDatasourceConfigRequest
	updateDatasourceConfigReq.DatasourceConfig = lakewatch.DatasourceConfig{}
	var updateDatasourceConfigJson flags.JsonFlag

	cmd.Flags().Var(&updateDatasourceConfigJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateDatasourceConfigReq.Etag, "etag", updateDatasourceConfigReq.Etag, `Optimistic-concurrency precondition: if set and it does not match the resource's current etag, the update is rejected as a conflict (HTTP 409).`)
	cmd.Flags().Var(&updateDatasourceConfigReq.DatasourceConfig.DefaultComputeMode, "default-compute-mode", `Default compute mode applied to datasource ingestion jobs when the individual datasource doesn't override it. Supported values: [DS_COMPUTE_PERFORMANCE_OPTIMIZED, DS_COMPUTE_STANDARD]`)
	cmd.Flags().StringVar(&updateDatasourceConfigReq.DatasourceConfig.DefaultFolder, "default-folder", updateDatasourceConfigReq.DatasourceConfig.DefaultFolder, `Schema name of the registered folder (in this catalog) that a new datasource is placed in when the request does not specify a schema.`)
	cmd.Flags().StringVar(&updateDatasourceConfigReq.DatasourceConfig.Name, "name", updateDatasourceConfigReq.DatasourceConfig.Name, ``)
	// TODO: complex arg: updated_by

	cmd.Use = "update-datasource-config NAME UPDATE_MASK"
	cmd.Short = `*Beta* Update the datasource config for a catalog.`
	cmd.Long = `This command is in Beta and may change without notice.

Update the datasource config for a catalog.

  Updates the datasource ingestion configuration for a catalog.

  Arguments:
    NAME:
    UPDATE_MASK: The field mask must be a single string, with multiple fields separated by
      commas (no spaces). The field path is relative to the resource object,
      using a dot (.) to navigate sub-fields (e.g., author.given_name).
      Specification of elements in sequence or map fields is not allowed, as
      only the entire collection field can be specified. Field names must
      exactly match the resource field names.

      A field mask of * indicates full replacement. It’s recommended to
      always explicitly list the fields being updated and avoid using *
      wildcards, as it can lead to unintended results if the API changes in the
      future.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateDatasourceConfigJson.Unmarshal(&updateDatasourceConfigReq.DatasourceConfig)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateDatasourceConfigReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateDatasourceConfigReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		response, err := w.Lakewatch.UpdateDatasourceConfig(ctx, updateDatasourceConfigReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateDatasourceConfigOverrides {
		fn(cmd, &updateDatasourceConfigReq)
	}

	return cmd
}

// start update-rule command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateRuleOverrides []func(
	*cobra.Command,
	*lakewatch.UpdateRuleRequest,
)

func newUpdateRule() *cobra.Command {
	cmd := &cobra.Command{}

	var updateRuleReq lakewatch.UpdateRuleRequest
	updateRuleReq.Rule = lakewatch.Rule{}
	var updateRuleJson flags.JsonFlag

	cmd.Flags().Var(&updateRuleJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateRuleReq.Etag, "etag", updateRuleReq.Etag, `Optimistic-concurrency token from the latest read.`)
	// TODO: map via StringToStringVar: annotations
	// TODO: complex arg: batch
	cmd.Flags().StringVar(&updateRuleReq.Rule.Category, "category", updateRuleReq.Rule.Category, `Free-form rule category.`)
	cmd.Flags().StringVar(&updateRuleReq.Rule.ContentResourceId, "content-resource-id", updateRuleReq.Rule.ContentResourceId, `Immutable UUIDv4 identifying the authored content resource installed as this rule.`)
	// TODO: complex arg: created_by
	cmd.Flags().Var(&updateRuleReq.Rule.Deduplication, "deduplication", `Signal deduplication mode. Supported values: [DEDUPLICATION_DEFAULT, DEDUPLICATION_DISABLED, DEDUPLICATION_ENABLED]`)
	cmd.Flags().StringVar(&updateRuleReq.Rule.DeduplicationTimeWindow, "deduplication-time-window", updateRuleReq.Rule.DeduplicationTimeWindow, `Time window for deduplication, such as 1h, 24h, 7d, or automatic.`)
	cmd.Flags().StringVar(&updateRuleReq.Rule.Description, "description", updateRuleReq.Rule.Description, `Free-form description.`)
	// TODO: array: mitre
	cmd.Flags().StringVar(&updateRuleReq.Rule.Name, "name", updateRuleReq.Rule.Name, `Resource name, in the form rules/{rule_id}.`)
	cmd.Flags().StringVar(&updateRuleReq.Rule.Objective, "objective", updateRuleReq.Rule.Objective, `Long-form description of what this rule detects.`)
	// TODO: complex arg: original_content_pack
	// TODO: complex arg: owner
	// TODO: array: references
	cmd.Flags().StringVar(&updateRuleReq.Rule.RunAsPrincipal, "run-as-principal", updateRuleReq.Rule.RunAsPrincipal, `The principal whose identity this rule's job runs as, given as a user email or service-principal application ID.`)
	cmd.Flags().StringVar(&updateRuleReq.Rule.Runbook, "runbook", updateRuleReq.Rule.Runbook, `Long-form operational guidance carried from the authored rule metadata.`)
	// TODO: complex arg: schedule
	// TODO: complex arg: status
	// TODO: complex arg: streaming
	var suppressUntilTimeParam string
	cmd.Flags().StringVar(&suppressUntilTimeParam, "suppress-until-time", suppressUntilTimeParam, `If set, this rule's signals do not open new alerts until the given time.`)
	// TODO: complex arg: timerange
	// TODO: complex arg: updated_by

	cmd.Use = "update-rule NAME UPDATE_MASK CATALOG_NAME SCHEMA_NAME RULE_NAME SEVERITY FIDELITY OUTPUT"
	cmd.Short = `*Beta* Update a rule.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a rule.

  Updates an existing rule.

  Arguments:
    NAME: Resource name, in the form rules/{rule_id}. Assigned by the server on
      create; required on update.
    UPDATE_MASK: The set of fields to update. Only fields named in the mask are modified.
    CATALOG_NAME: Name of the Unity Catalog catalog that contains the rule. Required. Cannot
      be changed after creation. Reads return the value as supplied; the
      catalog's current name is reported in effective_catalog_name.
    SCHEMA_NAME: Name of the Unity Catalog schema that contains the rule. Required. Cannot
      be changed after creation. Reads return the value as supplied; the
      schema's current name is reported in effective_schema_name.
    RULE_NAME: Name of the rule within its Unity Catalog schema. Required on create; can
      be updated within the same catalog and schema. Get synchronizes this field
      with the current Unity Catalog name, also reported in
      effective_rule_name.
    SEVERITY: Free-form severity label, such as Informational, Medium, High, or
      Critical.
    FIDELITY: Signal quality for this rule.
      Supported values: [FIDELITY_DEVELOPMENT, FIDELITY_HIGH, FIDELITY_INVESTIGATIVE, FIDELITY_MEDIUM]
    OUTPUT: What to record in each signal.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'catalog_name', 'schema_name', 'rule_name', 'severity', 'fidelity', 'output' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(8)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateRuleJson.Unmarshal(&updateRuleReq.Rule)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateRuleReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateRuleReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			updateRuleReq.Rule.CatalogName = args[2]
		}
		if !cmd.Flags().Changed("json") {
			updateRuleReq.Rule.SchemaName = args[3]
		}
		if !cmd.Flags().Changed("json") {
			updateRuleReq.Rule.RuleName = args[4]
		}
		if !cmd.Flags().Changed("json") {
			updateRuleReq.Rule.Severity = args[5]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[6], &updateRuleReq.Rule.Fidelity)
			if err != nil {
				return fmt.Errorf("invalid FIDELITY: %s", args[6])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[7], &updateRuleReq.Rule.Output)
			if err != nil {
				return fmt.Errorf("invalid OUTPUT: %s", args[7])
			}

		}

		if suppressUntilTimeParam != "" {
			suppressUntilTimeBytes := []byte(fmt.Sprintf("\"%s\"", suppressUntilTimeParam))
			var suppressUntilTimeField sdktime.Time
			err = json.Unmarshal(suppressUntilTimeBytes, &suppressUntilTimeField)
			if err != nil {
				return fmt.Errorf("invalid SUPPRESS_UNTIL_TIME: %s", suppressUntilTimeParam)
			}
			updateRuleReq.Rule.SuppressUntilTime = &suppressUntilTimeField
		}

		response, err := w.Lakewatch.UpdateRule(ctx, updateRuleReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateRuleOverrides {
		fn(cmd, &updateRuleReq)
	}

	return cmd
}

// start update-rule-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateRuleConfigOverrides []func(
	*cobra.Command,
	*lakewatch.UpdateRuleConfigRequest,
)

func newUpdateRuleConfig() *cobra.Command {
	cmd := &cobra.Command{}

	var updateRuleConfigReq lakewatch.UpdateRuleConfigRequest
	updateRuleConfigReq.RuleConfig = lakewatch.RuleConfig{}
	var updateRuleConfigJson flags.JsonFlag

	cmd.Flags().Var(&updateRuleConfigJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateRuleConfigReq.Etag, "etag", updateRuleConfigReq.Etag, `Optimistic-concurrency precondition: if set and it does not match the resource's current etag, the update is rejected as a conflict (HTTP 409).`)
	cmd.Flags().StringVar(&updateRuleConfigReq.RuleConfig.DefaultFolder, "default-folder", updateRuleConfigReq.RuleConfig.DefaultFolder, `Schema name of the registered folder (in this catalog) that a new rule is placed in when the request does not specify a schema.`)
	// TODO: complex arg: default_schedule
	cmd.Flags().StringVar(&updateRuleConfigReq.RuleConfig.DefaultSignalCollationWindow, "default-signal-collation-window", updateRuleConfigReq.RuleConfig.DefaultSignalCollationWindow, `Default duration string for signal deduplication when a rule doesn't override it — compact day/hour/minute/second components, e.g.`)
	// TODO: array: detection_categories
	// TODO: array: effective_detection_categories
	cmd.Flags().StringVar(&updateRuleConfigReq.RuleConfig.Name, "name", updateRuleConfigReq.RuleConfig.Name, ``)
	// TODO: complex arg: updated_by

	cmd.Use = "update-rule-config NAME UPDATE_MASK"
	cmd.Short = `*Beta* Update the rule config for a catalog.`
	cmd.Long = `This command is in Beta and may change without notice.

Update the rule config for a catalog.

  Updates the detection-rule configuration for a catalog.

  Arguments:
    NAME:
    UPDATE_MASK: The field mask must be a single string, with multiple fields separated by
      commas (no spaces). The field path is relative to the resource object,
      using a dot (.) to navigate sub-fields (e.g., author.given_name).
      Specification of elements in sequence or map fields is not allowed, as
      only the entire collection field can be specified. Field names must
      exactly match the resource field names.

      A field mask of * indicates full replacement. It’s recommended to
      always explicitly list the fields being updated and avoid using *
      wildcards, as it can lead to unintended results if the API changes in the
      future.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateRuleConfigJson.Unmarshal(&updateRuleConfigReq.RuleConfig)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateRuleConfigReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateRuleConfigReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		response, err := w.Lakewatch.UpdateRuleConfig(ctx, updateRuleConfigReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateRuleConfigOverrides {
		fn(cmd, &updateRuleConfigReq)
	}

	return cmd
}

// end service Lakewatch
