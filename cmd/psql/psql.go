package psql

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/databricks/cli/libs/cmdio"
	lakebasepsql "github.com/databricks/cli/libs/lakebase/psql"
	lakebasev2 "github.com/databricks/cli/libs/lakebase/v2"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	return newLakebaseConnectCommand()
}

func newLakebaseConnectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "psql [TARGET] [flags] [-- PSQL_ARGS...]",
		Short:   "Connect to a Lakebase Postgres database",
		GroupID: "database",
		Long: `Connect to a Lakebase Postgres database.

Requires psql to be installed.

Examples:

  # Interactive selection (shows all available databases)
  databricks psql
  databricks psql --provisioned
  databricks psql --autoscaling

  # Lakebase Provisioned
  databricks psql my-instance

  # Lakebase Autoscaling (auto-selects branch/endpoint if only one exists)
  databricks psql projects/my-project/branches/main/endpoints/primary
  databricks psql --project my-project
  databricks psql --project my-project --branch main --endpoint primary

  # Pass additional arguments to psql
  databricks psql my-instance -- -c "SELECT 1"
  databricks psql --project my-project -- -d mydb

For more information, see: https://docs.databricks.com/aws/en/oltp/
`,
	}

	var maxRetries int
	var provisionedFlag, autoscalingFlag bool
	var projectFlag, branchFlag, endpointFlag string

	cmd.Flags().IntVar(&maxRetries, "max-retries", 3, "Connection retries; 0 to disable")

	productGroup := cmdgroup.NewFlagGroup("Product Selection")
	productGroup.FlagSet().SortFlags = false
	productGroup.FlagSet().BoolVar(&provisionedFlag, "provisioned", false, "Only show Lakebase Provisioned instances")
	productGroup.FlagSet().BoolVar(&autoscalingFlag, "autoscaling", false, "Only show Lakebase Autoscaling projects")

	autoscalingGroup := cmdgroup.NewFlagGroup("Autoscaling")
	autoscalingGroup.FlagSet().SortFlags = false
	autoscalingGroup.FlagSet().StringVar(&projectFlag, "project", "", "Project ID")
	autoscalingGroup.FlagSet().StringVar(&branchFlag, "branch", "", "Branch ID (default: auto-select)")
	autoscalingGroup.FlagSet().StringVar(&endpointFlag, "endpoint", "", "Endpoint ID (default: auto-select)")

	wrappedCmd := cmdgroup.NewCommandWithGroupFlag(cmd)
	wrappedCmd.AddFlagGroup(productGroup)
	wrappedCmd.AddFlagGroup(autoscalingGroup)

	cmd.MarkFlagsMutuallyExclusive("provisioned", "autoscaling")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Split args at --
		argsLenAtDash := cmd.ArgsLenAtDash()
		if argsLenAtDash < 0 {
			argsLenAtDash = len(args)
		}
		target := ""
		if argsLenAtDash == 1 {
			target = args[0]
		} else if argsLenAtDash > 1 {
			return errors.New("expected at most one positional argument for target")
		}
		extraArgs := args[argsLenAtDash:]

		retryConfig := lakebasepsql.RetryConfig{
			MaxRetries:    maxRetries,
			InitialDelay:  time.Second,
			MaxDelay:      10 * time.Second,
			BackoffFactor: 2.0,
		}

		hasAutoscalingFlags := projectFlag != "" || branchFlag != "" || endpointFlag != ""

		// Check for conflicting flags
		if provisionedFlag && hasAutoscalingFlags {
			return errors.New("cannot use --project, --branch, or --endpoint flags with --provisioned")
		}

		// Positional argument takes precedence
		if target != "" {
			if strings.HasPrefix(target, "projects/") {
				if provisionedFlag {
					return errors.New("cannot use --provisioned flag with an autoscaling resource path")
				}

				projectID, branchID, endpointID, err := parseResourcePath(target)
				if err != nil {
					return err
				}

				// Check for conflicts between path and flags
				if projectFlag != "" && projectFlag != projectID {
					return fmt.Errorf("--project flag conflicts with project in path: %s vs %s", projectFlag, projectID)
				}
				if branchFlag != "" && branchID != "" && branchFlag != branchID {
					return fmt.Errorf("--branch flag conflicts with branch in path: %s vs %s", branchFlag, branchID)
				}
				if endpointFlag != "" && endpointID != "" && endpointFlag != endpointID {
					return fmt.Errorf("--endpoint flag conflicts with endpoint in path: %s vs %s", endpointFlag, endpointID)
				}

				// Flags supplement missing components
				if branchID == "" {
					branchID = branchFlag
				}
				if endpointID == "" {
					endpointID = endpointFlag
				}

				return connectAutoscaling(ctx, projectID, branchID, endpointID, retryConfig, extraArgs)
			}

			// Provisioned instance name - cannot mix with autoscaling flags
			if hasAutoscalingFlags {
				return errors.New("cannot use --project, --branch, or --endpoint flags with a provisioned instance name")
			}
			if autoscalingFlag {
				return errors.New("cannot use --autoscaling flag with a provisioned instance name")
			}
			return connectProvisioned(ctx, target, retryConfig, extraArgs)
		}

		// No positional argument - use flags only
		if hasAutoscalingFlags {
			if projectFlag == "" {
				return errors.New("--project is required when using --branch or --endpoint")
			}
			return connectAutoscaling(ctx, projectFlag, branchFlag, endpointFlag, retryConfig, extraArgs)
		}

		// Product-specific interactive selection
		if provisionedFlag {
			return connectProvisioned(ctx, "", retryConfig, extraArgs)
		}
		if autoscalingFlag {
			return connectAutoscaling(ctx, "", "", "", retryConfig, extraArgs)
		}

		// No args, no flags -> interactive selection
		return showSelectionAndConnect(ctx, retryConfig, extraArgs)
	}

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		err := root.MustWorkspaceClient(cmd, args)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		instances, projects := listAllDatabases(ctx, w)

		var names []string
		for _, inst := range instances {
			names = append(names, inst.Name)
		}
		for _, proj := range projects {
			names = append(names, proj.Name)
		}

		return names, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

// parseResourcePath extracts project, branch, and endpoint IDs from a resource path.
// Returns an error for malformed paths.
func parseResourcePath(input string) (project, branch, endpoint string, err error) {
	parts := strings.Split(input, "/")

	// Must start with projects/{project_id}
	if len(parts) < 2 || parts[0] != "projects" {
		return "", "", "", fmt.Errorf("invalid resource path: %s", input)
	}
	if parts[1] == "" {
		return "", "", "", errors.New("invalid resource path: missing project ID")
	}
	project = parts[1]

	// Optional: branches/{branch_id}
	if len(parts) > 2 {
		if len(parts) < 4 || parts[2] != "branches" {
			return "", "", "", errors.New("invalid resource path: expected 'branches' after project")
		}
		if parts[3] == "" {
			return "", "", "", errors.New("invalid resource path: missing branch ID")
		}
		branch = parts[3]
	}

	// Optional: endpoints/{endpoint_id}
	if len(parts) > 4 {
		if len(parts) < 6 || parts[4] != "endpoints" {
			return "", "", "", errors.New("invalid resource path: expected 'endpoints' after branch")
		}
		if parts[5] == "" {
			return "", "", "", errors.New("invalid resource path: missing endpoint ID")
		}
		endpoint = parts[5]
	}

	return project, branch, endpoint, nil
}

// listAllDatabases fetches all database instances and projects in parallel.
// Errors are silently ignored; callers should check for empty results.
func listAllDatabases(ctx context.Context, w *databricks.WorkspaceClient) ([]database.DatabaseInstance, []postgres.Project) {
	type result[T any] struct {
		value []T
		err   error
	}

	instancesCh := make(chan result[database.DatabaseInstance], 1)
	projectsCh := make(chan result[postgres.Project], 1)

	go func() {
		instances, err := w.Database.ListDatabaseInstancesAll(ctx, database.ListDatabaseInstancesRequest{})
		instancesCh <- result[database.DatabaseInstance]{instances, err}
	}()

	go func() {
		projects, err := w.Postgres.ListProjectsAll(ctx, postgres.ListProjectsRequest{})
		projectsCh <- result[postgres.Project]{projects, err}
	}()

	instResult := <-instancesCh
	projResult := <-projectsCh

	var instances []database.DatabaseInstance
	var projects []postgres.Project
	if instResult.err == nil {
		instances = instResult.value
	}
	if projResult.err == nil {
		projects = projResult.value
	}

	return instances, projects
}

// showSelectionAndConnect shows a combined dropdown of Lakebase databases.
func showSelectionAndConnect(ctx context.Context, retryConfig lakebasepsql.RetryConfig, extraArgs []string) error {
	w := cmdctx.WorkspaceClient(ctx)

	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading Lakebase databases...")
	instances, projects := listAllDatabases(ctx, w)
	sp.Close()

	// Build selection list with connect functions
	type selectable struct {
		label   string
		connect func() error
	}
	var options []selectable

	for _, inst := range instances {
		options = append(options, selectable{
			label: inst.Name + " (provisioned)",
			connect: func() error {
				return connectProvisioned(ctx, inst.Name, retryConfig, extraArgs)
			},
		})
	}

	for _, proj := range projects {
		projectID := lakebasev2.ExtractIDFromName(proj.Name, "projects")
		displayName := projectID
		if proj.Status != nil && proj.Status.DisplayName != "" {
			displayName = proj.Status.DisplayName
		}
		options = append(options, selectable{
			label: displayName + " (autoscaling)",
			connect: func() error {
				return connectAutoscaling(ctx, projectID, "", "", retryConfig, extraArgs)
			},
		})
	}

	if len(options) == 0 {
		return errors.New("no Lakebase databases found in workspace")
	}

	// Build selection items
	var items []cmdio.Tuple
	for i, opt := range options {
		items = append(items, cmdio.Tuple{Name: opt.label, Id: strconv.Itoa(i)})
	}

	selected, err := cmdio.SelectOrdered(ctx, items, "Select database to connect to")
	if err != nil {
		return err
	}

	idx, err := strconv.Atoi(selected)
	if err != nil || idx < 0 || idx >= len(options) {
		return fmt.Errorf("unexpected selection: %s", selected)
	}

	return options[idx].connect()
}
