// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package compute_migrations

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/computemigrations"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

// Spec declares the command for this service.
var Spec = root.CommandSpec{
	Use:     "compute-migrations",
	Short:   `Manage migrations of Classic compute workloads to Serverless.`,
	GroupID: "compute",

	// This service is being previewed; hide from help output.
	Hidden: true,
	Build:  buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `Manage migrations of Classic compute workloads to Serverless.`
	cmd.RunE = root.ReportUnknownSubcommand

	cmd.Annotations = map[string]string{
		"launch_stage":         "PRIVATE_PREVIEW",
		"launch_stage_display": "Private Preview",
	}

	// Add methods
	cmd.AddCommand(newApproveMigrationExecution())
	cmd.AddCommand(newApproveMigrationTranslation())
	cmd.AddCommand(newCancelMigration())
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newRejectMigrationExecution())
	cmd.AddCommand(newRejectMigrationTranslation())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}
}

// start approve-migration-execution command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var approveMigrationExecutionOverrides []func(
	*cobra.Command,
	*computemigrations.ApproveMigrationExecutionRequest,
)

func newApproveMigrationExecution() *cobra.Command {
	cmd := &cobra.Command{}

	var approveMigrationExecutionReq computemigrations.ApproveMigrationExecutionRequest
	var approveMigrationExecutionJson flags.JsonFlag

	cmd.Flags().Var(&approveMigrationExecutionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&approveMigrationExecutionReq.AdditionalFeedback, "additional-feedback", approveMigrationExecutionReq.AdditionalFeedback, `Optional free-text feedback or instructions the user provides along with the approval, carried into the next stage of the migration.`)
	cmd.Flags().StringVar(&approveMigrationExecutionReq.Etag, "etag", approveMigrationExecutionReq.Etag, `Optional resource version for optimistic concurrency control.`)

	cmd.Use = "approve-migration-execution MIGRATION_ID"
	cmd.Short = `Approve a migration for execution.`
	cmd.Long = `Approve a migration for execution.

  Approves migration execution at the MIGRATION_REVIEW gate, advancing to
  execution.

  Valid only when the migration is in MIGRATION_REVIEW. If supplied, the etag
  must match the current server value.

  Returns NOT_FOUND if the migration does not exist. Returns FAILED_PRECONDITION
  if the migration is not in MIGRATION_REVIEW. Returns ABORTED if the etag does
  not match the current resource version.

  Arguments:
    MIGRATION_ID: The ID of the migration to approve for execution.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := approveMigrationExecutionJson.Unmarshal(&approveMigrationExecutionReq)
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
		approveMigrationExecutionReq.MigrationId = args[0]

		response, err := w.ComputeMigrations.ApproveMigrationExecution(ctx, approveMigrationExecutionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range approveMigrationExecutionOverrides {
		fn(cmd, &approveMigrationExecutionReq)
	}

	return cmd
}

// start approve-migration-translation command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var approveMigrationTranslationOverrides []func(
	*cobra.Command,
	*computemigrations.ApproveMigrationTranslationRequest,
)

func newApproveMigrationTranslation() *cobra.Command {
	cmd := &cobra.Command{}

	var approveMigrationTranslationReq computemigrations.ApproveMigrationTranslationRequest
	var approveMigrationTranslationJson flags.JsonFlag

	cmd.Flags().Var(&approveMigrationTranslationJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&approveMigrationTranslationReq.AdditionalFeedback, "additional-feedback", approveMigrationTranslationReq.AdditionalFeedback, `Optional free-text feedback or instructions the user provides along with the approval, carried into the next stage of the migration.`)
	cmd.Flags().StringVar(&approveMigrationTranslationReq.Etag, "etag", approveMigrationTranslationReq.Etag, `Optional resource version for optimistic concurrency control.`)

	cmd.Use = "approve-migration-translation MIGRATION_ID"
	cmd.Short = `Approve a migration working copy.`
	cmd.Long = `Approve a migration working copy.

  Approves the working copy at the TRANSLATION_READY gate. If the migration was
  created with run_validation, the working copy is then run to verify it
  executes successfully before the migration is presented for execution review;
  otherwise it advances straight to MIGRATION_REVIEW.

  Valid only when the migration is in TRANSLATION_READY. If supplied, the etag
  must match the current server value.

  Returns NOT_FOUND if the migration does not exist. Returns FAILED_PRECONDITION
  if the migration is not in TRANSLATION_READY. Returns ABORTED if the etag does
  not match the current resource version.

  Arguments:
    MIGRATION_ID: The ID of the migration to approve.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := approveMigrationTranslationJson.Unmarshal(&approveMigrationTranslationReq)
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
		approveMigrationTranslationReq.MigrationId = args[0]

		response, err := w.ComputeMigrations.ApproveMigrationTranslation(ctx, approveMigrationTranslationReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range approveMigrationTranslationOverrides {
		fn(cmd, &approveMigrationTranslationReq)
	}

	return cmd
}

// start cancel-migration command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cancelMigrationOverrides []func(
	*cobra.Command,
	*computemigrations.CancelMigrationRequest,
)

func newCancelMigration() *cobra.Command {
	cmd := &cobra.Command{}

	var cancelMigrationReq computemigrations.CancelMigrationRequest
	var cancelMigrationJson flags.JsonFlag

	cmd.Flags().Var(&cancelMigrationJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&cancelMigrationReq.CancellationReason, "cancellation-reason", cancelMigrationReq.CancellationReason, `Optional free-text reason for the cancellation.`)
	cmd.Flags().StringVar(&cancelMigrationReq.Etag, "etag", cancelMigrationReq.Etag, `Optional resource version for optimistic concurrency control.`)

	cmd.Use = "cancel-migration MIGRATION_ID"
	cmd.Short = `Cancel a migration.`
	cmd.Long = `Cancel a migration.

  Cancels the migration, transitioning it to CANCELLED.

  Valid from any non-terminal state, unlike the review-gate transitions, which
  each require a specific state. Any in-flight agent work is abandoned and the
  migration stops advancing. The resource is retained for auditability.

  Returns NOT_FOUND if the migration does not exist. Returns FAILED_PRECONDITION
  if the migration is already in a terminal state. Returns ABORTED if the etag
  does not match the current resource version.

  Arguments:
    MIGRATION_ID: The ID of the migration to cancel.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := cancelMigrationJson.Unmarshal(&cancelMigrationReq)
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
		cancelMigrationReq.MigrationId = args[0]

		response, err := w.ComputeMigrations.CancelMigration(ctx, cancelMigrationReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range cancelMigrationOverrides {
		fn(cmd, &cancelMigrationReq)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*computemigrations.CreateMigrationRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq computemigrations.CreateMigrationRequest
	createReq.Migration = computemigrations.Migration{}
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.RequestId, "request-id", createReq.RequestId, `Client-generated request ID for idempotent request deduplication.`)
	cmd.Flags().BoolVar(&createReq.Migration.AutoApprove, "auto-approve", createReq.Migration.AutoApprove, `When true, the server skips the translation review (TRANSLATION_READY) instead of waiting for an explicit ApproveMigrationTranslation.`)
	// TODO: complex arg: baking_config
	// TODO: complex arg: baking_report
	// TODO: complex arg: initiator
	cmd.Flags().StringVar(&createReq.Migration.MigrationId, "migration-id", createReq.Migration.MigrationId, `Server-generated unique identifier for this migration.`)
	// TODO: complex arg: migration_report
	cmd.Flags().BoolVar(&createReq.Migration.RunValidation, "run-validation", createReq.Migration.RunValidation, `When true, Databricks runs the working copy to validate it after the translation review.`)
	// TODO: array: state_history
	// TODO: complex arg: validation_report

	cmd.Use = "create ASSET_TYPE ASSET_URI"
	cmd.Short = `Create a migration.`
	cmd.Long = `Create a migration.

  Creates a new migration for a Classic compute asset.

  The migration is created in PLANNING_PENDING and advances through the state
  machine as planning work is scheduled. The caller must have appropriate
  permissions: workspace admin for infrastructure-scoped migrations, job
  owner/admin for individual workload migrations.

  At most one migration may hold an asset. Returns ALREADY_EXISTS if the asset
  has a migration in flight, or one that already migrated it successfully; a
  migration that ended without succeeding, or was deleted, releases the asset so
  it can be migrated again. Returns INVALID_PARAMETER_VALUE if the asset is not
  eligible for migration.

  Arguments:
    ASSET_TYPE: Type of Classic compute asset being migrated. Cannot be changed after
      creation.
      Supported values: [FILE, INTERACTIVE, JOB]
    ASSET_URI: URI of the source asset (e.g., "jobs/12345"). Cannot be changed after
      creation.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'asset_type', 'asset_uri' in your JSON input")
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
			diags := createJson.Unmarshal(&createReq.Migration)
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
			_, err = fmt.Sscan(args[0], &createReq.Migration.AssetType)
			if err != nil {
				return fmt.Errorf("invalid ASSET_TYPE: %s", args[0])
			}

		}
		if !cmd.Flags().Changed("json") {
			createReq.Migration.AssetUri = args[1]
		}

		response, err := w.ComputeMigrations.Create(ctx, createReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOverrides {
		fn(cmd, &createReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*computemigrations.DeleteMigrationRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq computemigrations.DeleteMigrationRequest

	cmd.Flags().StringVar(&deleteReq.Etag, "etag", deleteReq.Etag, `If nonempty, must match the current resource etag or the request fails with ABORTED.`)

	cmd.Use = "delete MIGRATION_ID"
	cmd.Short = `Delete a migration.`
	cmd.Long = `Delete a migration.

  Deletes a migration (soft-delete).

  Allowed from any state. Deleting a migration marks the record deleted only —
  it does not roll back a migration that already ran, and resources created
  during the migration are not cleaned up. The migration resource is retained
  for auditability. To stop an in-flight migration (agent teardown, transition
  to CANCELLED), use CancelMigration instead.

  Returns NOT_FOUND if the migration does not exist, or was already deleted: a
  deleted migration is hidden from every method (Get, Update, List, and Delete),
  so it cannot be deleted again. Returns ABORTED if a supplied nonempty request
  etag does not match the resource version.

  Arguments:
    MIGRATION_ID: The ID of the migration to cancel.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.MigrationId = args[0]

		response, err := w.ComputeMigrations.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteOverrides {
		fn(cmd, &deleteReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*computemigrations.GetMigrationRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq computemigrations.GetMigrationRequest

	cmd.Use = "get MIGRATION_ID"
	cmd.Short = `Get a migration.`
	cmd.Long = `Get a migration.

  Retrieves a migration by its ID.

  Returns the current state, configuration, and output artifacts of the
  migration. Use this to poll migration progress or inspect artifacts after
  state transitions. Returns NOT_FOUND if the migration does not exist in the
  caller's workspace.

  Arguments:
    MIGRATION_ID: The ID of the migration to retrieve.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.MigrationId = args[0]

		response, err := w.ComputeMigrations.Get(ctx, getReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOverrides {
		fn(cmd, &getReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*computemigrations.ListMigrationsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq computemigrations.ListMigrationsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	cmd.Flags().Var(&listReq.AssetType, "asset-type", `Filter by asset type. Supported values: [FILE, INTERACTIVE, JOB]`)
	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `Maximum number of migrations to return.`)
	// TODO: array: states

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List migrations.`
	cmd.Long = `List migrations.

  Lists migrations in the caller's workspace, ordered by migration ID, with
  cursor-based pagination and filtering by asset type and state.

  Eventually consistent: results are read from a cache, and pagination is not a
  point-in-time snapshot, so a migration created during a List may not appear
  until a subsequent one.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.ComputeMigrations.List(ctx, listReq)
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd, &listReq)
	}

	return cmd
}

// start reject-migration-execution command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var rejectMigrationExecutionOverrides []func(
	*cobra.Command,
	*computemigrations.RejectMigrationExecutionRequest,
)

func newRejectMigrationExecution() *cobra.Command {
	cmd := &cobra.Command{}

	var rejectMigrationExecutionReq computemigrations.RejectMigrationExecutionRequest
	var rejectMigrationExecutionJson flags.JsonFlag

	cmd.Flags().Var(&rejectMigrationExecutionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&rejectMigrationExecutionReq.Etag, "etag", rejectMigrationExecutionReq.Etag, `Optional resource version for optimistic concurrency control.`)

	cmd.Use = "reject-migration-execution MIGRATION_ID REJECTION_REASON"
	cmd.Short = `Reject a migration execution.`
	cmd.Long = `Reject a migration execution.

  Rejects migration execution at the MIGRATION_REVIEW gate, sending the
  migration to REVISION_PENDING to loop back for re-planning.

  Valid only when the migration is in MIGRATION_REVIEW. If supplied, the etag
  must match the current server value.

  Returns NOT_FOUND if the migration does not exist. Returns FAILED_PRECONDITION
  if the migration is not in MIGRATION_REVIEW. Returns ABORTED if the etag does
  not match the current resource version.

  Arguments:
    MIGRATION_ID: The ID of the migration to reject.
    REJECTION_REASON: Required free-text reason for the rejection, fed back into re-planning so
      the next plan can address the user's concerns.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only MIGRATION_ID as positional arguments. Provide 'rejection_reason' in your JSON input")
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
			diags := rejectMigrationExecutionJson.Unmarshal(&rejectMigrationExecutionReq)
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
		rejectMigrationExecutionReq.MigrationId = args[0]
		if !cmd.Flags().Changed("json") {
			rejectMigrationExecutionReq.RejectionReason = args[1]
		}

		response, err := w.ComputeMigrations.RejectMigrationExecution(ctx, rejectMigrationExecutionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range rejectMigrationExecutionOverrides {
		fn(cmd, &rejectMigrationExecutionReq)
	}

	return cmd
}

// start reject-migration-translation command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var rejectMigrationTranslationOverrides []func(
	*cobra.Command,
	*computemigrations.RejectMigrationTranslationRequest,
)

func newRejectMigrationTranslation() *cobra.Command {
	cmd := &cobra.Command{}

	var rejectMigrationTranslationReq computemigrations.RejectMigrationTranslationRequest
	var rejectMigrationTranslationJson flags.JsonFlag

	cmd.Flags().Var(&rejectMigrationTranslationJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&rejectMigrationTranslationReq.Etag, "etag", rejectMigrationTranslationReq.Etag, `Optional resource version for optimistic concurrency control.`)

	cmd.Use = "reject-migration-translation MIGRATION_ID REJECTION_REASON"
	cmd.Short = `Reject a migration working copy.`
	cmd.Long = `Reject a migration working copy.

  Rejects the working copy at the TRANSLATION_READY gate, sending the migration
  to REVISION_PENDING to loop back for re-planning.

  Valid only when the migration is in TRANSLATION_READY. If supplied, the etag
  must match the current server value.

  Returns NOT_FOUND if the migration does not exist. Returns FAILED_PRECONDITION
  if the migration is not in TRANSLATION_READY. Returns ABORTED if the etag does
  not match the current resource version.

  Arguments:
    MIGRATION_ID: The ID of the migration to reject.
    REJECTION_REASON: Required free-text reason for the rejection, fed back into re-planning so
      the next plan can address the user's concerns.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only MIGRATION_ID as positional arguments. Provide 'rejection_reason' in your JSON input")
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
			diags := rejectMigrationTranslationJson.Unmarshal(&rejectMigrationTranslationReq)
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
		rejectMigrationTranslationReq.MigrationId = args[0]
		if !cmd.Flags().Changed("json") {
			rejectMigrationTranslationReq.RejectionReason = args[1]
		}

		response, err := w.ComputeMigrations.RejectMigrationTranslation(ctx, rejectMigrationTranslationReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range rejectMigrationTranslationOverrides {
		fn(cmd, &rejectMigrationTranslationReq)
	}

	return cmd
}

// end service ComputeMigrations
