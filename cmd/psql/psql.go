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
	"github.com/databricks/cli/libs/cmdio"
	lakebasepsql "github.com/databricks/cli/libs/lakebase/psql"
	lakebasev1 "github.com/databricks/cli/libs/lakebase/v1"
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
		Use:     "psql [TARGET] [-- PSQL_ARGS...]",
		Short:   "Connect to a Lakebase Postgres database",
		GroupID: "database",
		Long: `Connect to a Lakebase Postgres database.

This command requires a psql client to be installed on your machine for the connection to work.

The command includes automatic retry logic for connection failures. You can configure the retry behavior using the flags below.

Usage modes:

1. Lakebase Provisioned (database instances):
   databricks psql my-database-instance

2. Lakebase Autoscaling with full endpoint path:
   databricks psql projects/my-project/branches/main/endpoints/primary

3. Lakebase Autoscaling using flags (prompts for branch/endpoint if multiple exist):
   databricks psql --project my-project
   databricks psql --project my-project --branch main
   databricks psql --project my-project --branch main --endpoint primary

4. Interactive selection (shows dropdown with all available databases):
   databricks psql

You can pass additional arguments to psql after a double-dash (--):
  databricks psql my-database -- -c "SELECT * FROM my_table"
  databricks psql --project my-project -- --echo-all -d "my-db"
`,
	}

	// Add retry configuration flag
	cmd.Flags().Int("max-retries", 3, "Maximum number of connection retry attempts (set to 0 to disable retries)")

	// Add Lakebase Autoscaling flags
	cmd.Flags().String("project", "", "Lakebase Autoscaling project ID")
	cmd.Flags().String("branch", "", "Lakebase Autoscaling branch ID (default: auto-select)")
	cmd.Flags().String("endpoint", "", "Lakebase Autoscaling endpoint ID (default: auto-select)")

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

		// Read flags
		projectFlag, _ := cmd.Flags().GetString("project")
		branchFlag, _ := cmd.Flags().GetString("branch")
		endpointFlag, _ := cmd.Flags().GetString("endpoint")
		maxRetries, _ := cmd.Flags().GetInt("max-retries")

		retryConfig := lakebasepsql.RetryConfig{
			MaxRetries:    maxRetries,
			InitialDelay:  time.Second,
			MaxDelay:      10 * time.Second,
			BackoffFactor: 2.0,
		}

		hasAutoscalingFlags := projectFlag != "" || branchFlag != "" || endpointFlag != ""

		// Positional argument takes precedence
		if target != "" {
			if strings.HasPrefix(target, "projects/") {
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
			return connectProvisioned(ctx, target, retryConfig, extraArgs)
		}

		// No positional argument - use flags only
		if hasAutoscalingFlags {
			if projectFlag == "" {
				return errors.New("--project is required when using --branch or --endpoint")
			}
			return connectAutoscaling(ctx, projectFlag, branchFlag, endpointFlag, retryConfig, extraArgs)
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

// connectProvisioned connects to a Lakebase Provisioned database instance.
func connectProvisioned(ctx context.Context, instanceName string, retryConfig lakebasepsql.RetryConfig, extraArgs []string) error {
	w := cmdctx.WorkspaceClient(ctx)

	db, err := lakebasev1.GetDatabaseInstance(ctx, w, instanceName)
	if err != nil {
		return err
	}

	cmdio.LogString(ctx, "Instance: "+db.Name+" (provisioned)")
	return lakebasev1.Connect(ctx, w, db, retryConfig, extraArgs...)
}

// connectAutoscaling connects to a Lakebase Autoscaling endpoint.
func connectAutoscaling(ctx context.Context, projectID, branchID, endpointID string, retryConfig lakebasepsql.RetryConfig, extraArgs []string) error {
	w := cmdctx.WorkspaceClient(ctx)

	endpoint, err := resolveEndpoint(ctx, w, projectID, branchID, endpointID)
	if err != nil {
		return err
	}

	return lakebasev2.Connect(ctx, w, endpoint, retryConfig, extraArgs...)
}

// resolveEndpoint resolves a partial specification to a full endpoint.
// Uses interactive selection when components are missing.
func resolveEndpoint(ctx context.Context, w *databricks.WorkspaceClient, projectID, branchID, endpointID string) (*postgres.Endpoint, error) {
	projectName := "projects/" + projectID

	// Get project to display its name
	project, err := w.Postgres.GetProject(ctx, postgres.GetProjectRequest{Name: projectName})
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	displayName := projectID
	if project.Status != nil && project.Status.DisplayName != "" {
		displayName = project.Status.DisplayName
	}
	cmdio.LogString(ctx, "Project: "+displayName)

	// If branch not specified, select one
	if branchID == "" {
		var err error
		branchID, err = selectBranchID(ctx, w, projectName)
		if err != nil {
			return nil, fmt.Errorf("failed to select branch: %w", err)
		}
	}
	cmdio.LogString(ctx, "Branch: "+branchID)

	// If endpoint not specified, select one
	if endpointID == "" {
		var err error
		endpointID, err = selectEndpointID(ctx, w, projectName+"/branches/"+branchID)
		if err != nil {
			return nil, fmt.Errorf("failed to select endpoint: %w", err)
		}
	}
	cmdio.LogString(ctx, "Endpoint: "+endpointID)

	return lakebasev2.GetEndpoint(ctx, w, projectID, branchID, endpointID)
}

// selectBranchID auto-selects if there's only one branch, otherwise prompts user to select.
// Returns the branch ID (not the full branch object).
func selectBranchID(ctx context.Context, w *databricks.WorkspaceClient, projectName string) (string, error) {
	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading branches...")
	branches, err := w.Postgres.ListBranchesAll(ctx, postgres.ListBranchesRequest{
		Parent: projectName,
	})
	sp.Close()
	if err != nil {
		return "", err
	}

	if len(branches) == 0 {
		return "", errors.New("no branches found in project")
	}

	// Auto-select if there's only one branch
	if len(branches) == 1 {
		return lakebasev2.ExtractIDFromName(branches[0].Name, "branches"), nil
	}

	// Multiple branches, prompt user to select
	var items []cmdio.Tuple
	for _, branch := range branches {
		branchID := lakebasev2.ExtractIDFromName(branch.Name, "branches")
		items = append(items, cmdio.Tuple{Name: branchID, Id: branchID})
	}

	return cmdio.SelectOrdered(ctx, items, "Select branch")
}

// selectEndpointID auto-selects if there's only one endpoint, otherwise prompts user to select.
// Returns the endpoint ID (not the full endpoint object).
func selectEndpointID(ctx context.Context, w *databricks.WorkspaceClient, branchName string) (string, error) {
	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading endpoints...")
	endpoints, err := w.Postgres.ListEndpointsAll(ctx, postgres.ListEndpointsRequest{
		Parent: branchName,
	})
	sp.Close()
	if err != nil {
		return "", err
	}

	if len(endpoints) == 0 {
		return "", errors.New("no endpoints found in branch")
	}

	// Auto-select if there's only one endpoint
	if len(endpoints) == 1 {
		return lakebasev2.ExtractIDFromName(endpoints[0].Name, "endpoints"), nil
	}

	// Multiple endpoints, prompt user to select
	var items []cmdio.Tuple
	for _, endpoint := range endpoints {
		endpointID := lakebasev2.ExtractIDFromName(endpoint.Name, "endpoints")
		items = append(items, cmdio.Tuple{Name: endpointID, Id: endpointID})
	}

	return cmdio.SelectOrdered(ctx, items, "Select endpoint")
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
		return errors.New("could not find any Lakebase databases in the workspace")
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
